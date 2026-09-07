#!/usr/bin/env python3
"""Codex 0.146 JSONL adapter for offline foundation development tasks."""
import argparse
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys


DISABLED_FEATURES = (
    "apps", "browser_use", "browser_use_external", "chronicle", "computer_use",
    "hooks", "image_generation", "in_app_browser", "memories", "multi_agent",
    "multi_agent_v2", "plugins", "remote_plugin", "skill_search", "tool_suggest",
    "workspace_dependencies", "shell_snapshot",
)


def cli_environment(workspace):
    return {"PATH": str(workspace / "bin") + os.pathsep + os.defpath,
            "HOME": str(workspace / "home"), "TMPDIR": str(workspace / "tmp"),
            "LANG": "C.UTF-8", "LC_ALL": "C.UTF-8",
            "LARKSUITE_CLI_CONFIG_DIR": str(workspace / "feishu-config"),
            "LARKSUITE_CLI_REMOTE_META": "off", "LARKSUITE_CLI_NO_UPDATE_NOTIFIER": "1",
            "LARKSUITE_CLI_NO_SKILLS_NOTIFIER": "1"}


def configuration(auth_home, workspace, reasoning):
    # Inspect skill path names only. Never read config.toml, auth files, or skill contents.
    skill_paths = sorted(path.parent for path in (auth_home / "skills").rglob("SKILL.md"))
    disabled_skills = ",".join("{path=" + json.dumps(str(path)) + ",enabled=false}" for path in skill_paths)
    shell_env = ",".join(json.dumps(key) + "=" + json.dumps(value) for key, value in cli_environment(workspace).items())
    return ["model_reasoning_effort=" + json.dumps(reasoning), "approval_policy=\"never\"",
            "sandbox_workspace_write.network_access=false", "web_search=\"disabled\"",
            "project_doc_max_bytes=0", "shell_environment_policy.inherit=\"none\"",
            "shell_environment_policy.set={" + shell_env + "}", "skills.config=[" + disabled_skills + "]",
            "mcp_servers={}", "tools.view_image=false"] + ["features." + feature + "=false" for feature in DISABLED_FEATURES]


def parse_events(stdout, returncode):
    """Codex usage covers each entire user turn, including its internal tool loop."""
    spans = []
    started = 0
    completed = 0
    malformed = False
    failed = False
    commands = 0
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
            if item_kind in ("mcp_tool_call", "web_search", "collab_tool_call"):
                unexpected_tools = True
    complete = started > 0 and started == completed == len(spans) and not (malformed or failed or unexpected_tools)
    return {"turns": spans, "accounting_complete": complete,
            "usage_granularity": "codex_user_turn", "command_executions": commands,
            "unexpected_tools": unexpected_tools}


def run(request, codex, auth_home, expected_version):
    if request["settings"]["backend"] != "offline-foundation-v1":
        raise ValueError("this adapter supports offline foundation fixtures only")
    workspace = Path(request["workspace"]).resolve()
    for directory in ("bin", "home", "tmp", "feishu-config"):
        (workspace / directory).mkdir(exist_ok=True)
    # Both conventional names point to the selected arm, so embedded official examples work.
    for name in ("lark-cli", "feishu-cli-next"):
        (workspace / "bin" / name).symlink_to(request["cli"])
    shutil.copytree(Path(request["skill"]).parent, workspace / "guidance")
    env = cli_environment(workspace)
    env["CODEX_HOME"] = str(auth_home)
    # Auth is consumed by Codex itself. This adapter neither reads nor copies credentials.
    version = subprocess.run([str(codex), "--version"], env=env, capture_output=True, text=True, timeout=10, check=True)
    if version.stdout.strip() != expected_version:
        raise ValueError("Codex version does not match the pinned adapter version")
    schema = workspace / "answer.schema.json"
    schema.write_text(json.dumps(request["fixture"]["answer_schema"]), encoding="utf-8")
    command = [str(codex), "exec", "--json", "--ephemeral", "--ignore-user-config", "--ignore-rules",
               "--skip-git-repo-check", "--sandbox", "workspace-write", "--model", request["settings"]["model"],
               "--cd", str(workspace), "--color", "never", "--output-schema", str(schema),
               "--output-last-message", str(workspace / "answer.json")]
    for value in configuration(auth_home, workspace, request["settings"]["reasoning"]):
        command.extend(["--config", value])
    command.append("-")
    skill = (workspace / "guidance" / Path(request["skill"]).name).read_text(encoding="utf-8")
    prompt = ("Complete this offline CLI task using the selected binary on PATH. The project has an empty "
              "Feishu configuration. Network access for shell commands is disabled; do not log in, "
              "initialize an app, access credentials, or change any backend resources. Work only in this "
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
