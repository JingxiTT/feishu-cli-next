# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

import json
from pathlib import Path
import sys
import tempfile
from types import SimpleNamespace
import unittest
from unittest import mock

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "evaluation"))
from scenarios import extensions


class ExtensionScenarioTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.workspace = Path(self.temporary.name)
        self.scenario = extensions.Scenario()
        self.scenario.selected_cli = self.workspace / "host-cli"
        self.scenario.setup(self.workspace)
        self.stdout = '{"ok":true,"identity":"bot","data":{"value":0}}\n'
        (self.workspace / "host-result.json").write_text(self.stdout)
        self.item = {"type": "command_execution", "command": "host-cli im +fixture-host --as bot",
                     "exit_code": 0, "aggregated_output": self.stdout}
        self.scenario.observe_execution(self.item)
        self.receipt = {"scenario": "extensions-dev-01", "requests": [], "state": self.scenario.snapshot()}

    def verify(self, answer=None, receipt=None):
        return extensions.verify(self.workspace, {"value": 0} if answer is None else answer,
                                 self.receipt if receipt is None else receipt, extensions.TASK["expected"])

    def test_public_fixture_contains_types_not_answer(self):
        self.assertEqual(extensions.TASK["fixture"], {
            "scenario": "extensions-dev-01", "answer_schema": {
                "type": "object", "properties": {"value": {"type": "integer"}},
                "required": ["value"], "additionalProperties": False}})

    def test_real_observation_and_artifact_preserve_integer_zero(self):
        self.assertTrue(self.verify())
        legacy = json.dumps(json.loads(self.stdout), indent=2) + "\n"
        (self.workspace / "host-result.json").write_text(legacy)
        self.scenario.observe_execution({**self.item, "aggregated_output": legacy})
        self.receipt["state"] = self.scenario.snapshot()
        self.assertTrue(self.verify())

    def test_missing_string_boolean_float_null_or_nonzero_answer_fails(self):
        for value in ["0", False, True, 0.0, None, 1, -1]:
            with self.subTest(value=repr(value)):
                self.assertFalse(self.verify({"value": value}))
        self.assertFalse(self.verify({}))

    def test_forged_artifact_without_trusted_execution_fails(self):
        self.receipt["state"]["host_executions"] = []
        self.assertFalse(self.verify())

    def test_bad_execution_and_config_evidence_fails(self):
        for key, value in [("command", "host-cli im +other"), ("exit_code", 1), ("exit_code", False),
                           ("aggregated_output", "error"), ("type", "model_message")]:
            with self.subTest(key=key, value=value):
                self.scenario.executions = []
                self.scenario.observe_execution({**self.item, key: value})
                receipt = {**self.receipt, "state": self.scenario.snapshot()}
                self.assertFalse(self.verify(receipt=receipt))
        self.receipt["state"]["config_unchanged"] = False
        self.assertFalse(self.verify())

    def test_simple_redirect_is_proven_but_shell_compositions_are_not(self):
        self.scenario.executions = []
        self.scenario.observe_execution({**self.item, "command": "/bin/zsh -lc 'host-cli im +fixture-host --as bot > host-result.json'", "aggregated_output": ""})
        self.receipt["state"] = self.scenario.snapshot()
        self.assertTrue(self.verify())
        for command in ["echo host-cli im +fixture-host", "host-cli im +fixture-host; true",
                        "host-cli im +fixture-host | cat", "host-cli im +fixture-host > host-result.json; true",
                        "host-cli im +fixture-host > elsewhere.json", "host-cli im +fixture-host --as user",
                        "/untrusted/host-cli im +fixture-host", "host-cli im +fixture-host >> host-result.json"]:
            with self.subTest(command=command):
                self.scenario.executions = []
                self.scenario.observe_execution({**self.item, "command": command, "aggregated_output": ""})
                self.assertEqual(self.scenario.snapshot()["host_executions"], [])

    def test_published_execution_proof_contains_no_raw_output(self):
        proof = self.scenario.snapshot()["host_executions"][0]
        self.assertEqual(set(proof), {"command_path", "exit_code", "capture", "sha256"})

    def test_supported_json_alias_preserves_observed_and_redirected_proof(self):
        digest = self.receipt["state"]["host_executions"][0]["sha256"]
        for flag in ("--json", "--json=true"):
            for redirect in (False, True):
                with self.subTest(flag=flag, redirect=redirect):
                    self.scenario.executions = []
                    command = "host-cli im +fixture-host --as bot " + flag
                    if redirect:
                        command += " > host-result.json"
                    self.scenario.observe_execution({**self.item, "command": command,
                                                     "aggregated_output": "" if redirect else self.stdout})
                    receipt = {**self.receipt, "state": self.scenario.snapshot()}
                    self.assertTrue(self.verify(receipt=receipt))
                    self.assertEqual(receipt["state"]["host_executions"][0]["sha256"], digest)
                    self.assertEqual((self.workspace / "host-result.json").read_text(), self.stdout)

    def test_false_value_and_compound_json_flags_do_not_prove_execution(self):
        for suffix in ("--json=false", "--json false", "--json=0", "--json=value", "--json true",
                       "--json; true", "--json | cat", "--json --format pretty"):
            with self.subTest(suffix=suffix):
                self.scenario.executions = []
                self.scenario.observe_execution({**self.item, "command": "host-cli im +fixture-host " + suffix})
                self.assertEqual(self.scenario.snapshot()["host_executions"], [])

    def test_artifact_changed_after_observation_fails(self):
        (self.workspace / "host-result.json").write_text(json.dumps(json.loads(self.stdout), indent=2))
        self.assertFalse(self.verify())

    def test_missing_expected_value_does_not_default_to_zero(self):
        self.assertFalse(extensions.verify(self.workspace, {"value": 0}, self.receipt, {}))

    def test_missing_failure_or_tampered_artifact_fails(self):
        for value in [False, "0", None, 1, 0.0]:
            (self.workspace / "host-result.json").write_text(json.dumps({"ok": True, "identity": "bot", "data": {"value": value}}))
            self.assertFalse(self.verify())
        for content in ["{}", self.stdout + "junk", '{"ok":false,"identity":"bot","data":{"value":0}}']:
            (self.workspace / "host-result.json").write_text(content)
            self.assertFalse(self.verify())
        (self.workspace / "host-result.json").unlink()
        self.assertFalse(self.verify())

    def test_requests_and_foreign_scenario_fail(self):
        self.receipt["requests"] = [{"route": "write", "status": 200}]
        self.assertFalse(self.verify())
        self.receipt["requests"] = []
        self.receipt["scenario"] = "other"
        self.assertFalse(self.verify())
        for method, path in [("GET", "/open-apis/authen/v1/user_info"), ("POST", "/anything")]:
            with self.assertRaises(ValueError):
                self.scenario.dispatch(method, path, b"{}", {})

    def test_snapshot_independently_detects_config_writes(self):
        self.assertTrue(self.scenario.snapshot()["config_unchanged"])
        config = Path(self.scenario.environment()["LARKSUITE_CLI_CONFIG_DIR"])
        (config / "unexpected.json").write_text("{}")
        self.assertFalse(self.scenario.snapshot()["config_unchanged"])

    def test_build_uses_reviewed_source_and_only_mode_marker_differs(self):
        source = self.workspace / "source with spaces"
        source.mkdir()
        original_module = "module github.com/larksuite/cli\n\ngo 1.26\n"
        (source / "go.mod").write_text(original_module)
        (source / "go.sum").write_text("")
        seen = []

        def build(command, **kwargs):
            work = Path(kwargs["cwd"])
            seen.append((work.joinpath("main.go").read_text(), work.joinpath("go.mod").read_text()))
            self.assertEqual(command[:3], ["go", "build", "-mod=mod"])
            self.assertEqual(kwargs["env"]["GOPROXY"], "off")
            self.assertNotIn("LARKSUITE_CLI_APP_SECRET", kwargs["env"])
            return SimpleNamespace(returncode=0, stderr="")

        with mock.patch.object(extensions.subprocess, "run", side_effect=build):
            for mode in (False, True):
                result = extensions.build_host(source, self.workspace / ("next" if mode else "official"), mode)
                self.assertIsInstance(result, Path)
        self.assertEqual(seen[1][0].replace("options = append(options, cmd.WithAgentMode())", ""), seen[0][0])
        self.assertIn(json.dumps(source.resolve().as_posix()), seen[0][1])
        self.assertNotIn("/internal/", seen[0][0])
        self.assertEqual((source / "go.mod").read_text(), original_module)
        with self.assertRaises(ValueError):
            extensions.build_host(source, source / "host", False)

    def test_failed_build_is_not_returned_as_ready(self):
        source = self.workspace / "source"
        source.mkdir()
        (source / "go.mod").write_text("module github.com/larksuite/cli\n")
        (source / "go.sum").write_text("")
        with mock.patch.object(extensions.subprocess, "run", return_value=SimpleNamespace(returncode=1, stderr="synthetic compile failure")):
            with self.assertRaisesRegex(RuntimeError, "synthetic compile failure"):
                extensions.build_host(source, self.workspace / "host", False)


if __name__ == "__main__":
    unittest.main()
