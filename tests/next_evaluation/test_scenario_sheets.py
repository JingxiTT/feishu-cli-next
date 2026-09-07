"""Artifact correctness tests run without network, credentials, or optional binaries."""
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
from scenarios import sheets
from codex_runner import cli_environment


class SheetsScenarioTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.workspace = Path(self.directory.name)
        self.scenario = sheets.Scenario()
        self.records = []

    def call(self, tool="get_cell_ranges", **inputs):
        data = {"excel_id": "shtSheetsDev01"}
        if tool == "get_cell_ranges":
            data.update(sheet_id="shdev01", ranges=["A1:E1"])
        data.update(inputs)
        if "sheet_name" in inputs:
            data.pop("sheet_id")
        raw = json.dumps({"tool_name": tool, "input": json.dumps(data)}).encode()
        reply = self.scenario.dispatch("POST", sheets.READ_PATH, raw, {"X-Lark-Proxy-Identity": "user"})
        self.records.append({"route": reply.route, "status": reply.status, "proof": reply.proof, "returned_ids": []})
        return json.loads(reply.json["data"]["output"])

    def identity(self):
        reply = self.scenario.dispatch("GET", "/open-apis/authen/v1/user_info", b"", {"X-Lark-Proxy-Identity": "user"})
        self.records.append({"route": reply.route, "status": reply.status, "proof": reply.proof})

    def check(self, payload=None, **answer):
        if payload is not None:
            (self.workspace / "sheet.json").write_text(json.dumps(payload, ensure_ascii=False), encoding="utf-8")
        return sheets.verify(self.workspace, answer or {"artifact_path": "sheet.json"},
                             {"scenario": sheets.SCENARIO, "requests": self.records,
                              "state": self.scenario.snapshot()}, sheets.TASK["expected"])

    def bound(self):
        payload = self.call()
        self.identity()
        return {"schema": "feishu-cli-next/result/v1", "binding": dict(sheets.BINDING, user_verified=True, tenant_verified=True),
                "result": {"ok": True, "data": {"schema": "feishu-cli-next/sheets-data/v1", "spreadsheet_token": "shtSheetsDev01",
                "sheet_id": "shdev01", "range": "A1:E1", "complete": True, "count": 1, "cells": payload["ranges"][0]["cells"]}}}

    def test_public_fixture_contains_types_only(self):
        fixture = sheets.TASK["fixture"]
        self.assertEqual(set(fixture), {"scenario", "answer_schema"})
        self.assertEqual(fixture["answer_schema"], {"type": "object", "required": ["artifact_path"],
                         "properties": {"artifact_path": {"type": "string"}}, "additionalProperties": False})
        self.assertNotIn("你好", json.dumps(fixture, ensure_ascii=False))
        self.assertNotIn("expected", fixture)

    def test_raw_and_formula_selectors_preserve_native_types(self):
        for selector in ({"sheet_id": "shdev01"}, {"sheet_name": "Fixture"}):
            for render in ("raw_value", "formula"):
                with self.subTest(selector=selector, render=render):
                    payload = self.call(**selector, value_render_option=render, include_styles=True,
                                        include_truncation_info=True, skip_hidden=True)
                    row = payload["ranges"][0]["cells"][0]
                    self.assertIs(type(row[0]["value"]), int)
                    self.assertIs(row[1]["value"], False)
                    self.assertEqual(row[2]["value"], "")
                    self.assertEqual(row[3]["value"], "你好")
                    self.assertEqual(row[4]["formula"], "=A1+1")
                    self.assertEqual(row[4]["value"], "=A1+1" if render == "formula" else 1)
                    self.assertEqual(row[1]["cell_styles"], {})
                    self.assertEqual(row[0]["cell_styles"], {"number_format": "General"})
                    self.assertIs(row[3]["isColTruncated"], False)
                    self.assertTrue(self.check(payload))
                    self.assertTrue(self.check({"ok": True, "data": payload}))
        self.assertEqual(self.scenario.snapshot()["mutation_attempts"], 0)

    def test_independent_partial_reads_can_collectively_cover_target(self):
        first = self.call(ranges=["A1:B1"])
        second = self.call(ranges=["C1:D1", "E1"])
        self.assertEqual(second["returned_cell_count"], 3)
        self.assertEqual(second["ranges"][1]["col_indices"], ["E"])
        payload = {"ranges": first["ranges"] + second["ranges"]}
        self.assertTrue(self.check(payload))
        payload["ranges"].append(deepcopy(first["ranges"][0]))
        self.assertTrue(self.check(payload), "identical overlapping coordinates are valid")
        payload["ranges"][-1]["cells"][0][0]["value"] = False
        self.assertFalse(self.check(payload), "conflicting duplicate zero/false must fail")

    def test_limits_return_only_actual_observed_cells(self):
        for limits in ({"cell_limit": 2}, {"max_chars": 14}, {"cell_limit": 1, "max_chars": 1}):
            with self.subTest(limits=limits):
                payload = self.call(**limits)
                region = payload["ranges"][0]
                self.assertTrue(payload["has_more"])
                self.assertTrue(region["truncated"])
                self.assertLess(payload["returned_cell_count"], 5)
                self.assertEqual(payload["returned_cell_count"], len(self.records[-1]["proof"]["coordinates"]))
                self.assertFalse(self.check(payload))
        full = self.call(cell_limit=5, max_chars=4096)
        self.assertFalse(full["has_more"])
        self.assertTrue(self.check(full))

    def test_rejects_wrong_resource_selectors_formats_types_and_queries(self):
        invalid = [{"excel_id": "foreign"}, {"sheet_id": "foreign"}, {"sheet_name": "Foreign"},
                   {"ranges": ["A2:E2"]}, {"ranges": ["A1:F1"]}, {"ranges": ["E1:A1"]},
                   {"ranges": []}, {"value_render_option": "display"}, {"cell_limit": True},
                   {"cell_limit": 0}, {"max_chars": -1}, {"include_styles": "true"}, {"unknown": 1}]
        for data in invalid:
            with self.subTest(data=data), self.assertRaises(ValueError):
                self.call(**data)
        for method, uri, body, headers in [
            ("GET", "/open-apis/authen/v1/user_info?extra=x", b"", {"X-Lark-Proxy-Identity": "user"}),
            ("GET", "/open-apis/authen/v1/user_info", b"", {"X-Lark-Proxy-Identity": "bot"}),
            ("POST", sheets.READ_PATH + "?extra=x", b"{}", {"X-Lark-Proxy-Identity": "user"}),
            ("POST", sheets.READ_PATH, b'{"tool_name":"get_cell_ranges","input":{}}', {"X-Lark-Proxy-Identity": "user"}),
        ]:
            with self.subTest(uri=uri), self.assertRaises(ValueError):
                self.scenario.dispatch(method, uri, body, headers)

    def test_metadata_or_claims_do_not_substitute_for_cell_read_proof(self):
        payload = self.call()
        self.records.clear()
        self.assertFalse(self.check(payload))
        self.call(tool="get_workbook_structure")
        self.identity()
        self.assertFalse(self.check(payload))
        self.call(ranges=["A1:D1"])
        self.assertFalse(self.check(payload))
        self.call(ranges=["E1"])
        self.assertTrue(self.check(payload))

    def test_typed_values_formula_and_empty_presence_are_checked(self):
        original = self.call()
        cases = [(0, {"value": False}), (0, {"value": "0"}), (1, {"value": 0}),
                 (1, {"value": "false"}), (2, {}), (2, {"value": None}), (2, {"value": False}),
                 (3, {"value": "您好"}), (4, {"value": "=A1+1"}), (4, {"formula": "=A1+2"}),
                 (4, {"formula": "=A1+1", "value": True})]
        for index, cell in cases:
            payload = deepcopy(original)
            payload["ranges"][0]["cells"][0][index] = cell
            with self.subTest(index=index, cell=cell):
                self.assertFalse(self.check(payload))
        original["ranges"][0]["cells"][0][4] = {"formula": "=A1+1"}
        self.assertTrue(self.check(original))
        original["ranges"][0]["cells"][0][0]["value"] = 0.0
        original["ranges"][0]["cells"][0][4]["value"] = 1.0
        self.assertTrue(self.check(original), "equivalent JSON numbers remain numbers")

    def test_truncation_and_extra_native_content_are_not_metadata(self):
        original = self.call()
        for key, value in (("isRowTruncated", True), ("isColTruncated", True),
                           ("rich_text", [{"text": "foreign"}]), ("multiple_values", ["foreign"])):
            broken = deepcopy(original)
            broken["ranges"][0]["cells"][0][0][key] = value
            self.assertFalse(self.check(broken))
        for key in ("has_more", "is_truncated"):
            broken = deepcopy(original)
            broken["ranges"][0][key] = True
            self.assertFalse(self.check(broken))

    def test_actual_file_bytes_extent_duplicates_and_paths_checked(self):
        payload = self.call()
        self.assertTrue(self.check(payload))
        for edit in (lambda p: p.update(has_more=True),
                     lambda p: p["ranges"][0].update(truncated=True),
                     lambda p: p["ranges"][0].update(row_indices=[True]),
                     lambda p: p["ranges"][0].update(col_indices=list("EDCBA")),
                     lambda p: p["ranges"][0]["cells"][0].append({"value": "foreign"}),
                     lambda p: p.update(spreadsheet_token="foreign")):
            broken = deepcopy(payload)
            edit(broken)
            self.assertFalse(self.check(broken))
        self.assertFalse(self.check(payload, artifact_path="missing.json"))
        self.assertFalse(self.check(payload, artifact_path=str(self.workspace / "sheet.json")))
        with tempfile.TemporaryDirectory() as external:
            path = Path(external) / "outside.json"
            path.write_text(json.dumps(payload))
            (self.workspace / "escape.json").symlink_to(path)
            self.assertFalse(self.check(artifact_path="escape.json"))
        for contents in ('{"ranges":[],"ranges":[]}', '{"ranges":NaN}', 'not json'):
            (self.workspace / "sheet.json").write_text(contents)
            self.assertFalse(self.check())

    def test_bound_export_checks_source_account_and_observed_claims(self):
        artifact = self.bound()
        self.assertTrue(self.check(artifact))
        for key in ("app_id", "user_id", "tenant_key", "environment", "endpoint", "identity"):
            broken = deepcopy(artifact)
            broken["binding"][key] = "foreign"
            with self.subTest(key=key):
                self.assertFalse(self.check(broken))
        for key, value in (("count", True), ("count", 5), ("complete", False), ("sheet_id", "foreign"), ("range", "A1:D1")):
            broken = deepcopy(artifact)
            broken["result"]["data"][key] = value
            self.assertFalse(self.check(broken))
        self.records = [r for r in self.records if r["route"] != "identity"]
        self.assertFalse(self.check(artifact), "claiming verified identity requires observed identity read")
        artifact["binding"].update(user_verified=False, tenant_verified=False)
        self.assertTrue(self.check(artifact), "unverified known binding remains allowed for read artifact")

    def test_rejected_writes_and_hidden_state_changes_fail(self):
        payload = self.call()
        for key, value in (("revision", 18), ("notifications", ["sent"]),
                           ("permissions", {"visibility": "public"}), ("cells", [[{"value": False}]])):
            before = deepcopy(self.scenario.current)
            self.scenario.current[key] = value
            self.assertFalse(self.check(payload))
            self.scenario.current = before
        with self.assertRaises(ValueError):
            self.scenario.dispatch("POST", sheets.READ_PATH.replace("invoke_read", "invoke_write"), b"{}",
                                   {"X-Lark-Proxy-Identity": "user"})
        self.assertFalse(self.check(payload))
        self.assertEqual(self.scenario.snapshot()["mutation_attempts"], 1)

    def test_transport_rejected_mutations_cannot_hide_before_dispatch(self):
        payload = self.call()
        self.records.append({"route": "rejected", "status": 403, "method": "POST",
                             "uri": sheets.READ_PATH.replace("invoke_read", "invoke_write")})
        self.assertEqual(self.scenario.snapshot()["mutation_attempts"], 0)
        self.assertFalse(self.check(payload))
        self.records[-1]["method"] = "GET"
        self.assertTrue(self.check(payload), "a failed read does not imply a mutation")


