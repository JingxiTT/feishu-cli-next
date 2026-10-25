#!/usr/bin/env python3
# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Codex JSONL adapter for real Feishu evaluation."""
import argparse
import functools
import gzip
import json
import os
from pathlib import Path
import shutil
import shlex
import signal
import subprocess
import sys


DISABLED_FEATURES = (
    "apps", "browser_use", "browser_use_external", "chronicle", "computer_use",
    "hooks", "image_generation", "in_app_browser", "memories", "multi_agent",
    "multi_agent_v2", "plugins", "remote_plugin", "skill_search", "tool_suggest",
    "workspace_dependencies", "shell_snapshot",
)

# Public errs/category.go and errs/subtypes.go vocabulary, deliberately finite.
# These fields report CLI output; they are never backend or success evidence.
CLI_ERROR_SUBTYPES = {
    "validation": {"invalid_argument", "failed_precondition", "command_unavailable"},
    "authentication": {"token_missing", "token_invalid", "token_expired", "refresh_token_invalid",
                       "refresh_token_expired", "refresh_token_revoked", "refresh_token_reused", "refresh_server_error"},
    "authorization": {"missing_scope", "user_unauthorized", "app_scope_not_applied", "token_scope_insufficient",
                      "app_unavailable", "app_disabled", "permission_denied"},
    "config": {"invalid_client", "not_configured", "invalid_config"},
    "network": {"transport", "timeout", "tls", "dns", "server_error", "protocol", "representation_changed"},
    "api": {"mutation_outcome", "rate_limit", "conflict", "cross_tenant", "cross_brand", "invalid_parameters",
            "ownership_mismatch", "not_found", "server_error", "quota_exceeded", "already_exists", "feature_not_available"},
    "policy": {"challenge_required", "access_denied", "content_safety"},
    "internal": {"sdk_error", "invalid_response", "file_io", "external_tool", "storage"},
    "confirmation": {"confirmation_required"},
}
PUSH_ERROR_REASONS = {
    "invalid document workspace sidecar": "workspace_sidecar_invalid",
    "invalid or unsupported document workspace sidecar": "workspace_sidecar_invalid",
    "local document edit is unsupported or ambiguous": "workspace_edit_unsupported",
    "current document differs from the expected native snapshot": "workspace_snapshot_changed",
    "native readback does not match the planned block edit": "workspace_readback_mismatch",
    "mutation evidence is incomplete; inspect receipts before continuing": "workspace_evidence_incomplete",
}
MUTATION_OUTCOMES = {"confirmed_unapplied", "partial", "unknown", "confirmed_applied"}


def sanitized_cli_error(value, command, *, wire=False, cause=False):
    """Rebuild only published enum fields; never copy arbitrary envelope content."""
    if not isinstance(value, dict):
        return None
    category, subtype = value.get("type"), value.get("subtype")
    if (not isinstance(category, str) or category not in CLI_ERROR_SUBTYPES or
            not isinstance(subtype, str) or subtype not in CLI_ERROR_SUBTYPES[category] | {"unknown"}):
        return None
    result = {"type": category, "subtype": subtype}
    param = value.get("param")
    if isinstance(param, str) and param in trace_vocabulary()[1]:
        result["param"] = param
    if command == ["docs", "+push"] and (category, subtype) == ("validation", "failed_precondition"):
        text = value.get("message") if wire else value.get("reason")
        reason = PUSH_ERROR_REASONS.get(text) if wire and isinstance(text, str) else text
        if isinstance(reason, str) and reason in PUSH_ERROR_REASONS.values():
            result["reason"] = reason
    if not cause and (category, subtype) == ("api", "mutation_outcome"):
        outcome = value.get("outcome")
        if isinstance(outcome, str) and outcome in MUTATION_OUTCOMES:
            result["outcome"] = outcome
        detail = sanitized_cli_error(value.get("cause"), command, wire=wire, cause=True)
        if detail:
            result["cause"] = detail
    return result


