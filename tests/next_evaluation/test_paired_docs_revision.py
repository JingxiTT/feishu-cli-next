"""Historical native Docs requests stay inconclusive without masking bad input."""
import copy
import json
from pathlib import Path
import sys
import tempfile
import unittest
from urllib.error import HTTPError
from urllib.request import Request, urlopen

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / "evaluation"))
from paired import docs_edit, docs_read
from paired.docs_revision import UNSUPPORTED_HISTORY
from scenarios import docs as native
from scenario_fixture import ScenarioFixture
from scenario_verifier import evaluate
from section_fixture import signed_headers
from harness import sanitized_verification_failures

LIST = native.DOCX_PATH + "/blocks"
PATCH = LIST + "/" + native.TARGET
HEADERS = {"X-Lark-Proxy-Identity": "user"}


def body(scenario, text="New "):
    elements = native.wire_block(scenario.state["blocks"][1])["text"]["elements"]
    elements[0]["text_run"]["content"] = text
    return json.dumps({"update_text_elements": {"elements": elements}}).encode()


class DocsRevisionTests(unittest.TestCase):
    def test_current_and_latest_reads_and_edits_keep_native_proofs(self):
        for revision in (str(native.INITIAL["revision_id"]), "-1"):
            for module in (docs_read, docs_edit):
                with self.subTest(revision=revision, module=module.__name__):
                    scenario = module.Scenario()
                    reply = scenario.dispatch("GET", LIST + "?document_revision_id=" + revision, b"{}", HEADERS)
                    self.assertEqual(reply.route, "docs_blocks")
                    self.assertEqual(scenario.mutations, [])
                    if module is docs_read:
                        self.assertTrue(reply.proof["fact_and_block_returned"])
            scenario = docs_edit.Scenario()
            reply = scenario.dispatch("PATCH", PATCH + "?document_revision_id=" + revision, body(scenario), HEADERS)
            receipt = {"scenario": docs_edit.TASK["fixture"]["scenario"], "state": scenario.snapshot(),
                       "requests": [{"route": reply.route, "status": reply.status, "proof": reply.proof}]}
            self.assertTrue(docs_edit.verify(None, native.EXPECTED_ANSWER, receipt, {}))

    def test_valid_historical_requests_leave_real_state_and_proofs_unchanged(self):
        for module in (docs_read, docs_edit):
            scenario = module.Scenario()
            initial_revision = scenario.state["revision_id"]
            if module is docs_edit:
                scenario.dispatch("PATCH", PATCH, body(scenario), HEADERS)
            old_revision = initial_revision if module is docs_edit else initial_revision - 1
            before = scenario.snapshot()
            cases = [("GET", LIST, b"{}")]
            if module is docs_edit:
                cases.append(("PATCH", PATCH, body(scenario, "Stale ")))
            for method, path, raw in cases:
                with self.subTest(module=module.__name__, method=method), self.assertRaises(ValueError) as caught:
                    scenario.dispatch(method, path + "?document_revision_id=" + str(old_revision), raw, HEADERS)
                self.assertEqual(str(caught.exception), UNSUPPORTED_HISTORY)
                self.assertEqual(scenario.snapshot(), before)

    def test_bad_historical_requests_remain_failures_before_any_mutation(self):
        scenario = docs_edit.Scenario()
        old = str(scenario.state["revision_id"] - 1)
        suffix = "?document_revision_id=" + old
        cases = [
            ("GET", LIST + suffix + "&page_size=501", b"{}", HEADERS),
            ("GET", LIST + suffix + "&page_token=999", b"{}", HEADERS),
            ("GET", LIST + suffix + "&unknown=1", b"{}", HEADERS),
            ("GET", LIST + suffix + "&document_revision_id=" + old, b"{}", HEADERS),
            ("GET", LIST + "?document_revision_id=invalid", b"{}", HEADERS),
            ("GET", LIST + "?document_revision_id=-2", b"{}", HEADERS),
            ("GET", LIST + "?document_revision_id=99999", b"{}", HEADERS),
            ("GET", LIST + suffix, b"[]", HEADERS),
            ("GET", LIST + suffix, b"{", HEADERS),
            ("GET", LIST + suffix, b"{}", {}),
            ("PATCH", LIST + "/unknown" + suffix, body(scenario), HEADERS),
            ("PATCH", PATCH + suffix, b"{}", HEADERS),
            ("PATCH", PATCH + suffix, b'{"update_text_elements":{"elements":"bad"}}', HEADERS),
            ("PATCH", PATCH + suffix, b'{"update_text_elements":{"elements":[{"text_run":{"content":"x","text_element_style":{"bold":"bad"}}}]}}', HEADERS),
            ("PATCH", PATCH + suffix, b'{"update_text_elements":{"elements":[{"text_run":{"content":"x","text_element_style":{"link":{"url":1}}}}]}}', HEADERS),
            ("GET", "/foreign/blocks" + suffix, b"{}", HEADERS),
        ]
        before = scenario.snapshot()
        for method, path, raw, headers in cases:
            with self.subTest(method=method, path=path, raw=raw), self.assertRaises(ValueError) as caught:
                scenario.dispatch(method, path, raw, headers)
            self.assertFalse(str(caught.exception).startswith("unsupported strategy:"))
            self.assertEqual(scenario.snapshot(), before)
        with self.assertRaisesRegex(ValueError, "read-only"):
            docs_read.Scenario().dispatch("PATCH", PATCH + suffix, body(scenario), HEADERS)

    def test_docs_ai_and_historical_original_scenario_are_unchanged(self):
        old = native.INITIAL["revision_id"] - 1
        for module in (docs_edit, docs_read):
            with self.assertRaisesRegex(ValueError, "synthetic revision conflict"):
                module.Scenario().dispatch("POST", native.AI_PATH + "/fetch",
                                           json.dumps({"revision_id": old}).encode(), HEADERS)
        with self.assertRaisesRegex(ValueError, "synthetic revision conflict"):
            native.Scenario().dispatch("GET", LIST + "?document_revision_id=" + str(old), b"{}", HEADERS)

    def test_signed_receipt_classifies_limitation_and_retains_known_failure(self):
        code = "unsupported_docs_historical_revision"
        self.assertEqual(sanitized_verification_failures([code]), [code])
        for module in (docs_read, docs_edit):
            scenario = module.Scenario()
            before = copy.deepcopy(scenario.snapshot())
            path = LIST if module is docs_read else PATCH
            method = "GET" if module is docs_read else "PATCH"
            raw = b"{}" if module is docs_read else body(scenario)
            uri = path + "?document_revision_id=" + str(scenario.state["revision_id"] - 1)
            with tempfile.TemporaryDirectory() as directory, ScenarioFixture(module.TASK["fixture"]["scenario"], scenario) as server:
                url = server.environment()["LARKSUITE_CLI_AUTH_PROXY"]
                with self.assertRaises(HTTPError) as caught:
                    urlopen(Request(url + uri, data=raw, method=method, headers=signed_headers(method, uri, raw)))
                self.assertEqual(caught.exception.code, 403)
                caught.exception.close()
                receipt = server.receipt()
                self.assertEqual(receipt["state"], before)
                self.assertEqual(receipt["requests"][0]["unsupported_reason"], code)
                self.assertNotIn("document_revision_id", json.dumps(receipt["requests"]))
                workspace = Path(directory)
                (workspace / "answer.json").write_text("{}")
                request = {"workspace": directory, "task_id": module.TASK["id"],
                           "fixture": module.TASK["fixture"], "expected": module.TASK["expected"]}
                (workspace / "backend-receipt.json").write_text(json.dumps(receipt))
                result = evaluate(request)
                self.assertFalse(result["success"])
                self.assertTrue(result["fixture_unsupported"])
                self.assertEqual(result["verification_failures"], ["request_outside_fixture", code])
                # A malformed attempt remains a known failure even alongside history.
                bad_uri = uri + "&unknown=1"
                with self.assertRaises(HTTPError) as caught:
                    urlopen(Request(url + bad_uri, data=raw, method=method, headers=signed_headers(method, bad_uri, raw)))
                caught.exception.close()
                (workspace / "backend-receipt.json").write_text(json.dumps(server.receipt()))
                self.assertFalse(evaluate(request)["fixture_unsupported"])
                self.assertEqual(scenario.snapshot(), before)


if __name__ == "__main__":
    unittest.main()
