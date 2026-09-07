# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT
"""Small real-process contracts; broader foundations reuse the upstream suites."""

import json
import os
from pathlib import Path
import subprocess
import unittest


class FoundationProcessTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        binary = os.environ.get("LARK_CLI_BIN")
        if not binary or not Path(binary).is_file():
            raise RuntimeError("Use scripts/next_test.py with --binary pointing to the built CLI")
        cls.binary = binary
        config = os.environ.get("LARKSUITE_CLI_CONFIG_DIR", "")
        home = Path(os.environ.get("HOME", "/missing-home"))
        if not config or home.parent != Path(config).parent or not home.parent.name.startswith("next-tests-"):
            raise RuntimeError("Foundation E2E requires the isolated next_test.py environment")

    def run_cli(self, *args):
        return subprocess.run([self.binary, *args], capture_output=True, text=True, timeout=30)

    def test_help_and_version_are_available_without_authentication(self):
        for args in (("--help",), ("--version",)):
            with self.subTest(args=args):
                result = self.run_cli(*args)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertTrue(result.stdout.strip())

    def test_invalid_command_has_machine_readable_failure(self):
        result = self.run_cli("no-such-foundation-command")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout.strip(), "")
        error = json.loads(result.stderr)
        self.assertIn("error", error)
        self.assertTrue(error["error"].get("type"))

    def test_fresh_auth_status_does_not_discover_developer_credentials(self):
        result = self.run_cli("auth", "status")
        payload = json.loads(result.stdout or result.stderr)
        encoded = json.dumps(payload)
        self.assertNotIn('"verified": true', encoded)
        self.assertNotIn('"status": "ready"', encoded)
        if result.returncode == 0:
            self.assertEqual(payload.get("identity"), "none")
        else:
            self.assertIn("error", payload)

    def test_schema_is_available_from_embedded_metadata(self):
        result = self.run_cli("schema", "im", "messages", "list")
        self.assertEqual(result.returncode, 0, result.stderr)
        payload = json.loads(result.stdout)
        self.assertTrue(payload)
        self.assertEqual(result.stderr.strip(), "")

    def test_config_show_reports_missing_config(self):
        result = self.run_cli("config", "show")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout.strip(), "")
        self.assertIn("error", json.loads(result.stderr))


if __name__ == "__main__":
    unittest.main()
