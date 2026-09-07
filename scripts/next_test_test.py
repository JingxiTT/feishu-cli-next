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



if __name__ == "__main__":
    unittest.main()
