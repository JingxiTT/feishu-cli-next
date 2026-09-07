"""State and fidelity checks for the Docs development adapter; no model calls."""
import copy
import json
import os
import subprocess
import tempfile
from pathlib import Path
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / "evaluation"))
from scenarios import docs


class DocsScenarioTests(unittest.TestCase):
    def setUp(self):
        self.scenario = docs.Scenario()
        self.requests = []

    def call(self, method, suffix, data=None, identity="user"):
        reply = self.scenario.dispatch(method, suffix, json.dumps(data or {}).encode(), {"X-Lark-Proxy-Identity": identity})
        self.requests.append({"route": reply.route, "status": reply.status, "proof": reply.proof})
        return reply.json["data"]

    def receipt(self):
        return {"scenario": docs.TASK["fixture"]["scenario"], "requests": self.requests, "state": self.scenario.snapshot()}

    def edit(self, **overrides):
        data = {"command": "str_replace", "format": "xml", "pattern": "Old status", "content": "New status", "revision_id": -1}
        data.update(overrides)
        return self.call("PUT", docs.AI_PATH, data)

    def test_public_task_contains_no_expected_state_or_answer_values(self):
        fixture = docs.TASK["fixture"]
        self.assertEqual(set(fixture), {"scenario", "answer_schema", "allow_synthetic_mutations"})
        self.assertNotIn("const", json.dumps(fixture))
        self.assertNotIn("enum", json.dumps(fixture))
        self.assertNotIn("New status", json.dumps(fixture))

    def test_native_read_str_replace_read_preserves_runs(self):
        initial = self.scenario.snapshot()["final"]
        read = self.call("POST", docs.AI_PATH + "/fetch", {"format": "xml", "export_option": {"export_block_id": True, "export_style_attrs": True}})
        self.assertIn('href="synthetic://reference"', read["document"]["content"])
        self.edit()
        final = self.scenario.snapshot()["final"]
        self.assertEqual(final["blocks"][1]["runs"][1:], initial["blocks"][1]["runs"][1:])
        self.assertEqual(final["blocks"][1]["text"], "New status — 你好")
        self.assertEqual(final["blocks"][2:], initial["blocks"][2:])
        self.assertTrue(docs.verify(None, docs.EXPECTED_ANSWER, self.receipt(), docs.TASK["expected"]))

    def test_equivalent_docx_patch_is_checked_independently(self):
        block = self.call("GET", docs.DOCX_PATH + "/blocks/" + docs.TARGET)["block"]
        elements = block["text"]["elements"]
        elements[0]["text_run"]["content"] = "New "
        self.call("PATCH", docs.DOCX_PATH + "/blocks/" + docs.TARGET, {"update_text_elements": {"elements": elements}})
        self.assertTrue(docs.verify(None, docs.EXPECTED_ANSWER, self.receipt(), docs.TASK["expected"]))

    def test_wrong_answer_and_success_claim_without_effect_fail(self):
        self.assertFalse(docs.verify(None, docs.EXPECTED_ANSWER, self.receipt(), docs.TASK["expected"]))
        self.edit()
        answer = dict(docs.EXPECTED_ANSWER, paragraph_text="wrong")
        self.assertFalse(docs.verify(None, answer, self.receipt(), docs.TASK["expected"]))
        receipt = self.receipt()
        receipt["requests"] = []
        self.assertFalse(docs.verify(None, docs.EXPECTED_ANSWER, receipt, docs.TASK["expected"]))

    def test_tampered_final_state_or_mutation_proof_fails(self):
        self.edit()
        for key in ["final", "initial"]:
            receipt = self.receipt()
            receipt["state"][key]["blocks"][1]["runs"][1]["style"]["bold"] = False
            self.assertFalse(docs.verify(None, docs.EXPECTED_ANSWER, receipt, docs.TASK["expected"]))
        receipt = self.receipt()
        receipt["state"]["mutations"][0]["after_hash"] = "forged"
        self.assertFalse(docs.verify(None, docs.EXPECTED_ANSWER, receipt, docs.TASK["expected"]))

    def test_missing_styles_are_not_restored_by_adapter(self):
        self.call("PATCH", docs.DOCX_PATH + "/blocks/" + docs.TARGET, {"update_text_elements": {"elements": [{"text_run": {"content": "New status — 你好"}}]}})
        self.assertFalse(docs.verify(None, docs.EXPECTED_ANSWER, self.receipt(), docs.TASK["expected"]))
        self.assertFalse(self.scenario.snapshot()["final"]["blocks"][1]["runs"][0]["style"]["bold"])

    def test_unrelated_change_then_restore_is_not_hidden(self):
        self.call("PATCH", docs.DOCX_PATH + "/blocks/" + docs.UNRELATED, {"update_text_elements": {"elements": [{"text_run": {"content": "temporary"}}]}})
        self.call("PATCH", docs.DOCX_PATH + "/blocks/" + docs.UNRELATED, {"update_text_elements": {"elements": [{"text_run": {"content": "Keep résumé 😀 and 001 unchanged."}}]}})
        self.edit()
        self.assertFalse(docs.verify(None, docs.EXPECTED_ANSWER, self.receipt(), docs.TASK["expected"]))

    def test_rejected_foreign_ids_identity_and_revision_do_not_mutate(self):
        before = self.scenario.snapshot()
        cases = [("PUT", docs.AI_PATH + "Foreign", {"command": "str_replace"}, "user"),
                 ("PUT", docs.AI_PATH, {"command": "str_replace"}, "unknown"),
                 ("PUT", docs.AI_PATH, {"command": "str_replace", "format": "xml", "pattern": "Old status", "content": "New status", "revision_id": 6}, "user"),
                 ("PATCH", docs.DOCX_PATH + "/blocks/foreign", {"update_text_elements": {"elements": []}}, "bot")]
        for method, path, data, identity in cases:
            with self.assertRaises(ValueError):
                self.call(method, path, data, identity)
            self.assertEqual(before, self.scenario.snapshot())

    def test_noop_and_intermediate_valid_edits_do_not_force_a_sequence(self):
        self.edit(content="Old status")
        self.assertEqual(self.scenario.snapshot()["mutations"], [])
        self.edit(content="Temporary status")
        self.edit(pattern="Temporary status")
        self.assertTrue(docs.verify(None, docs.EXPECTED_ANSWER, self.receipt(), docs.TASK["expected"]))

    def test_read_scopes_mark_partial_and_render_current_state(self):
        scoped = self.call("POST", docs.AI_PATH + "/fetch", {"format": "xml", "read_option": {"read_mode": "range", "start_block_id": docs.TARGET, "end_block_id": docs.TARGET}})
        self.assertFalse(scoped["complete"])
        self.assertIn("<fragment", scoped["document"]["content"])
        self.assertNotIn("Keep résumé", scoped["document"]["content"])
        self.edit()
        markdown = self.call("POST", docs.AI_PATH + "/fetch", {"format": "markdown", "read_option": {"read_mode": "keyword", "keyword": "New|absent"}})
        self.assertIn("New [**status**](synthetic://reference) — 你好", markdown["document"]["content"])
        page = self.call("GET", docs.DOCX_PATH + "/blocks?page_size=2")
        self.assertTrue(page["has_more"])
        second = self.call("GET", docs.DOCX_PATH + "/blocks?page_size=2&page_token=" + page["page_token"])
        self.assertNotEqual(page["items"][0]["block_id"], second["items"][0]["block_id"])

    def test_ambiguous_phrase_is_rejected_without_effect(self):
        self.scenario.state["blocks"][2]["text"] = "Old status"
        before = self.scenario.snapshot()
        with self.assertRaises(ValueError):
            self.edit()
        self.assertEqual(before, self.scenario.snapshot())

    def test_model_written_receipt_cannot_substitute_for_server_state(self):
        self.edit()
        forged = self.receipt()
        fresh = docs.Scenario()
        with tempfile.TemporaryDirectory() as directory:
            workspace = Path(directory)
            (workspace / "backend-receipt.json").write_text(json.dumps(forged))
            actual = {"scenario": docs.TASK["fixture"]["scenario"], "requests": [], "state": fresh.snapshot()}
            self.assertFalse(docs.verify(workspace, docs.EXPECTED_ANSWER, actual, docs.TASK["expected"]))

    def test_invalid_body_empty_pattern_and_missing_match_are_rejected(self):
        before = self.scenario.snapshot()
        for payload in [b"null", b"[]", b"{} {}", b'{"command":"str_replace","command":"overwrite"}', b'{"revision_id":NaN}']:
            with self.assertRaises(ValueError):
                self.scenario.dispatch("PUT", docs.AI_PATH, payload, {"X-Lark-Proxy-Identity": "user"})
        for pattern in ["", "not found"]:
            with self.assertRaises(ValueError):
                self.edit(pattern=pattern)
        self.assertEqual(before, self.scenario.snapshot())


