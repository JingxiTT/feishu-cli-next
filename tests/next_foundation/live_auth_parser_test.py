# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

import contextlib
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import live_auth


class LiveAuthParserTests(unittest.TestCase):
    def auth(self):
        return {"appId": "private-app", "identities": {
            "bot": {"status": "ready", "available": True, "verified": True},
            "user": {"status": "needs_refresh", "available": True, "verified": True,
                     "openId": "private-user", "userName": "private-name", "scope": "scope_a scope_b",
                     "hint": "private-hint", "message": "private-message"}}}

    def user(self):
        return {"ok": True, "data": {"open_id": "private-user", "tenant_key": "private-tenant",
                                    "name": "private-name", "access_token": "private-token"}}

    def test_auth_summary_reports_verified_refresh_state_without_private_values(self):
        evidence = live_auth.auth_evidence(self.auth(), 0)
        self.assertEqual(evidence["status"], "pass")
        self.assertEqual(evidence["identities"]["user"]["status"], "needs_refresh")
        self.assertTrue(evidence["identities"]["user"]["verified"])
        self.assertEqual(evidence["identities"]["user"]["scope_count"], 2)
        self.assertNotIn("private-", json.dumps(evidence))
        self.assertNotIn("scope_a", json.dumps(evidence))

    def test_matching_user_binds_tenant_without_claiming_independent_consistency(self):
        evidence = live_auth.own_identity_evidence(self.auth(), self.user(), 0)
        self.assertEqual(evidence["status"], "pass")
        self.assertEqual(evidence["tenant_consistency"], "unavailable")
        self.assertEqual(evidence["tenant_sha256"], live_auth.digest("private-tenant"))
        self.assertNotIn("private-", json.dumps(evidence))

    def test_identity_or_available_tenant_mismatch_fails(self):
        auth = self.auth()
        user = self.user()
        user["data"]["open_id"] = "another-user"
        self.assertEqual(live_auth.own_identity_evidence(auth, user, 0)["reason"], "identity_mismatch")
        auth["tenant_key"] = "another-tenant"
        evidence = live_auth.own_identity_evidence(auth, self.user(), 0)
        self.assertEqual(evidence["status"], "fail")
        self.assertEqual(evidence["tenant_consistency"], "mismatch")
        auth["tenant_key"] = "private-tenant"
        self.assertEqual(live_auth.own_identity_evidence(auth, self.user(), 0)["tenant_consistency"], "match")

    def test_malformed_and_unavailable_evidence_is_not_a_pass(self):
        self.assertEqual(live_auth.auth_evidence(None, 0)["status"], "fail")
        self.assertEqual(live_auth.auth_evidence({}, 0)["status"], "fail")
        self.assertEqual(live_auth.own_identity_evidence(self.auth(), {}, 0)["status"], "fail")
        self.assertEqual(live_auth.own_identity_evidence(self.auth(), None, 3, '{"error":{"type":"authorization"}}')["status"], "skip")
        self.assertEqual(live_auth.own_identity_evidence(self.auth(), None, 124)["status"], "fail")
        auth = self.auth()
        del auth["identities"]["user"]["openId"]
        self.assertEqual(live_auth.own_identity_evidence(auth, self.user(), 0)["status"], "blocked")
        auth = self.auth()
        del auth["appId"]
        self.assertEqual(live_auth.auth_evidence(auth, 0)["status"], "blocked")

    def test_failure_diagnostics_do_not_preserve_messages_or_unknown_types(self):
        stderr = json.dumps({"error": {"type": "private-type", "message": "private-token"}})
        evidence = live_auth.auth_evidence(None, 1, stderr)
        self.assertEqual(evidence["error_type"], "unrecognized")
        self.assertNotIn("private-", json.dumps(evidence))

    def test_default_execution_is_blocked_and_never_invokes_cli(self):
        with tempfile.TemporaryDirectory() as directory:
            report = Path(directory) / "report.json"
            with patch.object(live_auth, "invoke") as invoke, contextlib.redirect_stdout(io.StringIO()):
                code = live_auth.main(["--binary", __file__, "--report", str(report)])
            self.assertEqual(code, 2)
            invoke.assert_not_called()
            evidence = json.loads(report.read_text())
            self.assertEqual(evidence["evidence"], "live")
            self.assertEqual(evidence["status"], "blocked")

    def test_opt_in_runs_only_the_two_fixed_read_commands_and_saves_sanitized_report(self):
        with tempfile.TemporaryDirectory() as directory:
            report = Path(directory) / "report.json"
            with patch.object(live_auth, "invoke", side_effect=[(self.auth(), 0, ""), (self.user(), 0, "")]) as invoke, contextlib.redirect_stdout(io.StringIO()) as output:
                code = live_auth.main(["--allow-live-read", "--binary", __file__, "--report", str(report)])
            self.assertEqual(code, 0)
            self.assertEqual([call.args[1] for call in invoke.call_args_list], [live_auth.AUTH_ARGS, live_auth.USER_ARGS])
            self.assertEqual(live_auth.AUTH_ARGS, ("auth", "status", "--json", "--verify"))
            self.assertEqual(live_auth.USER_ARGS, ("api", "GET", "/open-apis/authen/v1/user_info", "--as", "user"))
            self.assertNotIn("private-", report.read_text())
            self.assertNotIn("private-", output.getvalue())
            self.assertIn("limitations", json.loads(report.read_text()))

    def test_unavailable_verified_user_skips_get_and_blocks_overall_evidence(self):
        auth = self.auth()
        auth["identities"]["user"]["verified"] = False
        with tempfile.TemporaryDirectory() as directory:
            report = Path(directory) / "report.json"
            with patch.object(live_auth, "invoke", return_value=(auth, 0, "")) as invoke, contextlib.redirect_stdout(io.StringIO()):
                code = live_auth.main(["--allow-live-read", "--binary", __file__, "--report", str(report)])
            self.assertEqual(code, 2)
            self.assertEqual(invoke.call_count, 1)
            evidence = json.loads(report.read_text())
            self.assertEqual(evidence["tests"][1]["status"], "skip")


if __name__ == "__main__":
    unittest.main()