@unittest.skipUnless(os.environ.get("EVAL_SHEETS_BINARIES"), "set EVAL_SHEETS_BINARIES to clean archive binary paths")
class SheetsBinaryTests(unittest.TestCase):
    def test_native_artifacts_from_clean_binaries(self):
        from scenario_fixture import ScenarioFixture
        # JSON array of {path, next}; no dirty-source builds and no model calls.
        for binary in json.loads(os.environ["EVAL_SHEETS_BINARIES"]):
            strategies = [["+cells-get", "--include", "value,formula,style,truncation", "--output-path", "sheet.json"]]
            if binary.get("next"):
                strategies.append(["+data-export", "--save-result", "sheet.json"])
            for strategy in strategies:
                with self.subTest(binary=binary["path"], strategy=strategy), tempfile.TemporaryDirectory() as directory:
                    workspace = Path(directory)
                    for name in ("home", "tmp", "feishu-config"):
                        (workspace / name).mkdir()
                    scenario = sheets.Scenario()
                    scenario.setup(workspace)
                    with ScenarioFixture(sheets.SCENARIO, scenario) as server:
                        result = subprocess.run([binary["path"], "sheets", *strategy, "--spreadsheet-token", "shtSheetsDev01",
                                                 "--sheet-id", "shdev01", "--range", "A1:E1", "--as", "user"],
                                                cwd=workspace, env={**cli_environment(workspace), **server.environment()},
                                                capture_output=True, text=True, timeout=30)
                        self.assertEqual(result.returncode, 0, result.stderr)
                        self.assertTrue(sheets.verify(workspace, {"artifact_path": "sheet.json"}, server.receipt(), sheets.TASK["expected"]))


if __name__ == "__main__":
    unittest.main()
