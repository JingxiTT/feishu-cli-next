"""Paired Contact/discovery controls; no provider or live service calls."""
from copy import deepcopy
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from urllib.parse import urlsplit

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "evaluation"))
from paired import contact, discovery
from codex_runner import cli_environment
from scenario_fixture import ScenarioFixture
from section_fixture import signed_headers


class ContactTests(unittest.TestCase):
    def setUp(self):
        self.backend, self.requests = contact.Scenario(), []

    def call(self, method, uri, body=None, actor="user"):
        raw = json.dumps(body).encode() if body is not None else b""
        reply = self.backend.dispatch(method, uri, raw, signed_headers(method, uri, raw, identity=actor))
        self.requests.append({"route": reply.route, "status": reply.status, "signature_valid": True,
                              "method": method, "uri": urlsplit(uri).path, "proof": deepcopy(reply.proof)})
        return reply

    def receipt(self):
        return {"scenario": contact.SCENARIO, "requests": deepcopy(self.requests), "state": self.backend.snapshot()}

    def success(self):
        self.call("POST", contact.USER_SEARCH, {"query": "alex.two@example.invalid"})
        self.call("POST", contact.BOT_SEARCH, {"query": "Relay"})

    def verify(self, receipt=None, answer=None):
        return contact.verify(None, contact.ANSWER if answer is None else answer,
                              self.receipt() if receipt is None else receipt, {})

    def test_signed_evidence_not_answer_guess_and_ambiguous_names(self):
        self.assertFalse(self.verify())
        user = self.call("POST", contact.USER_SEARCH, {"query": "Alex"})
        bots = self.call("POST", contact.BOT_SEARCH, {"query": "Relay"})
        self.assertEqual(len(user.json["data"]["items"]), 2)
        self.assertEqual(len(bots.json["data"]["items"]), 3)
        self.assertTrue(self.verify())
        answer = dict(contact.ANSWER, bot_id=contact.BOTS[0]["open_id"])
        self.assertFalse(self.verify(answer=answer))
        answer = dict(contact.ANSWER, user_id=contact.USERS[0]["open_id"])
        self.assertFalse(self.verify(answer=answer))

    def test_get_and_batch_are_equivalent_user_identity_proofs(self):
        for method, uri, data in (("GET", contact.GET_BASE + contact.USER_ID, None),
                                  ("POST", contact.USER_BATCH, {"user_ids": [contact.USER_ID]})):
            with self.subTest(method=method):
                self.setUp()
                self.call(method, uri, data, "bot")
                self.call("POST", contact.BOT_SEARCH, {"query": "Quartz incident relay"})
                self.assertTrue(self.verify())

    def test_pages_do_not_claim_unseen_identity(self):
        self.call("POST", contact.USER_SEARCH, {"query": "alex.two@example.invalid"})
        first = self.call("POST", contact.BOT_SEARCH + "?page_size=1", {"query": "Relay"})
        self.assertTrue(first.json["data"]["has_more"])
        self.assertFalse(self.verify())
        token = first.json["data"]["page_token"]
        second = self.call("POST", contact.BOT_SEARCH + "?page_size=1&page_token=" + token, {"query": "Relay"})
        self.assertTrue(second.json["data"]["has_more"])
        self.assertTrue(self.verify())  # Exact discriminators; no forced full enumeration.

    def test_unsigned_mutated_or_mislabelled_proof_fails(self):
        self.success()
        for field, value in (("signature_valid", False), ("status", 403), ("method", "DELETE"), ("uri", "/foreign")):
            r = self.receipt(); r["requests"][0][field] = value
            self.assertFalse(self.verify(receipt=r))
        r = self.receipt(); r["requests"][0]["proof"]["records"] = []
        self.assertFalse(self.verify(receipt=r))
        r = self.receipt(); r["state"]["bots"][1]["tenant_id"] = "other"
        self.assertFalse(self.verify(receipt=r))
        r = self.receipt(); r["state"]["read_count"] = True
        self.assertFalse(self.verify(receipt=r))

    def test_filters_apply_or_are_explicitly_unsupported(self):
        for value, count in ((True, 2), (False, 0)):
            response = self.call("POST", contact.USER_SEARCH, {"query": "Alex", "filter": {"has_enterprise_email": value}})
            self.assertEqual(len(response.json["data"]["items"]), count)
        for key in ("has_contact", "exclude_outer_contact", "is_resigned"):
            with self.assertRaisesRegex(ValueError, "unsupported strategy"):
                self.call("POST", contact.USER_SEARCH, {"query": "Alex", "filter": {key: True}})
        with self.assertRaisesRegex(ValueError, "unsupported strategy"):
            self.call("POST", contact.BOT_SEARCH, {"query": "Relay", "filter": {"has_chatter": True}})

    def test_strict_routes_identity_and_selectors(self):
        cases = [("DELETE", contact.GET_BASE + contact.USER_ID, None),
                 ("POST", contact.USER_SEARCH + "?page_size=0", {"query": "Alex"}),
                 ("POST", contact.BOT_SEARCH + "?page_size=31", {"query": "Relay"}),
                 ("POST", contact.USER_SEARCH + "?page_size=1&page_size=2", {"query": "Alex"}),
                 ("POST", contact.USER_SEARCH + "?page_token=wrong", {"query": "Alex"}),
                 ("GET", "https://example.invalid" + contact.GET_BASE + contact.USER_ID, None)]
        for method, uri, body in cases:
            with self.subTest(uri=uri), self.assertRaises(ValueError):
                self.call(method, uri, body)
        for route in (contact.USER_SEARCH, contact.BOT_SEARCH):
            with self.assertRaisesRegex(ValueError, "user identity"):
                self.call("POST", route, {"query": "Alex"}, "bot")

    def test_public_fixture_contains_types_not_hidden_identity_answers(self):
        for module in (contact, discovery):
            data = json.dumps(module.TASK["fixture"])
            self.assertNotIn(contact.USER_ID, data)
            self.assertNotIn(contact.BOTS[1]["open_id"], data)
            self.assertNotIn('"const"', data)
            self.assertNotIn('"enum"', data)


