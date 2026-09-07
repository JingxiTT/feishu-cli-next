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

    def test_equivalent_native_run_segmentation_is_not_a_hidden_requirement(self):
        elements = []
        for run in docs.INITIAL["blocks"][1]["runs"]:
            text = "New " if run["text"] == "Old " else run["text"]
            style = {"bold": run["style"]["bold"]}
            if run["style"]["link"]:
                style["link"] = {"url": run["style"]["link"]["url"]}
            elements.extend({"text_run": {"content": char, "text_element_style": style}} for char in text)
        self.call("PATCH", docs.DOCX_PATH + "/blocks/" + docs.TARGET, {"update_text_elements": {"elements": elements}})
        self.assertTrue(docs.verify(None, docs.EXPECTED_ANSWER, self.receipt(), docs.TASK["expected"]))

    def test_inline_xml_preserves_native_semantics_across_markup_order_and_entities(self):
        for content in ['New <a href="synthetic://reference"><b>status</b></a>',
                        '<span>New </span><b><a href="synthetic://reference">sta&#116;us</a></b>']:
            with self.subTest(content=content):
                self.setUp()
                self.edit(content=content)
                self.assertTrue(docs.verify(None, docs.EXPECTED_ANSWER, self.receipt(), docs.TASK["expected"]))

    def test_explicit_xml_wrong_styles_are_applied_and_fail_verification(self):
        for content in ['<span>New status</span>', 'New <b>status</b>',
                        'New <a href="synthetic://wrong"><b>status</b></a>',
                        '<b>New </b><a href="synthetic://reference"><b>status</b></a>']:
            with self.subTest(content=content):
                self.setUp()
                self.edit(content=content)
                self.assertEqual(len(self.scenario.mutations), 1)
                self.assertFalse(docs.verify(None, docs.EXPECTED_ANSWER, self.receipt(), docs.TASK["expected"]))

    def test_block_replace_changes_only_target_identity_and_new_id_is_readable(self):
        for addressing in [{"block_id": docs.TARGET},
                           {"start_block_id": docs.TARGET, "end_block_id": docs.TARGET}]:
            with self.subTest(addressing=addressing):
                self.setUp()
                self.call("PUT", docs.AI_PATH, {"command": "block_replace", "format": "xml", **addressing,
                    "content": '<p>New <b><a href="synthetic://reference">status</a></b> — 你好</p>'})
                final = self.scenario.snapshot()["final"]
                new_id = final["blocks"][1]["block_id"]
                self.assertNotEqual(new_id, docs.TARGET)
                self.assertEqual(final["block_order"][1], new_id)
                self.assertEqual(final["blocks"][2:], docs.INITIAL["blocks"][2:])
                self.assertIn(new_id, self.call("POST", docs.AI_PATH + "/fetch", {"format": "xml", "export_option": {"export_block_id": True}})["document"]["content"])
                self.assertEqual(self.call("GET", docs.DOCX_PATH + "/blocks/" + new_id)["block"]["block_id"], new_id)
                with self.assertRaises(ValueError):
                    self.call("GET", docs.DOCX_PATH + "/blocks/" + docs.TARGET)
                self.assertTrue(docs.verify(None, docs.EXPECTED_ANSWER, self.receipt(), docs.TASK["expected"]))

    def test_bad_block_replacement_content_is_not_repaired(self):
        for content in ['<p>New status — 你好</p>',
                        '<p>New <a href="synthetic://reference"><b>status</b></a> — 你坏</p>']:
            with self.subTest(content=content):
                self.setUp()
                self.call("PUT", docs.AI_PATH, {"command": "block_replace", "block_id": docs.TARGET,
                    "content": content})
                self.assertFalse(docs.verify(None, docs.EXPECTED_ANSWER, self.receipt(), docs.TASK["expected"]))

    def test_unmodeled_xml_and_reconstruction_reject_without_effect(self):
        for content in ['<em>New status</em>', '<span bold="true">New status</span>',
                        '<a href="x" title="unknown">New status</a>', '<b>unclosed',
                        '<!DOCTYPE x [<!ENTITY y "New">]>&y; status',
                        '<span>&unknown;</span>', '<p>New status</p>']:
            with self.subTest(content=content):
                before = self.scenario.snapshot()
                with self.assertRaises(ValueError):
                    self.edit(content=content)
                self.assertEqual(before, self.scenario.snapshot())
        for content in ["<span>" * 34 + "New status" + "</span>" * 34, "x" * 65537]:
            with self.assertRaises(ValueError):
                self.edit(content=content)
        for command in ["overwrite", "block_insert_after"]:
            with self.assertRaises(ValueError):
                self.edit(command=command)

    def test_xml_entities_literal_text_and_explicit_noop(self):
        self.edit(content='Old <b><a href="synthetic://reference">status</a></b>')
        self.assertEqual(self.scenario.snapshot()["mutations"], [])
        self.edit(pattern="Old", content="A &amp; B &lt; C &gt; D")
        self.assertEqual(self.scenario.state["blocks"][1]["text"], "A & B < C > D status — 你好")
        self.assertFalse(docs.verify(None, docs.EXPECTED_ANSWER, self.receipt(), docs.TASK["expected"]))

    def test_replaced_paragraph_supports_followup_edit_and_rejects_stale_write(self):
        self.call("PUT", docs.AI_PATH, {"command": "block_replace", "block_id": docs.TARGET,
            "content": '<p block-id="doxcnEvalTargetBlock">Temporary <b><a href="synthetic://reference">status</a></b> — 你好</p>'})
        before = self.scenario.snapshot()
        with self.assertRaises(ValueError):
            self.call("PUT", docs.AI_PATH, {"command": "block_replace", "block_id": docs.TARGET,
                "content": "<p>stale</p>"})
        self.assertEqual(before, self.scenario.snapshot())
        self.edit(pattern="Temporary status")
        self.assertTrue(docs.verify(None, docs.EXPECTED_ANSWER, self.receipt(), docs.TASK["expected"]))

    def test_other_block_identity_resource_and_state_remain_required(self):
        # _commit creates internally consistent signed-style proofs, so these
        # failures exercise task invariants rather than only hash tamper checks.
        for mutate in [lambda s: s["blocks"][2].update(block_id="replacementOther"),
                       lambda s: s["blocks"][3].update(token="replacementToken"),
                       lambda s: s["blocks"][3].update(width=1),
                       lambda s: s["permissions"].update(public=True),
                       lambda s: s["blocks"].reverse()]:
            with self.subTest(mutate=mutate):
                self.setUp()
                self.edit()
                candidate = copy.deepcopy(self.scenario.state)
                mutate(candidate)
                proof = self.scenario._commit(candidate, "docs_update")
                self.requests.append({"route": "docs_update", "status": 200, "proof": proof})
                self.assertFalse(docs.verify(None, docs.EXPECTED_ANSWER, self.receipt(), docs.TASK["expected"]))

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
            for strategy in ("shortcut", "native_patch", "inline_xml", "block_replace"):
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
                    elif strategy == "inline_xml":
                        write = [binary, "docs", "+update", "--doc", docs.DOCUMENT, "--command", "str_replace", "--pattern", "Old status", "--content", 'New <a href="synthetic://reference"><b>status</b></a>', "--doc-format", "xml", "--revision-id", "7", "--as", "user"]
                    elif strategy == "block_replace":
                        write = [binary, "docs", "+update", "--doc", docs.DOCUMENT, "--command", "block_replace", "--block-id", docs.TARGET, "--content", '<p>New <a href="synthetic://reference"><b>status</b></a> — 你好</p>', "--doc-format", "xml", "--revision-id", "7", "--as", "user"]
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
