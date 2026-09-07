"""Controlled Base fixture/verifier tests; no models or live backend access."""
from copy import deepcopy
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "evaluation"))
from scenarios.base import Scenario, TASK, SCENARIO, PREFIX, TARGET, OTHER, INITIAL, FINAL, verify
from section_fixture import signed_headers


class BaseScenarioTests(unittest.TestCase):
    def setUp(self):
        self.scenario = Scenario()
        self.requests = []

    def call(self, method, path, data=None, identity="user"):
        body = json.dumps(data).encode() if data is not None else b""
        reply = self.scenario.dispatch(method, path, body, signed_headers(method, path, body, identity))
        self.requests.append({"route": reply.route, "status": reply.status, "returned_ids": [], "proof": reply.proof})
        return reply.json["data"]

    def receipt(self):
        return {"scenario": SCENARIO, "requests": deepcopy(self.requests), "state": self.scenario.snapshot()}

    def accepted(self, answer=None, receipt=None):
        return verify(Path("/unused"), TASK["expected"]["answer"] if answer is None else answer,
                      self.receipt() if receipt is None else receipt, TASK["expected"])

    def test_patch_numeric_zero_empty_and_native_preservation(self):
        self.call("PATCH", PREFIX + "/records/" + TARGET, {"fld_amount": 0, "fld_text": ""})
        self.assertTrue(self.accepted())
        current = self.call("POST", PREFIX + "/records/batch_get", {"record_id_list": [TARGET]})
        self.assertEqual(current["data"], [[0, "", True, INITIAL[TARGET]["fld_attachment"]]])
        self.assertIs(current["data"][0][2], True)
        self.assertEqual(self.scenario.snapshot()["records"], FINAL)
        self.assertTrue(self.accepted())

    def test_equivalent_batch_name_keys_and_split_delta_updates(self):
        self.call("POST", PREFIX + "/records/batch_update", {"update_records": {TARGET: {"Amount": 0}}})
        self.call("PATCH", PREFIX + "/records/" + TARGET, {"Text": "", "Checked": True, "Attachment": INITIAL[TARGET]["fld_attachment"]})
        self.assertTrue(self.accepted())
        self.call("PATCH", PREFIX + "/records/" + TARGET, {"fld_amount": 0.0, "fld_text": ""})
        self.assertTrue(self.accepted(), "equivalent numeric spelling and unchanged repeat are valid")

    def test_projection_pagination_fields_and_missing_rows_are_real(self):
        page = self.call("GET", PREFIX + "/records?offset=0&limit=1&field_id=Amount&field_id=Text")
        self.assertEqual(page["record_id_list"], [TARGET])
        self.assertEqual(page["data"], [[8, "old"]])
        self.assertIs(page["has_more"], True)
        page = self.call("GET", PREFIX + "/records?offset=1&limit=1")
        self.assertEqual(page["record_id_list"], [OTHER])
        self.assertIs(page["has_more"], False)
        missing = self.call("POST", PREFIX + "/records/batch_get", {"record_id_list": ["rec_missing"], "select_fields": ["fld_checked"]})
        self.assertEqual(missing["record_not_found"], ["rec_missing"])
        self.assertEqual(missing["data"], [])
        fields = self.call("GET", PREFIX + "/fields?offset=1&limit=2")
        self.assertEqual([f["id"] for f in fields["items"]], ["fld_text", "fld_checked"])
        self.assertIs(fields["has_more"], True)
        self.assertEqual(self.call("GET", PREFIX + "/fields/fld_amount")["type"], "number")
        self.assertFalse(self.accepted(), "reads and a correct claimed answer are insufficient")

    def test_invalid_or_unintended_writes_are_atomic_rejections(self):
        cases = [
            ("PATCH", PREFIX + "/records/" + OTHER, {"fld_amount": 0}),
            ("PATCH", PREFIX + "/records/" + TARGET, {"fld_amount": 0, "fld_checked": False}),
            ("PATCH", PREFIX + "/records/" + TARGET, {"fld_attachment": []}),
            ("PATCH", PREFIX + "/records/" + TARGET, {"fld_amount": "0"}),
            ("PATCH", PREFIX + "/records/" + TARGET, {"fld_amount": False}),
            ("PATCH", PREFIX + "/records/" + TARGET, {"fields": {"fld_amount": 0}}),
            ("PATCH", PREFIX + "/records/" + TARGET, {"Amount": 0, "fld_amount": 0}),
            ("POST", PREFIX + "/records/batch_update", {"update_records": {TARGET: {"fld_amount": 0}, OTHER: {"fld_text": ""}}}),
            ("DELETE", PREFIX + "/records/" + TARGET, None),
            ("POST", PREFIX + "/records", {"fld_amount": 0}),
            ("PATCH", PREFIX.replace("bas_eval_base_dev_01", "bas_foreign") + "/records/" + TARGET, {"fld_amount": 0}),
        ]
        for method, path, data in cases:
            with self.subTest(method=method, path=path, data=data), self.assertRaises(ValueError):
                self.call(method, path, data)
            self.assertEqual(self.scenario.snapshot()["records"], INITIAL)
            self.assertEqual(self.scenario.snapshot()["effects"], [])
        with self.assertRaises(ValueError):
            self.call("PATCH", PREFIX + "/records/" + TARGET, {"fld_amount": 0}, identity="bot")

    def test_unknown_query_duplicate_json_and_nonfinite_are_rejected(self):
        for path in (PREFIX + "/records?offset=-1", PREFIX + "/records?limit=0", PREFIX + "/records?limit=2&limit=1", PREFIX + "/records?view_id=foreign", PREFIX + "/fields/unknown"):
            with self.subTest(path=path), self.assertRaises(ValueError):
                self.call("GET", path)
        for raw in (b'{"fld_amount":0,"fld_amount":9}', b'{"fld_amount":NaN}', b'{"fld_amount":Infinity}', b'{"fld_amount":'+b'9'*400+b'}'):
            path = PREFIX + "/records/" + TARGET
            with self.assertRaises(ValueError):
                self.scenario.dispatch("PATCH", path, raw, signed_headers("PATCH", path, raw))

    def test_null_omission_and_transient_wrong_updates_cannot_pass(self):
        self.call("PATCH", PREFIX + "/records/" + TARGET, {"fld_amount": 0})
        self.assertFalse(self.accepted(), "omitted text is not an empty string")
        self.call("PATCH", PREFIX + "/records/" + TARGET, {"fld_text": None})
        self.assertIsNone(self.scenario.snapshot()["records"][TARGET]["fld_text"])
        self.assertFalse(self.accepted(), "null is not an empty string")
        self.call("PATCH", PREFIX + "/records/" + TARGET, {"fld_text": ""})
        self.assertFalse(self.accepted(), "restoration cannot conceal an unintended clear")

    def test_answer_and_server_evidence_tampering_fail(self):
        self.call("PATCH", PREFIX + "/records/" + TARGET, {"fld_amount": 0, "fld_text": ""})
        valid = self.receipt()
        self.assertTrue(self.accepted(receipt=valid))
        for key, value in (("amount", False), ("amount", "0"), ("text", None), ("record_id", OTHER)):
            answer = dict(TASK["expected"]["answer"], **{key: value})
            self.assertFalse(self.accepted(answer=answer))
        for modify in (
            lambda r: r["state"]["records"][TARGET].update(fld_checked=1),
            lambda r: r["state"]["records"][TARGET]["fld_attachment"][0].update(size=13),
            lambda r: r["state"]["records"][OTHER].update(fld_text=""),
            lambda r: r["state"]["records"].pop(OTHER),
            lambda r: r["state"]["fields"][0].update(type="text"),
            lambda r: r.update(requests=[]),
            lambda r: r["requests"][0].update(proof={}),
            lambda r: r["requests"].append({"route": "rejected", "status": 403}),
            lambda r: r["state"].update(effects=[]),
        ):
            receipt = deepcopy(valid)
            modify(receipt)
            self.assertFalse(self.accepted(receipt=receipt))
        with tempfile.TemporaryDirectory() as directory:
            workspace = Path(directory)
            (workspace / "backend-receipt.json").write_text(json.dumps(valid))
            (workspace / "answer.json").write_text(json.dumps(TASK["expected"]["answer"]))
            self.assertFalse(verify(workspace, TASK["expected"]["answer"], {"scenario": SCENARIO, "requests": [], "state": Scenario().snapshot()}, TASK["expected"]))

    def test_public_schema_has_no_hidden_answers_or_state(self):
        fixture = TASK["fixture"]
        self.assertEqual(set(fixture), {"scenario", "allow_synthetic_mutations", "answer_schema"})
        encoded = json.dumps(fixture)
        for forbidden in ('"const"', '"enum"', TARGET, "box_fixture_file", "unrelated"):
            self.assertNotIn(forbidden, encoded)
        snapshot = self.scenario.snapshot()
        snapshot["records"][TARGET]["fld_attachment"].clear()
        self.assertEqual(self.scenario.snapshot()["records"], INITIAL)