class DiscoveryTests(unittest.TestCase):
    def test_probe_allowlist_rejects_executables_shell_auth_paths_and_mutations(self):
        good = ["contact", "+get-user", "--user-id", contact.USER_ID]
        self.assertTrue(discovery.safe_argv(good))
        self.assertTrue(discovery.safe_argv(["api", "GET", contact.GET_BASE + contact.USER_ID, "--as=bot"]))
        for args in (["/bin/sh", "-c", "true"], ["auth", "logout"], good + ["--profile", "x"],
                     good + ["--save-result", "/tmp/result"], good + ["--data", "@file"],
                     ["api", "DELETE", contact.GET_BASE + contact.USER_ID],
                     ["api", "GET", "https://example.invalid" + contact.GET_BASE + contact.USER_ID],
                     good + [";", "touch", "file"], good + ["--params", "-"],
                     good + ["--as", "user", "--as", "bot"]):
            with self.subTest(args=args):
                self.assertFalse(discovery.safe_argv(args))

    def test_discovery_allows_bootstrap_but_not_business_read(self):
        backend = discovery.Scenario()
        result = backend.dispatch("GET", "/open-apis/authen/v1/user_info", b"", {})
        self.assertEqual(result.route, "identity")
        with self.assertRaises(ValueError):
            backend.dispatch("GET", contact.GET_BASE + contact.USER_ID, b"", {})
        self.assertFalse(discovery.verify(None, {"argv": ["contact", "+get-user", "--user-id", contact.USER_ID]},
                                         {"scenario": discovery.SCENARIO, "requests": [], "state": backend.snapshot()}, {}))


class ControlledBinaries(unittest.TestCase):
    def test_both_archived_binaries_contact_and_discovery(self):
        binaries = [ROOT / ".tmp/luna-medium-preparation" / (arm + "-cli") for arm in ("official", "next")]
        if os.environ.get("EVAL_PAIRED_CONTACT_BINARIES"):
            binaries = [Path(path) for path in json.loads(os.environ["EVAL_PAIRED_CONTACT_BINARIES"])]
        if not all(p.is_file() for p in binaries):
            self.skipTest("requires supplied archived paired controlled binaries")
        for binary in binaries:
            historical_next = hashlib.sha256(binary.read_bytes()).hexdigest() == "a96268761560efa996ba7ce99c231423de14277cc814f694c18b9b5cbb7a7a48"
            with self.subTest(arm=binary.name), tempfile.TemporaryDirectory() as directory:
                w = Path(directory)
                for name in ("home", "tmp", "bin", "feishu-config"):
                    (w / name).mkdir()
                backend = contact.Scenario()
                with ScenarioFixture(contact.SCENARIO, backend) as server:
                    env = {**cli_environment(w), **server.environment()}
                    for argv in (["contact", "+search-user", "--query", "alex.two@example.invalid"],
                                 ["contact", "+search-bot", "--query", "Relay"]):
                        r = subprocess.run([str(binary), *argv], cwd=w, env=env, capture_output=True, text=True, timeout=20)
                        self.assertEqual(r.returncode, 0, r.stderr)
                    self.assertTrue(contact.verify(w, contact.ANSWER, server.receipt(), {}))
                alternatives = [
                    ["contact", "+get-user", "--user-id", contact.USER_ID],
                    ["contact", "+get-user", "--user-id", contact.USER_ID, "--as", "bot"],
                    ["contact", "+search-user", "--user-ids", contact.USER_ID],
                    ["contact", "user_profiles", "batch_query", "--user-id-type", "open_id", "--data", json.dumps({"user_ids": [contact.USER_ID]})],
                    ["api", "GET", contact.GET_BASE + contact.USER_ID, "--as", "bot"],
                    ["api", "POST", contact.USER_BATCH, "--params", '{"user_id_type":"open_id"}', "--data", json.dumps({"user_ids": [contact.USER_ID]})],
                ]
                for argv in alternatives:
                    with self.subTest(argv=argv):
                        answer = {"argv": argv}; (w / "answer.json").write_text(json.dumps(answer))
                        d = discovery.Scenario(); d.selected_cli = str(binary); d.setup(w); d.close()
                        receipt = {"scenario": discovery.SCENARIO, "requests": [], "state": d.snapshot()}
                        # Preserve the supplied Next binary's known raw read-POST
                        # exit22 as a failed alternative, never a verifier success.
                        if argv[:3] == ["api", "POST", contact.USER_BATCH] and historical_next and d.probe.get("exit_code") == 22:
                            self.assertFalse(discovery.verify(w, answer, receipt, {}))
                            self.assertIn(contact.USERS[1], contact.observed_records(d.probe["receipt"])["users"])
                            continue
                        self.assertTrue(discovery.verify(w, answer, receipt, {}), d.snapshot())
                        altered = deepcopy(receipt); altered["state"]["verifier_only_probe"]["receipt"]["requests"] = []
                        self.assertFalse(discovery.verify(w, answer, altered, {}))
                        altered = deepcopy(receipt); altered["requests"] = [{"route": "contact_get", "status": 200, "signature_valid": True}]
                        self.assertFalse(discovery.verify(w, answer, altered, {}))
                        altered = deepcopy(receipt); altered["state"]["verifier_only_probe"]["target_in_output"] = False
                        self.assertFalse(discovery.verify(w, answer, altered, {}))


if __name__ == "__main__":
    unittest.main()
