#!/usr/bin/env python3
"""Independent, offline observation for foundation discovery answers."""
import json
from pathlib import Path
import subprocess
import sys

from codex_runner import cli_environment


def verify(request):
    workspace = Path(request["workspace"])
    answer = json.loads((workspace / "answer.json").read_text(encoding="utf-8"))
    expected = request["expected"]
    if answer != expected["answer"]:
        return False
    # No configuration may have been created during read-only setup/discovery tasks.
    if any(path.is_file() for path in (workspace / "feishu-config").rglob("*")):
        return False
    # Check the source of truth through a separate actual CLI process, not an agent claim.
    command = expected["observe"]
    if command not in (["auth", "status", "--json"], ["docs", "+fetch", "--help"]):
        return False
    result = subprocess.run([request["cli"], *command], cwd=workspace, env=cli_environment(workspace),
                            capture_output=True, text=True, timeout=15)
    if result.returncode != expected["exit_code"]:
        return False
    if command[0] == "auth":
        failure = json.loads(result.stderr)
        return not result.stdout and failure.get("error", {}).get("subtype") == "not_configured"
    return all(fragment in result.stdout for fragment in expected["help_contains"])


if __name__ == "__main__":
    try:
        success = verify(json.load(sys.stdin))
    except (ValueError, KeyError, TypeError, OSError, subprocess.SubprocessError):
        success = False
    print(json.dumps({"success": success}))