def reported_cli_error(output, choices, exit_code):
    """Parse only complete bounded JSON output from one failed CLI shell choice.

    Pretty JSON and a partial-result envelope followed by an error are supported.
    Mixed shell output, truncation and unknown shapes stay unavailable. No search
    inside arbitrary text; all parse failures are independent of token accounting.
    """
    if (type(exit_code) is not int or exit_code == 0 or len(choices) != 1 or
            choices[0].get("kind") != "cli" or not isinstance(output, str) or len(output) > 65536):
        return None

    def unique_object(pairs):
        value = {}
        for key, item in pairs:
            if key in value:
                raise ValueError("duplicate JSON key")
            value[key] = item
        return value

    def invalid_constant(_):
        raise ValueError("non-JSON constant")

    decoder = json.JSONDecoder(object_pairs_hook=unique_object, parse_constant=invalid_constant)
    remaining, errors = output.strip(), []
    try:
        for _ in range(2):
            envelope, end = decoder.raw_decode(remaining)
            if not isinstance(envelope, dict) or type(envelope.get("ok")) is not bool:
                return None
            if envelope["ok"] is False and isinstance(envelope.get("error"), dict):
                errors.append(envelope["error"])
            remaining = remaining[end:].strip()
            if not remaining:
                break
        if remaining or len(errors) != 1:
            return None
        return sanitized_cli_error(errors[0], choices[0].get("command"), wire=True)
    except (ValueError, TypeError, RecursionError):
        return None


@functools.lru_cache(maxsize=1)
def trace_vocabulary():
    """Only published command/flag names may appear in retained trace evidence."""
    snapshot = Path(__file__).resolve().parents[1] / "baseline/registrations.json.gz"
    with gzip.open(snapshot, "rt", encoding="utf-8") as stream:
        registrations = json.load(stream)
    paths = {("discover",), ("guide",), ("sheets", "+data-export"),
             ("docs", "+pull"), ("docs", "+push")} | {("guide", action) for action in ("show", "install", "update", "uninstall")}
    flags = {"--help", "-h", "--flags", "--project", "--result-detail", "--result-fields", "--save-result"}
    for variant in registrations.values():
        for command, descriptor in variant["commands"].items():
            paths.add(tuple(command.split()))
            for name, metadata in descriptor.get("flags", {}).items():
                flags.add("--" + name)
                if metadata.get("shorthand"):
                    flags.add("-" + metadata["shorthand"])
    return paths, flags


def command_choices(command, depth=0):
    """Conservative submitted-shell choices, never arguments, payloads, or paths."""
    if not isinstance(command, str) or depth > 2:
        return [{"kind": "unparsed_shell"}]
    try:
        lexer = shlex.shlex(command, posix=True, punctuation_chars=";&|()")
        lexer.whitespace_split = True
        tokens = list(lexer)
    except ValueError:
        return [{"kind": "unparsed_shell"}]
    segments = [[]]
    for token in tokens:
        if token and all(char in ";&|()" for char in token):
            segments.append([])
        else:
            segments[-1].append(token)
    paths, allowed_flags = trace_vocabulary()
    result = []
    for segment in segments:
        while segment and "=" in segment[0] and not segment[0].startswith("-"):
            segment = segment[1:]
        if not segment:
            continue
        executable = Path(segment[0]).name
        if executable in ("bash", "zsh", "sh") and len(segment) >= 3 and segment[1] in ("-c", "-lc"):
            result.extend(command_choices(segment[2], depth + 1))
        elif executable in ("lark-cli", "feishu-cli-next"):
            path = []
            for token in segment[1:]:
                candidate = tuple(path + [token])
                if candidate not in paths:
                    break
                path.append(token)
            flags = sorted({token.split("=", 1)[0] for token in segment[1:]
                            if token.split("=", 1)[0] in allowed_flags})
            result.append({"kind": "cli", "command": path, "flags": flags})
        else:
            kind = "file_read" if executable in ("cat", "head", "tail", "sed") else \
                   "file_search" if executable in ("rg", "grep", "find") else \
                   "script" if executable in ("python", "python3", "node") else "other_shell"
            result.append({"kind": kind})
    return result or [{"kind": "unparsed_shell"}]


