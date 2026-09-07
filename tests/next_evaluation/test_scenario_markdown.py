"""No live access/model calls; direct synthetic transitions and hostile artifacts."""
import copy
import json
from pathlib import Path
import sys
import tempfile
import unittest

EVALUATION = Path(__file__).resolve().parents[2] / "evaluation"
sys.path.insert(0, str(EVALUATION))
from scenarios import markdown as scenario
from section_fixture import signed_headers


class MarkdownScenarioTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.workspace = Path(self.temporary.name)
        self.backend = scenario.Scenario()
        self.backend.setup(self.workspace)
        self.requests = []
        self.answer = {"document_id": scenario.DOCUMENT_ID,
                       "artifact_path": "converted.xml", "unsupported": []}

    def dispatch(self, method, path, body=None, identity="user"):
        raw = json.dumps(body or {}, ensure_ascii=False).encode()
        reply = self.backend.dispatch(method, path, raw, signed_headers(method, path, raw, identity=identity))
        self.requests.append({"route": reply.route, "status": reply.status,
                              "returned_ids": [], "proof": reply.proof})
        return reply

    def successful(self, options=None):
        self.dispatch("POST", scenario.CREATE_PATH, {"format": "markdown", "content": scenario.SOURCE})
        reply = self.dispatch("POST", scenario.FETCH_PATH, {
            "format": "xml", "export_option": options or {"export_block_id": True}})
        (self.workspace / "converted.xml").write_text(reply.json["data"]["document"]["content"], encoding="utf-8")
        return {"scenario": scenario.SCENARIO, "state": self.backend.snapshot(), "requests": self.requests}

    def check(self, receipt, answer=None):
        return scenario.verify(self.workspace, self.answer if answer is None else answer,
                               receipt, scenario.TASK["expected"])

    def test_predeclared_source_and_public_schema(self):
        tasks = json.loads((EVALUATION / "tasks.development.json").read_text())
        declared = next(t for t in tasks["tasks"] if t["id"] == scenario.TASK["id"])
        self.assertEqual(declared["fixture"]["markdown"].encode(), (self.workspace / "input.md").read_bytes())
        self.assertEqual({"scenario", "answer_schema", "allow_synthetic_mutations"}, set(scenario.TASK["fixture"]))
        self.assertNotIn(scenario.DOCUMENT_ID, json.dumps(scenario.TASK))
        self.assertNotIn(scenario.NATIVE_XML, json.dumps(scenario.TASK))

    def test_create_fetch_and_independent_native_structure(self):
        receipt = self.successful()
        self.assertTrue(self.check(receipt))
        # Equivalent XML serialization is valid without relying on exact output bytes.
        path = self.workspace / "converted.xml"
        path.write_text(scenario.NATIVE_XML.replace("<p></p>", "<p />") + "\n", encoding="utf-8")
        self.assertTrue(self.check(receipt))
        self.dispatch("POST", scenario.FETCH_PATH, {"format": "xml", "export_option": {
            "export_block_id": True, "export_style_attrs": True, "export_cite_extra_data": True}})
        self.assertTrue(self.check({"scenario": scenario.SCENARIO,
                                   "state": self.backend.snapshot(), "requests": self.requests}))

    def test_foreign_identity_id_payload_and_effects_rejected(self):
        cases = [
            ("POST", scenario.CREATE_PATH, {"format": "markdown", "content": scenario.SOURCE}, "bot"),
            ("POST", scenario.CREATE_PATH, {"format": "xml", "content": scenario.NATIVE_XML}, "user"),
            ("POST", scenario.CREATE_PATH, {"format": "markdown", "content": scenario.SOURCE + " "}, "user"),
            ("POST", scenario.CREATE_PATH, {"format": "markdown", "content": scenario.SOURCE, "parent_token": "foreign"}, "user"),
            ("POST", scenario.FETCH_PATH, {"format": "xml"}, "user"),
            ("POST", scenario.CREATE_PATH + "/foreign/fetch", {"format": "xml"}, "user"),
            ("PUT", scenario.CREATE_PATH + "/foreign", {}, "user"),
            ("POST", "/open-apis/drive/v1/files/upload_all", {}, "user"),
            ("POST", scenario.CREATE_PATH + "?share=true", {}, "user"),
        ]
        for args in cases:
            with self.subTest(args=args), self.assertRaises(ValueError):
                self.dispatch(*args)
        self.assertFalse(self.backend.snapshot()["final"]["created"])
        self.assertEqual(0, self.backend.snapshot()["final"]["create_attempts"])

    def test_duplicate_create_partial_fetch_and_bad_types_rejected(self):
        self.successful()
        with self.assertRaises(ValueError):
            self.dispatch("POST", scenario.CREATE_PATH, {"format": "markdown", "content": scenario.SOURCE})
        for extra in ({"read_option": {"read_mode": "outline"}}, {"revision_id": True},
                      {"revision_id": 2}, {"export_option": {"export_block_id": 1}}):
            with self.subTest(extra=extra), self.assertRaises(ValueError):
                self.dispatch("POST", scenario.FETCH_PATH, {"format": "xml", **extra})
        with self.assertRaises(ValueError):
            self.backend.dispatch("POST", scenario.CREATE_PATH, b'{"content":"a","content":"b"}',
                                  {"X-Lark-Proxy-Identity": "user"})

    def test_forged_claim_without_backend_evidence_fails(self):
        (self.workspace / "converted.xml").write_text(scenario.NATIVE_XML, encoding="utf-8")
        self.assertFalse(self.check({"scenario": scenario.SCENARIO, "state": self.backend.snapshot(), "requests": []}))
        receipt = self.successful()
        for altered in ({}, {**receipt, "requests": []}, {**receipt, "state": {}}, {**receipt, "scenario": "other"}):
            self.assertFalse(self.check(altered))
        altered = copy.deepcopy(receipt)
        altered["requests"][-1]["proof"]["native_sha256"] = "forged"
        self.assertFalse(self.check(altered))
        altered = copy.deepcopy(receipt)
        altered["requests"].append({"route": "permission_grant", "status": 200})
        self.assertFalse(self.check(altered))
        for answer in ({**self.answer, "document_id": "foreign"}, {**self.answer, "unsupported": ["invented"]},
                       {**self.answer, "artifact_path": "../converted.xml"}):
            self.assertFalse(self.check(receipt, answer))

    def test_empty_cell_unicode_structure_and_ids_cannot_be_forged(self):
        receipt = self.successful()
        wrong = [scenario.SOURCE, scenario.NATIVE_XML.replace("你好", "??"),
                 scenario.NATIVE_XML.replace('<td block-id="cell_empty"><p></p></td>', ""),
                 scenario.NATIVE_XML.replace("<p></p>", "<p> </p>"),
                 scenario.NATIVE_XML.replace("<p></p>", "<p>null</p>"),
                 scenario.NATIVE_XML.replace("cell_empty", "cell_other"),
                 scenario.NATIVE_XML + "unrelated text", '<!DOCTYPE a [<!ENTITY b "x">]><a>&b;</a>']
        for content in wrong:
            with self.subTest(content=content):
                (self.workspace / "converted.xml").write_text(content, encoding="utf-8")
                self.assertFalse(self.check(receipt))
        (self.workspace / "converted.xml").write_text(scenario.NATIVE_XML, encoding="utf-8")
        (self.workspace / "input.md").write_text("changed", encoding="utf-8")
        self.assertFalse(self.check(receipt))

    def test_simple_fetch_cannot_prove_observed_native_ids(self):
        self.dispatch("POST", scenario.CREATE_PATH, {"format": "markdown", "content": scenario.SOURCE})
        reply = self.dispatch("POST", scenario.FETCH_PATH, {"format": "xml"})
        self.assertNotIn("block-id", reply.json["data"]["document"]["content"])
        (self.workspace / "converted.xml").write_text(scenario.NATIVE_XML, encoding="utf-8")
        self.assertFalse(self.check({"scenario": scenario.SCENARIO,
                                    "state": self.backend.snapshot(), "requests": self.requests}))

    def test_unsupported_conversion_preserves_failure_and_prevents_replay(self):
        self.backend = scenario.Scenario(unsupported=True)
        reply = self.dispatch("POST", scenario.CREATE_PATH, {"format": "markdown", "content": scenario.SOURCE})
        self.assertEqual("failed", reply.json["data"]["result"])
        self.assertEqual(["synthetic Markdown table conversion unsupported"], reply.json["data"]["warnings"])
        self.assertFalse(self.backend.snapshot()["final"]["created"])
        with self.assertRaises(ValueError):
            self.dispatch("POST", scenario.CREATE_PATH, {"format": "markdown", "content": scenario.SOURCE})
        with self.assertRaises(ValueError):
            self.dispatch("POST", scenario.FETCH_PATH, {"format": "xml"})

    def test_documented_drive_import_routes_are_an_adapter_gap(self):
        routes = [("POST", path) for path in scenario.DRIVE_IMPORT_POST_PATHS]
        routes.append(("GET", "/open-apis/drive/v1/import_tasks/tk_import"))
        for method, path in routes:
            with self.subTest(method=method, path=path):
                # Real upload routes must classify before parsing multipart as JSON.
                raw = b"--synthetic-boundary\r\nContent-Disposition: form-data\r\n"
                with self.assertRaisesRegex(ValueError, "^unsupported strategy: Drive import not modeled$"):
                    self.backend.dispatch(method, path, raw, signed_headers(method, path, raw))
        self.assertEqual(0, self.backend.snapshot()["final"]["create_attempts"])

    def test_foreign_and_malformed_mutation_paths_are_not_strategy_gaps(self):
        routes = [
            ("PUT", "/open-apis/drive/v1/import_tasks"),
            ("GET", "/open-apis/drive/v1/import_tasks"),
            ("POST", "/open-apis/drive/v1/import_tasks/tk_import"),
            ("POST", "/open-apis/drive/v1/import_tasks/"),
            ("POST", "/open-apis/drive/v1/medias/upload_all/extra"),
            ("POST", "/open-apis/drive/v1/medias/upload_all?share=true"),
            ("POST", "/open-apis/drive/v1/files/upload_all"),
            ("POST", "/open-apis/drive/v1/permissions/docxForeign/members"),
            ("GET", "/open-apis/drive/v1/import_tasks/../foreign"),
            ("GET", "/open-apis/drive/v1/import_tasks/.."),
            ("GET", "/open-apis/drive/v1/import_tasks/%2Fforeign"),
        ]
        for method, path in routes:
            with self.subTest(method=method, path=path), self.assertRaises(ValueError) as error:
                self.dispatch(method, path)
            self.assertNotIn("unsupported strategy:", str(error.exception))

    def test_snapshot_cannot_mutate_server_state(self):
        state = self.backend.snapshot()
        state["final"]["created"] = True
        self.assertFalse(self.backend.snapshot()["final"]["created"])


if __name__ == "__main__":
    unittest.main()