@unittest.skipUnless(os.environ.get("EVAL_DOCS_BINARIES"), "set EVAL_DOCS_BINARIES to clean archived authsidecar binaries")
class DocsBinaryTests(unittest.TestCase):
    def test_official_and_next_equivalent_strategies(self):
        from section_fixture import SectionFixture
        from codex_runner import cli_environment
        for binary in json.loads(os.environ["EVAL_DOCS_BINARIES"]):
            for strategy in ("shortcut", "native_patch"):
                scenario = docs.Scenario()
                proofs = []
                def bridge(method, uri, body, headers):
                    reply = scenario.dispatch(method, uri, body, headers)
                    proofs.append(copy.deepcopy(reply.proof))
                    return reply.route, reply.json, [docs.DOCUMENT]
                with self.subTest(binary=Path(binary).name, strategy=strategy), tempfile.TemporaryDirectory() as directory, SectionFixture(docs.TASK["fixture"]["scenario"], dispatch=bridge) as server:
                    workspace = Path(directory)
                    for name in ("home", "tmp", "feishu-config"):
                        (workspace / name).mkdir()
                    env = {**cli_environment(workspace), **server.environment()}
                    read = [binary, "docs", "+fetch", "--doc", docs.DOCUMENT, "--detail", "full", "--as", "user"]
                    before = subprocess.run(read, cwd=workspace, env=env, capture_output=True, text=True, timeout=20)
                    self.assertEqual(before.returncode, 0, before.stderr)
                    self.assertIn("Old", before.stdout)
                    if strategy == "shortcut":
                        write = [binary, "docs", "+update", "--doc", docs.DOCUMENT, "--command", "str_replace", "--pattern", "Old status", "--content", "New status", "--as", "user"]
                    else:
                        elements = docs.wire_block(scenario.snapshot()["initial"]["blocks"][1])["text"]["elements"]
                        elements[0]["text_run"]["content"] = "New "
                        write = [binary, "api", "PATCH", docs.DOCX_PATH + "/blocks/" + docs.TARGET, "--data", json.dumps({"update_text_elements": {"elements": elements}}, ensure_ascii=False), "--as", "user"]
                    edited = subprocess.run(write, cwd=workspace, env=env, capture_output=True, text=True, timeout=20)
                    self.assertEqual(edited.returncode, 0, edited.stderr)
                    after = subprocess.run(read, cwd=workspace, env=env, capture_output=True, text=True, timeout=20)
                    self.assertEqual(after.returncode, 0, after.stderr)
                    self.assertIn("New", after.stdout)
                    receipt = server.receipt()
                    self.assertEqual(len(receipt["requests"]), len(proofs))
                    self.assertTrue(all(record["status"] == 200 for record in receipt["requests"]))
                    for record, proof in zip(receipt["requests"], proofs):
                        record["proof"] = proof
                    receipt["state"] = scenario.snapshot()
                    self.assertTrue(docs.verify(workspace, docs.EXPECTED_ANSWER, receipt, docs.TASK["expected"]))


if __name__ == "__main__":
    unittest.main()