@unittest.skipUnless(os.environ.get("EVAL_BASE_BINARIES"), "set EVAL_BASE_BINARIES to clean archived authsidecar binaries")
class BaseBinaryTests(unittest.TestCase):
    def test_each_binary_updates_native_state(self):
        from section_fixture import SectionFixture
        from codex_runner import cli_environment
        for binary in json.loads(os.environ["EVAL_BASE_BINARIES"]):
            for strategy in ("single", "batch"):
                scenario = Scenario()
                # The shared server owns signatures and observation; adapting Reply
                # here keeps checks compatible with its initial tuple API.
                proofs = []
                def dispatch(method, uri, body, headers):
                    reply = scenario.dispatch(method, uri, body, headers)
                    proofs.append(deepcopy(reply.proof))
                    return reply.route, reply.json, reply.proof.get("record_ids", [])
                with self.subTest(binary=Path(binary).name, strategy=strategy), tempfile.TemporaryDirectory() as directory, SectionFixture(SCENARIO, dispatch=dispatch) as server:
                    workspace = Path(directory)
                    for name in ("home", "tmp", "feishu-config"):
                        (workspace / name).mkdir()
                    flags = ["--base-token", "bas_eval_base_dev_01", "--table-id", "tbl_eval_base_dev_01", "--as", "user"]
                    if strategy == "single":
                        args = ["+record-upsert", "--record-id", TARGET, "--json", '{"fld_amount":0,"fld_text":""}']
                    else:
                        args = ["+record-batch-update", "--json", json.dumps({"update_records": {TARGET: {"Amount": 0, "Text": ""}}})]
                    process = subprocess.run([binary, "base", *args, *flags], cwd=workspace, env={**cli_environment(workspace), **server.environment()}, capture_output=True, text=True, timeout=20)
                    # The native batch endpoint has no item-existence proof; the
                    # candidate may truthfully report unknown until readback.
                    self.assertIn(process.returncode, (0,) if strategy == "single" else (0, 22), process.stderr)
                    read = subprocess.run([binary, "base", "+record-get", "--record-id", TARGET, "--format", "json", *flags], cwd=workspace, env={**cli_environment(workspace), **server.environment()}, capture_output=True, text=True, timeout=20)
                    self.assertEqual(read.returncode, 0, read.stderr)
                    self.assertIn(TARGET, read.stdout)
                    receipt = server.receipt()
                    self.assertEqual(len(receipt["requests"]), len(proofs))
                    for record, proof in zip(receipt["requests"], proofs):
                        record["proof"] = proof
                    receipt["state"] = scenario.snapshot()
                    self.assertTrue(verify(workspace, TASK["expected"]["answer"], receipt, TASK["expected"]))


if __name__ == "__main__":
    unittest.main()
