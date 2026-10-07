# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Independent paired Base controls; no models, live APIs or old fixture edits."""
from copy import deepcopy
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
from paired import base
from section_fixture import SectionFixture, signed_headers


class PairedBaseTests(unittest.TestCase):
    def setUp(self):
        self.scenario, self.requests = base.Scenario(), []

    def call(self, method, uri, data=None, identity="user"):
        body = json.dumps(data).encode() if data is not None else b""
        reply = self.scenario.dispatch(method, uri, body, signed_headers(method, uri, body, identity))
        self.requests.append({"route": reply.route, "status": reply.status, "proof": deepcopy(reply.proof),
                              "signature_valid": True, "method": method, "uri": urlsplit(uri).path})
        return reply.json["data"]

    def receipt(self):
        return {"scenario": base.SCENARIO, "requests": deepcopy(self.requests), "state": self.scenario.snapshot()}

    def accepted(self, receipt=None, answer=None):
        return base.verify(Path("/unused"), base.TASK["expected"]["answer"] if answer is None else answer,
                           self.receipt() if receipt is None else receipt, base.TASK["expected"])

    def update(self):
        self.call("PATCH", base.PREFIX + "/records/" + base.TARGET, {"fld_amount": 0, "fld_text": None})

    def test_exact_zero_null_and_untouched_native_state(self):
        self.update()
        self.assertTrue(self.accepted())
        data = self.call("POST", base.PREFIX + "/records/batch_get", {"record_id_list": [base.TARGET, base.OTHER]})
        self.assertEqual(data["data"][0], [0, None, True, base.INITIAL[base.TARGET]["fld_attachment"]])
        self.assertEqual(self.scenario.snapshot()["records"][base.OTHER], base.INITIAL[base.OTHER])
        self.assertTrue(self.accepted())

    def test_equivalent_single_batch_name_id_and_split_deltas(self):
        self.call("POST", base.PREFIX + "/records/batch_update", {"update_records": {base.TARGET: {"Amount": 0.0}}})
        self.call("PATCH", base.PREFIX + "/records/" + base.TARGET,
                  {"Text": None, "Checked": True, "Attachment": base.INITIAL[base.TARGET]["fld_attachment"]})
        self.assertTrue(self.accepted())
        self.update()
        self.assertTrue(self.accepted(), "an unchanged repeated delta is still a valid strategy")

    def test_empty_string_update_is_unsupported_before_mutation(self):
        for method, path, payload in [
            ("PATCH", base.PREFIX + "/records/" + base.TARGET, {"Amount": 0, "Text": ""}),
            ("POST", base.PREFIX + "/records/batch_update", {"update_records": {base.TARGET: {"fld_amount": 0, "fld_text": ""}}}),
        ]:
            with self.subTest(method=method), self.assertRaisesRegex(ValueError, "^unsupported strategy:.*empty-text update"):
                self.call(method, path, payload)
            self.assertEqual(self.scenario.snapshot()["records"], base.INITIAL)
            self.assertEqual(self.scenario.snapshot()["effects"], [])
        self.update()
        self.assertTrue(self.accepted(), "explicit null remains the supported requested operation")

    def test_omission_wrong_number_and_transient_updates_fail(self):
        self.call("PATCH", base.PREFIX + "/records/" + base.TARGET, {"Amount": 0})
        self.assertFalse(self.accepted(), "text omission is not explicit clearing")
        self.call("PATCH", base.PREFIX + "/records/" + base.TARGET, {"Amount": 9})
        self.update()
        self.assertFalse(self.accepted(), "restored transient wrong values still fail")

    def test_preservation_foreign_identity_and_invalid_mutations_are_atomic(self):
        cases = [{"Amount": False}, {"Amount": "0"}, {"Checked": False}, {"Attachment": []},
                 {"Amount": 0, "fld_amount": 0}, {"Text": []}]
        for patch in cases:
            with self.subTest(patch=patch), self.assertRaises(ValueError):
                self.call("PATCH", base.PREFIX + "/records/" + base.TARGET, patch)
            self.assertEqual(self.scenario.snapshot()["records"], base.INITIAL)
        for method, path, payload, identity in [
            ("PATCH", base.PREFIX + "/records/" + base.OTHER, {"Amount": 0}, "user"),
            ("PATCH", base.PREFIX + "/records/" + base.TARGET, {"Amount": 0}, "bot"),
            ("DELETE", base.PREFIX + "/records/" + base.TARGET, None, "user"),
            ("POST", base.PREFIX + "/records", {"Amount": 0}, "user")]:
            with self.assertRaises(ValueError):
                self.call(method, path, payload, identity)
        self.assertEqual(self.scenario.snapshot()["effects"], [])

    def test_native_metadata_and_projected_paginated_reads(self):
        self.assertEqual(self.call("GET", base.BASE_PREFIX)["base_token"], base.BASE)
        self.assertEqual(self.call("GET", base.PREFIX)["id"], base.TABLE)
        self.assertEqual(self.call("GET", base.BASE_PREFIX + "/tables?offset=0&limit=1")["tables"], [base.TABLE_META])
        self.assertEqual(self.call("GET", base.PREFIX + "/views")["views"], [base.VIEW])
        self.assertEqual(self.call("GET", base.PREFIX + "/views/" + base.VIEW["id"]), base.VIEW)
        self.assertEqual(len(self.call("GET", base.PREFIX + "/fields")["fields"]), 4)
        self.assertEqual(self.call("GET", base.PREFIX + "/fields/fld_text")["type"], "text")
        first = self.call("GET", base.PREFIX + "/records?limit=1&field_id=Amount&view_id=" + base.VIEW["id"])
        self.assertEqual(first["data"], [[8]])
        self.assertTrue(first["has_more"])
        self.assertFalse(self.accepted(), "correct answer without a mutation is insufficient")
        self.update()
        self.assertTrue(self.accepted())

    def test_metadata_selector_and_duplicate_json_rejections(self):
        for uri in [base.PREFIX + "?unknown=1", base.PREFIX + "/views?limit=1&limit=2",
                    base.PREFIX + "/views/foreign", base.PREFIX + "/records?view_id=foreign",
                    base.BASE_PREFIX.replace(base.BASE, "foreign")]:
            with self.subTest(uri=uri), self.assertRaises(ValueError):
                self.call("GET", uri)
        for raw in [b'{"Amount":0,"Amount":8}', b'{"Amount":NaN}']:
            uri = base.PREFIX + "/records/" + base.TARGET
            with self.assertRaises(ValueError):
                self.scenario.dispatch("PATCH", uri, raw, signed_headers("PATCH", uri, raw))

    def test_signed_effect_and_complete_snapshot_tampering_fail(self):
        self.update()
        good = self.receipt()
        changes = [lambda r: r["requests"][0].update(signature_valid=False),
                   lambda r: r["requests"][0].update(uri=base.PREFIX + "/records/" + base.OTHER),
                   lambda r: r["requests"][0].update(method="GET"),
                   lambda r: r["requests"][0].update(proof={}),
                   lambda r: r.update(requests=[]),
                   lambda r: r["state"]["records"][base.TARGET].update(fld_checked=1),
                   lambda r: r["state"]["records"][base.TARGET]["fld_attachment"][0].update(size=99),
                   lambda r: r["state"]["records"][base.OTHER].update(fld_text="changed"),
                   lambda r: r["state"]["records"].pop(base.OTHER),
                   lambda r: r["state"]["fields"][0].update(type="text"),
                   lambda r: r["state"]["table"].update(name="changed"),
                   lambda r: r["state"]["views"][0].update(name="changed")]
        for change in changes:
            receipt = deepcopy(good)
            change(receipt)
            self.assertFalse(self.accepted(receipt), change)
        for key, value in (("amount", False), ("amount", "0"), ("text", ""), ("record_id", base.OTHER)):
            self.assertFalse(self.accepted(answer=dict(base.TASK["expected"]["answer"], **{key: value})))
        self.assertFalse(self.accepted({"scenario": base.SCENARIO, "state": base.Scenario().snapshot(), "requests": []}))

    def test_public_fixture_contains_type_schema_only(self):
        self.assertEqual(base.TASK["id"], "paired-base")
        self.assertEqual(base.TASK["fixture"]["scenario_section"], "paired-base")
        raw = json.dumps(base.TASK["fixture"])
        for forbidden in [base.TARGET, '"const"', '"enum"', "box_fixture_file", "unrelated"]:
            self.assertNotIn(forbidden, raw)
        self.assertEqual(base.INITIAL, base.native.INITIAL)
        self.assertEqual(base.native.TASK["expected"]["answer"]["text"], "", "historical fixture unchanged")


