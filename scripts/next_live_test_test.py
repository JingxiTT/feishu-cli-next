# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

import contextlib
import io
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import next_live_test


class LocalLiveTests(unittest.TestCase):
    def invoke(self, env=None, flags=("--allow-live",), section="contact", result=None):
        with tempfile.TemporaryDirectory() as directory:
            binary = Path(directory) / "cli"
            binary.touch()
            report = Path(directory) / "report.json"
            ready = {"NEXT_CONTACT_LIVE": "1", "NEXT_CONTACT_LIVE_ACCOUNT_ID": "synthetic-user",
                     "NEXT_CONTACT_LIVE_TENANT_KEY": "synthetic-tenant", "TEST_USER_ACCESS_TOKEN": "synthetic-token"}
            ready.update(env or {})
            with patch.dict(os.environ, ready, clear=True), patch.object(next_live_test, "run_command", return_value=result) as run:
                with contextlib.redirect_stdout(io.StringIO()):
                    code = next_live_test.main(["--section", section, "--binary", str(binary),
                                                "--report", str(report), *flags])
                data = json.loads(report.read_text())
                diagnostics = Path(data["diagnostics"]) if "diagnostics" in data else None
                log = diagnostics.read_text() if diagnostics else ""
                mode = diagnostics.stat().st_mode & 0o777 if diagnostics else None
                return code, data, run, log, mode

    def test_no_process_without_explicit_local_selection_and_fixture(self):
        for env, flags, section in (({}, (), "contact"), ({"CI": "true"}, ("--allow-live",), "contact"),
                                    ({"GITHUB_ACTIONS": "true"}, ("--allow-live",), "contact"),
                                    ({"NEXT_CONTACT_LIVE": ""}, ("--allow-live",), "contact"),
                                    ({"NEXT_CONTACT_LIVE_TENANT_KEY": ""}, ("--allow-live",), "contact"),
                                    ({}, ("--allow-live",), "all"), ({}, ("--allow-live",), "calendar")):
            with self.subTest(env=env, flags=flags, section=section):
                code, report, run, _, _ = self.invoke(env, flags, section)
                self.assertEqual(code, 2)
                self.assertEqual(report["status"], "blocked")
                self.assertEqual(report["commands"], [])
                run.assert_not_called()

    def test_exact_live_selection_uses_local_account_and_private_diagnostics(self):
        result = {"exit_code": 0, "status": "pass", "tests": {"pass": 1, "fail": 0, "skip": 0},
                  "stdout": "synthetic-private-profile", "stderr": "synthetic-private-diagnostic"}
        code, report, run, log, mode = self.invoke(result=result)
        self.assertEqual((code, report["status"], report["evidence"]), (0, "pass", "live"))
        command, env, timeout = run.call_args.args
        self.assertIn("./tests/cli_e2e/contact", command)
        self.assertEqual(command[-2:], ["-run", "^TestContactNextLiveCurrentAccountReads$"])
        self.assertIn("-count=1", command)
        self.assertEqual(env["TEST_USER_ACCESS_TOKEN"], "synthetic-token")
        self.assertIn("LARK_CLI_BIN", env)
        self.assertEqual(run.call_count, 1)
        self.assertNotIn("synthetic-private", json.dumps(report))
        self.assertNotIn("synthetic-token", json.dumps(report))
        self.assertIn("synthetic-private-profile", log)
        self.assertIn("synthetic-private-diagnostic", log)
        if os.name == "posix":
            self.assertEqual(mode, 0o600)

    def test_skipped_or_empty_test_run_is_blocked(self):
        for counts in ({"pass": 0, "skip": 1}, {"pass": 0, "skip": 0}, {"pass": 1, "skip": 1}):
            with self.subTest(counts=counts):
                code, report, run, _, _ = self.invoke(result={"exit_code": 0, "status": "pass", "tests": counts})
                self.assertEqual((code, report["status"]), (2, "blocked"))
                self.assertEqual(run.call_count, 1)

    def test_failure_or_timeout_never_replays(self):
        for exit_code in (1, 124, -9):
            with self.subTest(exit_code=exit_code):
                code, report, run, _, _ = self.invoke(result={"exit_code": exit_code, "status": "fail", "tests": {}})
                self.assertNotEqual(code, 0)
                self.assertEqual(report["status"], "fail")
                self.assertEqual(run.call_count, 1)

    def test_every_selector_names_a_real_account_bound_test(self):
        for section, (package, test, opt_in, required) in next_live_test.SECTIONS.items():
            with self.subTest(section=section):
                sources = "\n".join(path.read_text() for path in
                                    (next_live_test.ROOT / "tests/cli_e2e" / package).glob("*_test.go"))
                self.assertIn("func " + test + "(t *testing.T)", sources)
                self.assertIn('os.Getenv("' + opt_in + '") != "1"', sources)
                for key in required:
                    self.assertIn('os.Getenv("' + key + '")', sources)

    def test_live_entrypoint_is_separate_from_default_tests_and_all_workflows(self):
        makefile = (next_live_test.ROOT / "Makefile").read_text() + (next_live_test.ROOT / "next.mk").read_text()
        self.assertIn("offline-test: test\n", makefile)
        self.assertIn("test: vet fmt-check script-test unit-test examples-build integration-test evaluation-test\n", makefile)
        self.assertIn("live-test:\n\tpython3 scripts/next_live_test.py", makefile)
        for workflow in (next_live_test.ROOT / ".github/workflows").glob("*.y*ml"):
            self.assertNotIn("next_live_test.py", workflow.read_text())
            self.assertNotIn("make live-test", workflow.read_text())


if __name__ == "__main__":
    unittest.main()