def cli_environment(workspace):
    return {"PATH": str(workspace / "bin") + os.pathsep + os.defpath,
            "HOME": str(workspace / "home"), "TMPDIR": str(workspace / "tmp"),
            "LANG": "C.UTF-8", "LC_ALL": "C.UTF-8", "PYTHONUTF8": "1", "PYTHONIOENCODING": "utf-8",
            "LARKSUITE_CLI_CONFIG_DIR": str(workspace / "feishu-config"),
            "LARKSUITE_CLI_REMOTE_META": "off", "LARKSUITE_CLI_NO_UPDATE_NOTIFIER": "1",
            "LARKSUITE_CLI_NO_SKILLS_NOTIFIER": "1"}


def configuration(auth_home, workspace, reasoning, cli_env):
    # Inspect entry paths only; never read harness or Feishu credentials.
    skill_paths = sorted((auth_home / "skills").rglob("SKILL.md"))
    disabled_skills = ",".join("{path=" + json.dumps(str(path)) + ",enabled=false}" for path in skill_paths)
    environment = {**cli_environment(workspace), **cli_env}
    shell_env = ",".join(json.dumps(key) + "=" + json.dumps(value) for key, value in environment.items())
    return ["model_reasoning_effort=" + json.dumps(reasoning), "approval_policy=\"never\"",
            "web_search=\"disabled\"", "project_doc_max_bytes=0", "shell_environment_policy.inherit=\"none\"",
            "shell_environment_policy.set={" + shell_env + "}", "skills.config=[" + disabled_skills + "]",
            "mcp_servers={}", "tools.view_image=false", "sandbox_workspace_write.network_access=true",
            "sandbox_workspace_write.writable_roots=" + json.dumps([
                cli_env[key] for key in ('LARKSUITE_CLI_CONFIG_DIR', 'LARKSUITE_CLI_DATA_DIR')]),
            *["features." + feature + "=false" for feature in DISABLED_FEATURES]]


def parse_events(stdout, returncode):
    """Codex usage covers each entire user turn, including its internal tool loop."""
    spans = []
    started = 0
    completed = 0
    malformed = False
    failed = False
    commands = 0
    control_trace = []
    unexpected_tools = False
    for line in stdout.splitlines():
        try:
            event = json.loads(line)
            kind = event["type"]
        except (ValueError, KeyError, TypeError):
            malformed = True
            continue
        if kind == "turn.started":
            started += 1
        elif kind == "turn.completed":
            completed += 1
            usage = event.get("usage", {})
            if not isinstance(usage, dict):
                malformed = True
                continue
            counts = {key: usage.get(key) for key in ("input_tokens", "cached_input_tokens", "output_tokens")}
            if all(type(value) is int and value >= 0 for value in counts.values()) and counts["cached_input_tokens"] <= counts["input_tokens"]:
                spans.append(counts)
            else:
                malformed = True
        elif kind == "turn.failed":
            failed = True
        elif kind == "item.completed":
            item = event.get("item", {})
            if not isinstance(item, dict):
                malformed = True
                continue
            item_kind = item.get("type")
            if item_kind == "command_execution":
                commands += 1
                exit_code = item.get("exit_code")
                record = {"choices": command_choices(item.get("command")),
                          "exit_code": exit_code if type(exit_code) is int else None}
                diagnostic = reported_cli_error(item.get("aggregated_output"), record["choices"], exit_code)
                if diagnostic:
                    record["reported_cli_error"] = diagnostic
                control_trace.append(record)
            if item_kind in ("mcp_tool_call", "web_search", "collab_tool_call"):
                unexpected_tools = True
    complete = started > 0 and started == completed == len(spans) and not (malformed or failed or unexpected_tools)
    reasons = [code for code, present in (
        ("provider_event_invalid", malformed), ("provider_turn_failed", failed),
        ("provider_turn_count_mismatch", not (started > 0 and started == completed == len(spans))),
        ("unexpected_provider_tool", unexpected_tools)) if present]
    return {"turns": spans, "accounting_complete": complete,
            "accounting_diagnostics": reasons,
            "usage_granularity": "codex_user_turn", "command_executions": commands,
            "unexpected_tools": unexpected_tools, "control_trace": control_trace}


