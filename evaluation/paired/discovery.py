# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Local discovery, validated afterwards by the evaluator's selected CLI only."""
from copy import deepcopy
import json
from pathlib import Path
import subprocess
import tempfile

from scenario_protocol import ScenarioBase
from paired import contact

SCENARIO = "paired-discovery"
TASK = {
    "id": SCENARIO, "section": "contact",
    "prompt": "Using local CLI guidance/help/schema, find a working command that retrieves the user record identified by open_id ou_fixture_2. Do not execute a business API request yourself, change accounts, or modify remote resources. Return JSON with argv: an array of arguments after the CLI executable, containing the literal ID and any inline JSON needed, ready to run and return that user record. Use no shell syntax, external files, output paths or credential/profile overrides. Shortcuts, generated commands and the raw API are all allowed; no particular discovery sequence is required.",
    "fixture": {"scenario": SCENARIO, "scenario_section": SCENARIO, "answer_schema": {
        "type": "object", "additionalProperties": False, "required": ["argv"],
        "properties": {"argv": {"type": "array", "items": {"type": "string"}}}}},
    "expected": {"checks": ["agent performed no business reads", "selected CLI executes supplied read argv", "exact target record returned"]},
}


def safe_argv(argv):
    """Restrict the evaluator probe to existing directory reads and inline selectors."""
    if not isinstance(argv, list) or not 2 <= len(argv) <= 30 or any(not isinstance(x, str) or not x or len(x) > 4096 or "\x00" in x for x in argv):
        return False
    positionals, flags = [], {}
    allowed = {"--as", "--user-id", "--user-ids", "--user-id-type", "--department-id-type", "--data", "--params", "--query", "--page-size", "--format"}
    index = 0
    while index < len(argv):
        token = argv[index]
        if token.startswith("-"):
            flag, separator, value = token.partition("=")
            if flag not in allowed or flag in flags:
                return False
            if not separator:
                index += 1
                if index == len(argv):
                    return False
                value = argv[index]
            if not value or value.startswith(("@", "-")):
                return False
            flags[flag] = value
        else:
            positionals.append(token)
        index += 1
    if flags.get("--as", "user") not in ("user", "bot") or flags.get("--format", "json") not in ("json", "pretty"):
        return False
    for key in ("--data", "--params"):
        if key in flags:
            try:
                contact.object_body(flags[key].encode())
            except (ValueError, TypeError):
                return False
    if positionals in (["contact", "+get-user"], ["contact", "+search-user"], ["contact", "user_profiles", "batch_query"],
                       ["contact", "users", "get"], ["contact", "users", "basic_batch"], ["contact", "users", "search"]):
        return True
    if len(positionals) == 3 and positionals[0] == "api":
        method, uri = positionals[1:]
        try:
            path, _ = contact.selectors(uri)
        except ValueError:
            return False
        return ((method.upper() == "GET" and path == contact.GET_BASE + contact.USER_ID) or
                (method.upper() == "POST" and path in (contact.USER_BATCH, contact.USER_SEARCH, contact.USER_PROFILES)))
    return False


class Scenario(ScenarioBase):
    selected_cli = None

    def __init__(self):
        self.probe = {"status": "not_run"}

    def setup(self, workspace):
        self.workspace = Path(workspace)

    def dispatch(self, method, uri, body_bytes, headers):
        path, query = contact.selectors(uri)
        body = contact.object_body(body_bytes)
        result = contact.identity_reply(method, path, query, body)
        if result:
            return result
        raise ValueError("discovery allows identity bootstrap only; no business API calls")

    def close(self):
        if self.probe["status"] != "not_run":
            return
        self.probe = {"status": "invalid_answer"}
        try:
            answer = json.loads((self.workspace / "answer.json").read_text())
            if not isinstance(answer, dict) or set(answer) != {"argv"} or not safe_argv(answer["argv"]):
                return
            selected = Path(self.selected_cli)
            if not selected.is_absolute() or not selected.is_file():
                return
            # Fresh synthetic HOME/config; never inherit developer auth or an
            # agent-written executable/environment. This probe is evaluator work.
            from codex_runner import cli_environment
            from scenario_fixture import ScenarioFixture
            backend = contact.Scenario()
            with tempfile.TemporaryDirectory(prefix="paired-discovery-probe-") as directory:
                workspace = Path(directory)
                for child in ("bin", "home", "tmp", "feishu-config"):
                    (workspace / child).mkdir()
                with ScenarioFixture("paired-discovery-probe", backend) as fixture:
                    result = subprocess.run([str(selected), *answer["argv"]], cwd=workspace,
                        env={**cli_environment(workspace), **fixture.environment()},
                        stdin=subprocess.DEVNULL, capture_output=True, text=True, timeout=20, shell=False)
                self.probe = {"status": "completed", "argv": answer["argv"], "exit_code": result.returncode,
                              "target_in_output": contact.USER_ID in result.stdout,
                              "receipt": fixture.receipt()}
        except (OSError, ValueError, TypeError, subprocess.TimeoutExpired):
            self.probe = {"status": "probe_failed"}

    def snapshot(self):
        return {"verifier_only_probe": deepcopy(self.probe)}


def verify(workspace, answer, receipt, expected):
    del workspace, expected
    try:
        if receipt.get("scenario") != SCENARIO or not isinstance(answer, dict) or set(answer) != {"argv"} or not safe_argv(answer["argv"]):
            return False
        # Inherited official startup identity enrichment is allowed. The agent
        # must not execute the candidate business read; only our later probe does.
        if any(r.get("route") != "identity" or r.get("status") != 200 or r.get("signature_valid") is not True
               or r.get("method") != "GET" or r.get("uri") not in ("/open-apis/authen/v1/user_info", "/open-apis/bot/v3/info") for r in receipt["requests"]):
            return False
        probe = receipt["state"]["verifier_only_probe"]
        if (probe.get("status") != "completed" or probe.get("argv") != answer["argv"]
                or type(probe.get("exit_code")) is not int or probe["exit_code"] != 0 or probe.get("target_in_output") is not True):
            return False
        if probe["receipt"].get("scenario") != "paired-discovery-probe":
            return False
        seen = contact.observed_records(probe["receipt"])
        return (contact.USERS[1] in seen["users"] or contact.PROFILES[1] in seen["profiles"]) and not seen["bots"]
    except (KeyError, TypeError, ValueError, AttributeError):
        return False
