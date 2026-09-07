#!/usr/bin/env python3
"""Codex JSONL adapter for offline and loopback synthetic development tasks."""
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


@functools.lru_cache(maxsize=1)
def trace_vocabulary():
    """Only published command/flag names may appear in retained trace evidence."""
    snapshot = Path(__file__).resolve().parents[1] / "docs/reference/registrations.json.gz"
    with gzip.open(snapshot, "rt", encoding="utf-8") as stream:
        registrations = json.load(stream)
    paths = {("discover",)}
    flags = {"--help", "-h"}
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
            "LANG": "C.UTF-8", "LC_ALL": "C.UTF-8",
            "LARKSUITE_CLI_CONFIG_DIR": str(workspace / "feishu-config"),
            "LARKSUITE_CLI_REMOTE_META": "off", "LARKSUITE_CLI_NO_UPDATE_NOTIFIER": "1",
            "LARKSUITE_CLI_NO_SKILLS_NOTIFIER": "1"}


def configuration(auth_home, workspace, reasoning, fixture_env=None):
    # Inspect skill path names only. Never read config.toml, auth files, or skill contents.
    skill_paths = sorted(path.parent for path in (auth_home / "skills").rglob("SKILL.md"))
    disabled_skills = ",".join("{path=" + json.dumps(str(path)) + ",enabled=false}" for path in skill_paths)
    environment = {**cli_environment(workspace), **(fixture_env or {})}
    shell_env = ",".join(json.dumps(key) + "=" + json.dumps(value) for key, value in environment.items())
    network = (['default_permissions="evaluation"',
                'permissions.evaluation={extends=":workspace",network={enabled=true,allow_local_binding=true,domains={"127.0.0.1"="allow"}}}',
                'features.network_proxy={enabled=true,allow_upstream_proxy=false}']
               if fixture_env else ["sandbox_workspace_write.network_access=false"])
    return ["model_reasoning_effort=" + json.dumps(reasoning), "approval_policy=\"never\"",
            "web_search=\"disabled\"",
            "project_doc_max_bytes=0", "shell_environment_policy.inherit=\"none\"",
            "shell_environment_policy.set={" + shell_env + "}", "skills.config=[" + disabled_skills + "]",
            "mcp_servers={}", "tools.view_image=false"] + network + ["features." + feature + "=false" for feature in DISABLED_FEATURES]


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
                control_trace.append({"choices": command_choices(item.get("command")),
                                      "exit_code": exit_code if type(exit_code) is int else None})
            if item_kind in ("mcp_tool_call", "web_search", "collab_tool_call"):
                unexpected_tools = True
    complete = started > 0 and started == completed == len(spans) and not (malformed or failed or unexpected_tools)
    return {"turns": spans, "accounting_complete": complete,
            "usage_granularity": "codex_user_turn", "command_executions": commands,
            "unexpected_tools": unexpected_tools, "control_trace": control_trace}


def run(request, codex, auth_home, expected_version):
    backend = request["settings"]["backend"]
    if backend == "synthetic-contact-v1":
        from section_fixture import SectionFixture
        with SectionFixture(request["fixture"]["scenario"]) as fixture:
            try:
                response, code = run_agent(request, codex, auth_home, expected_version, fixture.environment())
                response["api_calls"] = len(fixture.receipt()["requests"])
                return response, code
            finally:
                fixture.write_receipt(request["workspace"])
    if backend != "offline-foundation-v1":
        raise ValueError("unsupported evaluation backend")
    return run_agent(request, codex, auth_home, expected_version)


def run_agent(request, codex, auth_home, expected_version, fixture_env=None):
    workspace = Path(request["workspace"]).resolve()
    for directory in ("bin", "home", "tmp", "feishu-config"):
        (workspace / directory).mkdir(exist_ok=True)
    # Both conventional names point to the selected arm, so embedded official examples work.
    for name in ("lark-cli", "feishu-cli-next"):
        (workspace / "bin" / name).symlink_to(request["cli"])
    shutil.copytree(Path(request["skill"]).parent, workspace / "guidance")
    shared = Path(request["skill"]).parent.parent / "lark-shared"
    if Path(request["skill"]).parent.name == "lark-contact" and shared.is_dir():
        shutil.copytree(shared, workspace / "lark-shared")
    env = {**cli_environment(workspace), **(fixture_env or {})}
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
    if not fixture_env:
        command.extend(["--sandbox", "workspace-write"])
    for value in configuration(auth_home, workspace, request["settings"]["reasoning"], fixture_env):
        command.extend(["--config", value])
    command.append("-")
    skill = (workspace / "guidance" / Path(request["skill"]).name).read_text(encoding="utf-8")
    environment_note = ("All Feishu API calls use a local synthetic read-only fixture through the preconfigured "
                        "auth sidecar; user and bot identities are available with sentinel credentials. Only "
                        "loopback HTTP is permitted. Do not inspect or change auth/proxy settings. " if fixture_env else
                        "The project has an empty Feishu configuration. Network access for shell commands is disabled. ")
    prompt = ("Complete this CLI task using the selected binary on PATH. " + environment_note +
              "Do not log in, initialize an app, access other credentials, or change any backend resources. Work only in this "
              "temporary workspace. Return the requested JSON answer. You may use the shell and files "
              "and choose your approach. The supplied CLI skill follows; its relative references are "
              "under ./guidance/.\n\n" + skill + "\n\nTask:\n" + request["prompt"])
    timeout = request["settings"]["budget"].get("timeout_seconds")
    if type(timeout) not in (int, float) or timeout <= 0:
        raise ValueError("Codex adapter requires a positive timeout_seconds task budget")
    child = subprocess.Popen(command, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                             text=True, cwd=workspace, env=env, start_new_session=True)
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