def run(request, codex, auth_home, expected_version):
    if request["settings"]["backend"] != "live-feishu-v1":
        raise ValueError("only real Feishu evaluation is supported")
    return run_agent(request, codex, auth_home, expected_version)


def validate_output_schema(schema):
    """Check the strict object subset used here before any provider request."""
    if not isinstance(schema, dict):
        raise ValueError("Codex output schema must be an object")
    if schema.get("type") == "object":
        properties = schema.get("properties", {})
        required = schema.get("required", [])
        if (schema.get("additionalProperties") is not False or not isinstance(properties, dict)
                or not isinstance(required, list) or not all(isinstance(key, str) for key in required)
                or len(required) != len(properties) or set(required) != set(properties)):
            raise ValueError("Codex output objects must be closed and require every declared property")
        for child in properties.values():
            validate_output_schema(child)
    if isinstance(schema.get("items"), dict):
        validate_output_schema(schema["items"])
    for keyword in ("anyOf", "allOf", "oneOf"):
        for child in schema.get(keyword, []):
            validate_output_schema(child)


def recipient_rule(fixture):
    recipient = fixture.get('authorized_recipient')
    if recipient is None:
        return 'Do not deliver content or grant access to any other account. '
    if (not isinstance(recipient, dict) or set(recipient) != {'open_id', 'tenant_key'}
            or not all(isinstance(value, str) and value.strip() for value in recipient.values())):
        raise ValueError('invalid verified recipient')
    return ('The fixture independently verified this configured recipient: ' + json.dumps(recipient) + '. '
            'Deliver content or grant access only to this identity, only when explicitly authorized by the task. '
            'Do not use display-name matches or groups containing other accounts. ')


