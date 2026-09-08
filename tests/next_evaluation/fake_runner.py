#!/usr/bin/env python3
"""Synthetic protocol fixture. Never use its invented counts as model evidence."""
import json
import os
from pathlib import Path
import sys

request = json.load(sys.stdin)
assert "expected" not in request
assert "FEISHU_EVAL_TEST_SECRET" not in os.environ
assert not (Path(request["workspace"]) / "state.json").exists()
fixture = request["fixture"]
if fixture.get("invocation_log"):
    with Path(fixture["invocation_log"]).open("a") as log:
        log.write(json.dumps({"task_id": request["task_id"], "trial": request["trial"], "arm": request["arm"]}) + "\n")
if fixture.get("invalid_usage"):
    print("invalid protocol")
    sys.exit(0)
state = {"value": fixture["value"] + 1}
if fixture.get("wrong_effect"):
    state["value"] = -1
Path("state.json").write_text(json.dumps(state))
print(json.dumps({"claimed_success": True, "usage_source": "synthetic",
                  "model": request["settings"]["model"], "tokenizer": request["settings"]["tokenizer"],
                  "accounting_complete": True,
                  "turns": [{"input_tokens": 100, "cached_input_tokens": 20, "output_tokens": 10},
                            {"input_tokens": 50, "cached_input_tokens": 0, "output_tokens": 5}],
                  "api_calls": 2, "retries": 1, "command_executions": 1,
                  "control_trace": [{"choices": [{"kind": "cli", "command": ["auth", "status"], "flags": ["--json"], "payload": "private marker"}], "exit_code": 3}]}))
if fixture.get("operator_stop_file"):
    Path(fixture["operator_stop_file"]).write_text("controlled operator cancellation")
print("synthetic stderr private marker", file=sys.stderr)
sys.exit(7 if fixture.get("nonzero_exit") else 0)