@unittest.skipUnless(os.environ.get("EVAL_PAIRED_BASE_BINARIES"), "requires supplied clean archived binaries")
class PairedBaseBinaryTests(unittest.TestCase):
    def test_clean_binaries_metadata_single_batch_and_raw_api(self):
        from codex_runner import cli_environment
        for binary in json.loads(os.environ["EVAL_PAIRED_BASE_BINARIES"]):
            for strategy in ("single", "batch", "raw-api"):
                scenario = base.Scenario()
                with self.subTest(binary=Path(binary).name, strategy=strategy), tempfile.TemporaryDirectory() as directory, SectionFixture(base.SCENARIO, dispatch=scenario.dispatch) as server:
                    workspace = Path(directory)
                    for name in ("home", "tmp", "feishu-config"):
                        (workspace / name).mkdir()
                    env = {**cli_environment(workspace), **server.environment()}
                    flags = ["--base-token", base.BASE, "--table-id", base.TABLE, "--as", "user", "--format", "json"]
                    metadata = subprocess.run([binary, "base", "+table-get", *flags], cwd=workspace, env=env, capture_output=True, text=True, timeout=20)
                    self.assertEqual(metadata.returncode, 0, metadata.stderr)
                    args = (["+record-upsert", "--record-id", base.TARGET, "--json", '{"fld_amount":0,"fld_text":null}'] if strategy == "single" else
                            ["+record-batch-update", "--json", json.dumps({"update_records": {base.TARGET: {"Amount": 0, "Text": None}}})])
                    command = ([binary, "api", "PATCH", base.PREFIX + "/records/" + base.TARGET,
                                "--data", '{"Amount":0,"Text":null}', "--as", "user", "--format", "json"]
                               if strategy == "raw-api" else [binary, "base", *args, *flags])
                    updated = subprocess.run(command, cwd=workspace, env=env, capture_output=True, text=True, timeout=20)
                    self.assertIn(updated.returncode, (0, 22) if strategy == "batch" else (0,), updated.stderr)
                    read = subprocess.run([binary, "base", "+record-get", "--record-id", base.TARGET, *flags], cwd=workspace, env=env, capture_output=True, text=True, timeout=20)
                    self.assertEqual(read.returncode, 0, read.stderr)
                    receipt = server.receipt()
                    receipt["state"] = scenario.snapshot()
                    self.assertTrue(base.verify(workspace, base.TASK["expected"]["answer"], receipt, base.TASK["expected"]))


if __name__ == "__main__":
    unittest.main()