def run_agent(request, codex, auth_home, expected_version):
    validate_output_schema(request["fixture"]["answer_schema"])
    delivery_rule = recipient_rule(request['fixture'])
    delivery = request.get("settings", {}).get("guidance_delivery", "inline")
    if delivery not in ("inline", "file"):
        raise ValueError("guidance_delivery must be inline or file")
    if delivery == "file" and Path(request["skill"]).name != "SKILL.md":
        raise ValueError("file guidance requires a SKILL.md entry")
    # Required for safe command traces; reject incomplete archives before billing.
    trace_vocabulary()
    workspace = Path(request["workspace"]).resolve()
    for directory in ("bin", "home", "tmp", "feishu-config"):
        (workspace / directory).mkdir(exist_ok=True)
    # Both conventional names point to the selected arm, so embedded official examples work.
    for name in ("lark-cli", "feishu-cli-next"):
        if os.name == "nt":
            # Native Windows aliases must not require Developer Mode symlink rights.
            shutil.copyfile(request["cli"], workspace / "bin" / (name + ".exe"))
        else:
            (workspace / "bin" / name).symlink_to(request["cli"])
    shutil.copytree(Path(request["skill"]).parent, workspace / "guidance")
    # Frozen official distributions use cross-skill relative references (including
    # compatibility entries). Make those files available without injecting them.
    distribution = Path(request["skill"]).parent.parent
    if (distribution / "lark-shared").is_dir():
        for dependency in distribution.iterdir():
            if dependency.is_dir() and dependency.name.startswith("lark-"):
                shutil.copytree(dependency, workspace / dependency.name)
    cli_env = {}
    for key in ('cli_config_dir', 'cli_data_dir'):
        directory = Path(request['fixture'][key])
        if not directory.is_absolute() or not directory.is_dir():
            raise ValueError('live CLI credentials are unavailable')
        cli_env['LARKSUITE_' + key.upper()] = str(directory)
    env = {**cli_environment(workspace), **cli_env}
    env["CODEX_HOME"] = str(auth_home)
    # Auth is consumed by Codex itself. This adapter neither reads nor copies credentials.
    version = subprocess.run([str(codex), "--version"], env=env, capture_output=True, text=True, timeout=10, check=True)
    if version.stdout.strip() != expected_version:
        raise ValueError("Codex version does not match the pinned adapter version")
    schema = workspace / "answer.schema.json"
    schema.write_text(json.dumps(request["fixture"]["answer_schema"]), encoding="utf-8")
    command = [str(codex), "exec", "--json", "--ephemeral", "--ignore-user-config", "--ignore-rules",
               "--skip-git-repo-check", "--model", request["settings"]["model"],
               "--cd", str(workspace), "--color", "never", "--output-schema", str(schema),
               "--output-last-message", str(workspace / "answer.json")]
    command.extend(["--sandbox", "workspace-write"])
    for value in configuration(auth_home, workspace, request["settings"]["reasoning"], cli_env):
        command.extend(["--config", value])
    command.append("-")
    if delivery == "inline":
        skill = (workspace / "guidance" / Path(request["skill"]).name).read_text(encoding="utf-8")
        guidance = ("The supplied CLI skill follows; its relative references are "
                    "under ./guidance/.\n\n" + skill)
    else:
        guidance = ("The selected CLI skill is at ./guidance/SKILL.md. Read it completely before task actions; "
                    "its relative references are under ./guidance/.")
    environment_note = ('This task uses real Feishu through the preconfigured current account. '
                        'Do not inspect credentials or change authentication. ')
    mutation_rule = ('Use only the private resources this run created and the changes authorized in the task. '
                     + delivery_rule +
                     'Do not replay a mutation with an unknown outcome. ')
    prompt = ("Complete this CLI task using the selected binary on PATH. " + environment_note +
              "Do not log in, initialize an app, or access other credentials. " + mutation_rule + "Work only in this "
              "temporary workspace. Return the requested JSON answer. You may use the shell and files "
              "and choose your approach. " + guidance + "\n\nTask:\n" + request["prompt"])
    timeout = request["settings"]["budget"].get("timeout_seconds")
    if type(timeout) not in (int, float) or timeout <= 0:
        raise ValueError("Codex adapter requires a positive timeout_seconds task budget")
    child = subprocess.Popen(command, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                             text=True, encoding="utf-8", cwd=workspace, env=env, start_new_session=True)
    try:
        stdout, _ = child.communicate(prompt, timeout=timeout)
    except subprocess.TimeoutExpired:
        os.killpg(child.pid, signal.SIGKILL)
        stdout, _ = child.communicate()
    accounting = parse_events(stdout, child.returncode)
    # Raw model/tool output is intentionally discarded; only numeric metadata is emitted.
    response = {"claimed_success": (workspace / "answer.json").is_file(), "usage_source": "provider",
                "model": request["settings"]["model"], "tokenizer": request["settings"]["tokenizer"],
                **accounting}
    return response, 0 if child.returncode == 0 and not accounting["unexpected_tools"] else 1


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--codex", type=Path, required=True, help="Absolute native Codex executable (not a Node shim)")
    parser.add_argument("--auth-home", type=Path, required=True, help="Existing Codex auth home; contents are never read by adapter")
    parser.add_argument("--codex-version", default="codex-cli 0.146.0")
    args = parser.parse_args()
    try:
        if not args.codex.is_absolute() or not args.auth_home.is_absolute():
            raise ValueError("paths must be absolute")
        response, code = run(json.load(sys.stdin), args.codex, args.auth_home, args.codex_version)
        print(json.dumps(response))
        return code
    except (ValueError, KeyError, TypeError, OSError, subprocess.SubprocessError):
        print(json.dumps({"accounting_complete": False, "category": "codex_adapter_failure"}))
        return 1


if __name__ == "__main__":
    sys.exit(main())
