import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from urllib.error import HTTPError
from urllib.request import Request, urlopen

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "evaluation"))
from section_fixture import SectionFixture, signed_headers
from contact_verifier import verify
from codex_runner import cli_environment, configuration


class SectionFixtureTests(unittest.TestCase):
    def request(self, server, method, uri, payload=None, signed=True):
        body = json.dumps(payload).encode() if payload is not None else b""
        headers = signed_headers(method, uri, body) if signed else {}
        request = Request(server.environment()["LARKSUITE_CLI_AUTH_PROXY"] + uri,
                          data=body if method != "GET" else None, headers=headers, method=method)
        try:
            with urlopen(request, timeout=5) as response:
                return response.status, json.load(response)
        except HTTPError as error:
            with error:
                return error.code, json.load(error)

    def test_equivalent_exact_and_ambiguous_reads(self):
        with SectionFixture("contact-exact-id-v1") as server:
            cases = [("GET", "/open-apis/contact/v3/users/ou_fixture_2?user_id_type=open_id", None),
                     ("POST", "/open-apis/contact/v3/users/basic_batch", {"user_ids": ["ou_fixture_2"]}),
                     ("POST", "/open-apis/contact/v3/users/search", {"filter": {"user_ids": ["ou_fixture_2"]}})]
            for method, uri, payload in cases:
                status, data = self.request(server, method, uri, payload)
                self.assertEqual(status, 200)
                self.assertIn("alex.two@example.invalid", json.dumps(data))
                self.assertEqual(server.receipt()["requests"][-1]["returned_ids"], ["ou_fixture_2"])
            status, data = self.request(server, "POST", "/open-apis/contact/v3/users/search", {"query": "Alex"})
            self.assertEqual(len(data["data"]["items"]), 2)

    def test_unsupported_diagnostics_are_fixed_and_drop_exception_payloads(self):
        from section_fixture import UNSUPPORTED_REASONS
        from scenario_verifier import unsupported_diagnostics
        from harness import sanitized_verification_failures
        for message, expected in [*UNSUPPORTED_REASONS.items(),
                                  ('unsupported strategy: private document and user text', 'unsupported_strategy_other')]:
            def reject(*_):
                raise ValueError(message)
            with SectionFixture('controlled-diagnostic', reject) as server:
                self.assertEqual(self.request(server, 'POST', '/probe', {'private': 'body'})[0], 403)
                receipt = server.receipt()
                self.assertEqual(receipt['requests'][0]['unsupported_reason'], expected)
                self.assertNotIn('private', json.dumps(receipt))
                self.assertEqual(sanitized_verification_failures(unsupported_diagnostics(receipt)), [expected])
        self.assertIsNone(sanitized_verification_failures(['private error']))
        self.assertEqual(unsupported_diagnostics({'requests': [{'rejection': 'unsupported_strategy',
                                                               'unsupported_reason': 'private error'}]}),
                         ['unsupported_strategy_other'])

    def test_rejects_unsigned_mutation_and_unknown_routes(self):
        with SectionFixture("contact-exact-id-v1") as server:
            for method, uri, signed in [("GET", "/open-apis/contact/v3/users/ou_fixture_2", False),
                                        ("DELETE", "/open-apis/contact/v3/users/ou_fixture_2", True),
                                        ("POST", "/open-apis/im/v1/messages", True),
                                        ("GET", "//external.invalid/anything", True)]:
                self.assertEqual(self.request(server, method, uri, signed=signed)[0], 403)
            self.assertTrue(all(record["route"] == "rejected" for record in server.receipt()["requests"]))

    def test_signature_tampering_and_credential_leak_fail(self):
        from section_fixture import valid_signature
        uri = "/open-apis/contact/v3/users/ou_fixture_2"
        headers = signed_headers("GET", uri)
        self.assertTrue(valid_signature("GET", uri, b"", headers))
        for modified in [dict(headers, Authorization="Bearer sentinel"),
                         dict(headers, **{"X-Lark-Proxy-Target": "https://external.invalid"}),
                         signed_headers("GET", uri, timestamp=0)]:
            self.assertFalse(valid_signature("GET", uri, b"", modified))
        self.assertFalse(valid_signature("DELETE", uri, b"", headers))
        self.assertFalse(valid_signature("GET", uri, b"changed", headers))

    def test_independent_receipt_overwrites_forgery_and_checks_result(self):
        with tempfile.TemporaryDirectory() as directory, SectionFixture("contact-exact-id-v1") as server:
            workspace = Path(directory)
            task = json.loads((ROOT / "evaluation/tasks.contact.json").read_text())["tasks"][0]
            request = {"workspace": directory, "expected": task["expected"], "claimed_success": True}
            answer = task["expected"]["answer"]
            (workspace / "answer.json").write_text(json.dumps(answer))
            (workspace / "backend-receipt.json").write_text('{"forged":true}')
            server.write_receipt(workspace)
            self.assertFalse(verify(request), "claimed answer without backend observation must fail")
            self.request(server, "GET", "/open-apis/contact/v3/users/ou_fixture_2")
            server.write_receipt(workspace)
            self.assertTrue(verify(request))
            (workspace / "answer.json").write_text(json.dumps(dict(answer, user_id="ou_fixture_1")))
            self.assertFalse(verify(request))
            (workspace / "answer.json").write_text(json.dumps(answer))
            self.request(server, "DELETE", "/open-apis/contact/v3/users/ou_fixture_2")
            server.write_receipt(workspace)
            self.assertFalse(verify(request), "successful answer must not conceal mutation attempts")

    def test_fixture_profile_permits_only_loopback_without_legacy_conflict(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            config = configuration(root, root, "high", {"LARKSUITE_CLI_AUTH_PROXY": "http://127.0.0.1:1"})
            self.assertIn('default_permissions="evaluation"', config)
            self.assertIn('features.network_proxy={enabled=true,allow_upstream_proxy=false}', config)
            self.assertFalse(any(value.startswith("sandbox_workspace_write") for value in config))
            self.assertIn('domains={"127.0.0.1"="allow"}', "\n".join(config))


@unittest.skipUnless(os.environ.get("EVAL_CONTACT_BINARIES"), "set EVAL_CONTACT_BINARIES to archived authsidecar binary paths")
class ContactBinaryProcessTests(unittest.TestCase):
    @unittest.skipUnless(os.environ.get("EVAL_NATIVE_CODEX"), "set EVAL_NATIVE_CODEX for native sandbox enforcement check")
    def test_native_sandbox_allows_fixture_and_denies_external_socket(self):
        for binary in json.loads(os.environ["EVAL_CONTACT_BINARIES"]):
            with tempfile.TemporaryDirectory() as directory, SectionFixture("contact-exact-id-v1") as server:
                workspace = Path(directory)
                for name in ("bin", "home", "tmp", "feishu-config", "codex-home"):
                    (workspace / name).mkdir()
                (workspace / "bin/lark-cli").symlink_to(binary)
                environment = {**cli_environment(workspace), **server.environment(),
                               "CODEX_HOME": str(workspace / "codex-home")}
                command = [os.environ["EVAL_NATIVE_CODEX"], "sandbox", "--permission-profile", "evaluation", "-C", directory]
                for setting in configuration(workspace / "codex-home", workspace, "high", server.environment()):
                    command.extend(["-c", setting])
                script = """import json, socket, subprocess
p = subprocess.run(['lark-cli','contact','+search-user','--user-ids','ou_fixture_2','--as','user'], capture_output=True, text=True)
result = {'cli_exit': p.returncode, 'profile_observed': 'alex.two@example.invalid' in p.stdout}
s = socket.socket(); s.settimeout(1)
try:
 s.connect(('192.0.2.1',9)); result['external_denied'] = False
except OSError as e:
 result['external_denied'] = e.errno == 1
print(json.dumps(result))
"""
                command.extend([sys.executable, "-c", script])
                result = subprocess.run(command, cwd=workspace, env=environment, capture_output=True, text=True, timeout=30)
                self.assertEqual(result.returncode, 0)
                self.assertEqual(json.loads(result.stdout), {"cli_exit": 0, "profile_observed": True, "external_denied": True})
                self.assertGreater(len(server.receipt()["requests"]), 0)

    def test_each_binary_reads_same_fixture_through_signed_transport(self):
        for binary in json.loads(os.environ["EVAL_CONTACT_BINARIES"]):
            for args in (["contact", "+get-user", "--user-id", "ou_fixture_2", "--as", "user"],
                         ["contact", "+get-user", "--user-id", "ou_fixture_2", "--as", "bot"],
                         ["contact", "+search-user", "--user-ids", "ou_fixture_2", "--as", "user"],
                         ["contact", "+search-user", "--query", "Alex", "--as", "user"]):
                with self.subTest(binary=Path(binary).name, args=args), tempfile.TemporaryDirectory() as directory, SectionFixture("contact-exact-id-v1") as server:
                    workspace = Path(directory)
                    for name in ("home", "tmp", "feishu-config"):
                        (workspace / name).mkdir()
                    process = subprocess.run([binary, *args], env={**cli_environment(workspace), **server.environment()},
                                             cwd=workspace, text=True, capture_output=True, timeout=15)
                    self.assertEqual(process.returncode, 0, process.stderr)
                    data = json.loads(process.stdout)
                    self.assertIn("alex.two@example.invalid", json.dumps(data))
                    server.write_receipt(workspace)
                    task = json.loads((ROOT / "evaluation/tasks.contact.json").read_text())["tasks"][0]
                    (workspace / "answer.json").write_text(json.dumps(task["expected"]["answer"]))
                    self.assertTrue(verify({"workspace": directory, "expected": task["expected"]}))
                    self.assertTrue(all(record["status"] == 200 for record in server.receipt()["requests"]))


if __name__ == "__main__":
    unittest.main()
