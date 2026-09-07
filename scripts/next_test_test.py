# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

import contextlib
import io
import json
import os
from pathlib import Path
import socket
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

import next_test


class RunnerTests(unittest.TestCase):
    def test_child_cannot_inherit_credentials_or_state(self):
        with tempfile.TemporaryDirectory() as directory:
            parent = dict(os.environ, HOME="/private/developer", TEST_USER_ACCESS_TOKEN="secret",
                          LARKSUITE_CLI_CONFIG_DIR="/private/config", LARKSUITE_CLI_USER_ACCESS_TOKEN="secret",
                          LARK_CLI_BIN="/private/cli", LARKSUITE_CLI_BRAND="lark", AWS_SECRET_ACCESS_KEY="secret", PYTHONPATH="/private/modules",
                          NO_PROXY="*", GOFLAGS="-modfile=/private/go.mod")
            env = next_test.isolated_env(directory, parent, "http://127.0.0.1:1")
            result = next_test.run_command([sys.executable, "-c", "import json,os;print(json.dumps(dict(os.environ)))"], env, 10)
            self.assertEqual(result["exit_code"], 0)
            child = json.loads(result["stdout"])
            for name in ("TEST_USER_ACCESS_TOKEN", "LARKSUITE_CLI_USER_ACCESS_TOKEN", "AWS_SECRET_ACCESS_KEY",
                         "LARK_CLI_BIN", "LARKSUITE_CLI_BRAND", "PYTHONPATH", "GOFLAGS"):
                self.assertNotIn(name, child)
            for name in ("HOME", "XDG_CONFIG_HOME", "XDG_STATE_HOME", "LARKSUITE_CLI_CONFIG_DIR"):
                self.assertTrue(Path(child[name]).is_relative_to(directory))
                self.assertTrue(Path(child[name]).is_dir())
            self.assertEqual(child["LARKSUITE_CLI_REMOTE_META"], "off")
            self.assertEqual(child["GOENV"], "off")
            self.assertEqual(child["GOPROXY"], "off")
            self.assertEqual(child["HTTP_PROXY"], "http://127.0.0.1:1")
            self.assertNotEqual(child["NO_PROXY"], "*")

    def test_proxy_blocks_external_http_without_contacting_target(self):
        with tempfile.TemporaryDirectory() as directory, socket.socket() as proxy:
            proxy.bind(("127.0.0.1", 0))
            env = next_test.isolated_env(directory, os.environ, "http://127.0.0.1:" + str(proxy.getsockname()[1]))
            source = """import json,socket,urllib.request
original = socket.create_connection
def connect(address, *args, **kwargs):
    print(json.dumps(list(address)), flush=True)
    return original(address, *args, **kwargs)
socket.create_connection = connect
urllib.request.urlopen('http://must-not-resolve.invalid',timeout=0.2)
"""
            result = next_test.run_command([sys.executable, "-c", source], env, 10)
            self.assertNotEqual(result["exit_code"], 0)
            self.assertEqual(json.loads(result["stdout"]), ["127.0.0.1", proxy.getsockname()[1]])

    def test_exit_and_go_test_skip_evidence(self):
        source = "import json,sys;print(json.dumps({'Test':'TestFixture','Action':'skip'}));print('diagnostic',file=sys.stderr);sys.exit(7)"
        with tempfile.TemporaryDirectory() as directory:
            result = next_test.run_command([sys.executable, "-c", source],
                                          next_test.isolated_env(directory, os.environ, "http://127.0.0.1:1"), 10)
        self.assertEqual(result["exit_code"], 7)
        self.assertEqual(result["status"], "fail")
        self.assertEqual(result["tests"], {"pass": 0, "fail": 0, "skip": 1})
        self.assertEqual(result["stderr"], "diagnostic\n")

    def test_python_unittest_counts_keep_skips_and_failures_separate(self):
        for summary, expected, code in [
            ("OK (skipped=2)", {"pass": 3, "fail": 0, "skip": 2}, 0),
            ("FAILED (failures=1, errors=1, skipped=1)", {"pass": 2, "fail": 2, "skip": 1}, 1),
        ]:
            source = "import sys;print(" + repr("Ran 5 tests in 0.1s\n\n" + summary) + ",file=sys.stderr);sys.exit(" + str(code) + ")"
            with tempfile.TemporaryDirectory() as directory:
                result = next_test.run_command([sys.executable, "-c", source],
                                              next_test.isolated_env(directory, os.environ, "http://127.0.0.1:1"), 10)
            self.assertEqual(result["tests"], expected)
            self.assertEqual(result["exit_code"], code)

    def test_harness_selectors_run_complete_python_discovery_without_binary(self):
        for harness, folder, pattern in [("scripts", "scripts", "next_*test.py"),
                                         ("evaluation", "tests/next_evaluation", "test_*.py")]:
            with tempfile.TemporaryDirectory() as directory:
                report_path = Path(directory) / "report.json"
                def passed(command, env, _timeout):
                    self.assertTrue(Path(env["HOME"]).is_dir())
                    self.assertNotIn("TEST_USER_ACCESS_TOKEN", env)
                    self.assertEqual(command, [sys.executable, "-m", "unittest", "discover", "-s", folder, "-p", pattern, "-v"])
                    return {"command": command, "status": "pass", "exit_code": 0,
                            "stdout": "", "stderr": "", "tests": {"pass": 1, "fail": 0, "skip": 0}}
                with patch.object(next_test, "run_command", side_effect=passed), contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
                    code = next_test.main(["--harness", harness, "--binary", "does-not-exist", "--report", str(report_path)])
                self.assertEqual(code, 0)
                report = json.loads(report_path.read_text())
                self.assertEqual(len(report["commands"]), 1)
                self.assertEqual(report["evidence"], "offline")

    def test_timeout_is_failure(self):
        with tempfile.TemporaryDirectory() as directory:
            result = next_test.run_command([sys.executable, "-c", "import time;time.sleep(30)"],
                                          next_test.isolated_env(directory, os.environ, "http://127.0.0.1:1"), 0.05)
        self.assertEqual(result["exit_code"], 124)
        self.assertEqual(result["status"], "timeout")

    def test_main_forwards_failure_and_removes_isolated_state(self):
        with tempfile.TemporaryDirectory() as directory:
            report_path = Path(directory) / "report.json"
            child_state = Path(directory) / "child-state"
            source = "import os,pathlib,sys;pathlib.Path(sys.argv[1]).write_text(os.environ['HOME']);sys.exit(9)"
            command = [sys.executable, "-c", source, str(child_state)]
            with patch.object(next_test, "commands_for", return_value=[command]), contextlib.redirect_stdout(io.StringIO()) as output, contextlib.redirect_stderr(io.StringIO()):
                code = next_test.main(["--suite", "unit", "--report", str(report_path)])
            report = json.loads(report_path.read_text())
            self.assertEqual(code, 9)
            self.assertEqual(json.loads(output.getvalue())["exit_code"], 9)
            self.assertEqual(report["status"], "fail")
            self.assertEqual(report["commands"][0]["exit_code"], 9)
            self.assertFalse(Path(child_state.read_text()).exists())

    def test_live_is_blocked_even_if_opted_in(self):
        with tempfile.TemporaryDirectory() as directory:
            report_path = Path(directory) / "report.json"
            for opt_in in ([], ["--allow-live"]):
                command = [sys.executable, str(Path(next_test.__file__)), "--suite", "live", "--report", str(report_path)] + opt_in
                result = subprocess.run(command, capture_output=True, text=True)
                self.assertEqual(result.returncode, 2)
                report = json.loads(report_path.read_text())
                self.assertEqual(report["status"], "blocked")
                self.assertEqual(report["evidence"], "live")
                self.assertEqual(report["commands"], [])

    def test_section_selector_keeps_live_workflows_out(self):
        commands = next_test.commands_for("docs", "all")
        self.assertIn("./shortcuts/doc/...", commands[0])
        self.assertIn("./tests/cli_e2e/docs", commands[1])
        self.assertEqual(commands[1][-2:], ["-run", "DryRun|Regression"])
        all_commands = next_test.commands_for("all", "e2e")
        self.assertTrue(any("./tests/plugin_e2e" in command for command in all_commands))
        self.assertTrue(any("./tests/cli_e2e/dryrun" in command for command in all_commands))
        self.assertFalse(any("./tests/..." in command for command in all_commands))

    def test_report_exposes_excluded_tests(self):
        selection = next_test.selection_evidence(next_test.commands_for("im", "e2e")[0])
        self.assertTrue(selection["selected_tests"])
        self.assertTrue(selection["excluded_tests"])
        self.assertTrue(all("DryRun" in item["test"] or "Regression" in item["test"]
                            for item in selection["selected_tests"]))
        self.assertTrue(any("Workflow" in item["test"] for item in selection["excluded_tests"]))


    @unittest.skipUnless(os.name == "posix", "Unix socket path limits apply on POSIX")
    def test_default_temp_root_keeps_unix_socket_paths_short(self):
        source = "import os,socket; sock=socket.socket(socket.AF_UNIX); sock.bind(os.path.join(os.environ['TMPDIR'], 's' * 60)); sock.close()"
        command = [sys.executable, "-c", source]
        with tempfile.TemporaryDirectory() as directory:
            with patch.object(next_test, "commands_for", return_value=[command]), contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
                code = next_test.main(["--suite", "unit", "--report", str(Path(directory) / "report.json")])
        self.assertEqual(code, 0)


    def make_section_suite(self, root, section):
        path = root / "tests/next_sections" / section
        path.mkdir(parents=True)
        (path / "workflow_test.go").write_text("package section\n")

    def test_section_includes_its_next_package_without_name_filter(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.make_section_suite(root, "docs")
            self.make_section_suite(root, "sheets")
            (root / "tests/cli_e2e/docs").mkdir(parents=True)
            with patch.object(next_test, "ROOT", root):
                for suite in ("e2e", "all"):
                    commands = next_test.commands_for("docs", suite)
                    own = [command for command in commands if "./tests/next_sections/docs" in command]
                    self.assertEqual(len(own), 1)
                    self.assertNotIn("-run", own[0])
                    self.assertFalse(any("./tests/next_sections/sheets" in command for command in commands))
                    self.assertTrue(any(command[-2:] == ["-run", "DryRun|Regression"] for command in commands))

    def test_all_discovers_existing_sections_and_foundation_stays_separate(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for section in ("docs", "sheets", "events", "extensions", "packaging"):
                self.make_section_suite(root, section)
            with patch.object(next_test, "ROOT", root):
                commands = next_test.commands_for("all", "e2e")
                for section in ("docs", "sheets", "events", "extensions", "packaging"):
                    self.assertTrue(any("./tests/next_sections/" + section in command and "-run" not in command for command in commands))
                foundation = next_test.commands_for("foundation", "e2e")
                self.assertFalse(any(any("next_sections" in arg for arg in command) for command in foundation))
                self.assertEqual(next_test.section_suites("foundation"), [])

    def test_missing_section_suite_blocks_despite_passing_inherited_subset(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            report_path = root / "report.json"
            (root / "tests/cli_e2e/docs").mkdir(parents=True)
            (root / "tests/next_sections/docs").mkdir(parents=True)
            def passed(command, *_args):
                return {"command": command, "status": "pass", "exit_code": 0,
                        "stdout": "", "stderr": "", "tests": {"pass": 1, "fail": 0, "skip": 0}}
            with patch.object(next_test, "ROOT", root), patch.object(next_test, "run_command", side_effect=passed), contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
                code = next_test.main(["--section", "docs", "--suite", "e2e", "--binary", str(Path(__file__).resolve()), "--report", str(report_path)])
            self.assertEqual(code, 2)
            report = json.loads(report_path.read_text())
            self.assertEqual(report["status"], "blocked")
            self.assertEqual(report["commands"][0]["status"], "pass")
            self.assertEqual(report["section_suites"], [{"section": "docs", "package": "./tests/next_sections/docs", "status": "missing"}])

    def test_present_section_suite_runs_and_is_reported(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.make_section_suite(root, "docs")
            report_path = root / "report.json"
            def passed(command, *_args):
                return {"command": command, "status": "pass", "exit_code": 0,
                        "stdout": "", "stderr": "", "tests": {"pass": 1, "fail": 0, "skip": 0}}
            with patch.object(next_test, "ROOT", root), patch.object(next_test, "run_command", side_effect=passed) as run, contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
                code = next_test.main(["--section", "docs", "--suite", "e2e", "--binary", str(Path(__file__).resolve()), "--report", str(report_path)])
            self.assertEqual(code, 0)
            self.assertEqual(run.call_count, 1)
            self.assertIn("./tests/next_sections/docs", run.call_args.args[0])
            report = json.loads(report_path.read_text())
            self.assertEqual(report["section_suites"][0]["status"], "present")
            self.assertEqual(report["status"], "pass")


    def test_events_unit_suite_covers_command_and_runtime_owners(self):
        command = next_test.commands_for("events", "unit")[0]
        for package in ("./shortcuts/event/...", "./events/...", "./cmd/event/...", "./internal/event/..."):
            self.assertIn(package, command)

    def test_business_selects_shared_generated_owners_without_a_shortcut_package(self):
        self.assertIn("business", next_test.DOMAINS)
        command = next_test.commands_for("business", "unit")[0]
        for package in ("./cmd/service/...", "./internal/apicatalog/...", "./internal/registry/...",
                        "./internal/schema/...", "./internal/client/...", "./internal/nextattempt/..."):
            self.assertIn(package, command)
        self.assertFalse(any("shortcuts/business" in arg for arg in command))
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with patch.object(next_test, "ROOT", root):
                self.assertIn({"section": "business", "package": "./tests/next_sections/business", "status": "missing"}, next_test.section_suites("all"))
            self.make_section_suite(root, "business")
            with patch.object(next_test, "ROOT", root):
                commands = next_test.commands_for("business", "e2e")
                self.assertEqual(len(commands), 1)
                self.assertIn("./tests/next_sections/business", commands[0])
                self.assertNotIn("-run", commands[0])


    def test_integration_runs_only_the_requested_complete_package(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            package = root / "tests/next_integration/sheets_base"
            package.mkdir(parents=True)
            (package / "transfer_test.go").write_text("package integration\n")
            report_path = root / "report.json"
            def passed(command, *_args):
                return {"command": command, "status": "pass", "exit_code": 0,
                        "stdout": "", "stderr": "", "tests": {"pass": 1, "fail": 0, "skip": 0}}
            with patch.object(next_test, "ROOT", root), patch.object(next_test, "run_command", side_effect=passed) as run, contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
                code = next_test.main(["--integration", "sheets_base", "--binary", str(Path(__file__).resolve()), "--report", str(report_path)])
            self.assertEqual(code, 0)
            self.assertEqual(run.call_count, 1)
            self.assertEqual(run.call_args.args[0], ["go", "test", "-json", "-count=1", "-timeout=10m", "./tests/next_integration/sheets_base"])
            report = json.loads(report_path.read_text())
            self.assertEqual(report["section"], "integration")
            self.assertEqual(report["integration"], "sheets_base")

    def test_missing_integration_is_blocked_without_running_other_suites(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            report_path = root / "report.json"
            with patch.object(next_test, "ROOT", root), patch.object(next_test, "run_command") as run, contextlib.redirect_stdout(io.StringIO()):
                code = next_test.main(["--integration", "sheets_base", "--binary", str(Path(__file__).resolve()), "--report", str(report_path)])
            self.assertEqual(code, 2)
            run.assert_not_called()
            report = json.loads(report_path.read_text())
            self.assertEqual(report["status"], "blocked")
            self.assertEqual(report["section_suites"][0]["status"], "missing")

    def test_integration_rejects_paths_and_conflicting_selection(self):
        import argparse
        for name in ("../foundation", "/tmp/other", "sheets/base", "sheets-base"):
            with self.assertRaises(argparse.ArgumentTypeError):
                next_test.integration_name(name)
        with contextlib.redirect_stderr(io.StringIO()), self.assertRaises(SystemExit):
            next_test.main(["--integration", "sheets_base", "--section", "docs"])



if __name__ == "__main__":
    unittest.main()
