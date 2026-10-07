# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Controlled scaling fixture tests; optional clean CLI runs never invoke a model."""
from copy import deepcopy
import csv
from decimal import Decimal
import hashlib
import io
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "evaluation"))
from scaling import sheets
from codex_runner import cli_environment
from scenario_fixture import ScenarioFixture

READ_PATH = "/open-apis/sheet_ai/v2/spreadsheets/shtScaleDev01/tools/invoke_read"
IDENTITY_PATH = "/open-apis/authen/v1/user_info"
SIZES = (100, 1000, 10000)


def csv_records(annotated):
    # The frozen prefix-removal implementation strips a comma after the marker.
    records = list(csv.reader(io.StringIO(annotated)))
    if any(not record or re.fullmatch(r"\[row=[1-9][0-9]*\]", record[0]) is None for record in records):
        raise ValueError("invalid native CSV row prefix")
    return [record[1:] for record in records]


def payload_matrix(payload):
    """Independently extract complete native output in row/column order."""
    if payload.get("schema") == "feishu-cli-next/result/v1":
        return payload["result"]["data"]["cells"]
    if "ok" in payload:
        payload = payload["data"]
    rows = {}
    for region in payload["ranges"]:
        for index, row in zip(region["row_indices"], region["cells"]):
            for column, cell in zip(region["col_indices"], row):
                rows[index, column] = cell
    count = max(index for index, _ in rows)
    return [[rows[index, column] for column in "ABCDEFGH"] for index in range(1, count + 1)]


def independent_hash(matrix):
    """Contract encoding, independently implemented and not delegated to the verifier."""
    digest = hashlib.sha256()
    for row_index, row in enumerate(matrix, 1):
        for column, cell in zip("ABCDEFGH", row):
            if column == "H":
                token = ["formula", cell["formula"]]
            elif "value" not in cell:
                token = ["omitted"]
            else:
                value = cell["value"]
                if value is None:
                    token = ["null"]
                elif type(value) is bool:
                    token = ["boolean", value]
                elif isinstance(value, (int, float, Decimal)):
                    decimal = Decimal(str(value))
                    number = format(decimal, "f")
                    if "." in number:
                        number = number.rstrip("0").rstrip(".")
                    token = ["number", "0" if decimal == 0 else number]
                else:
                    token = ["string", value]
            digest.update(json.dumps([row_index, ord(column) - ord("A"), token], ensure_ascii=False,
                                     separators=(",", ":")).encode("utf-8") + b"\n")
    return digest.hexdigest()


class ScalingSheetsTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.workspace = Path(self.directory.name)
        self.reset(100)

    def reset(self, count):
        self.count = count
        self.task = sheets.task(count)
        self.scenario = sheets.Scenario(count)
        self.records = []

    def call(self, tool="get_cell_ranges", **kwargs):
        data = {"excel_id": "shtScaleDev01"}
        if tool == "get_cell_ranges":
            data.update(sheet_id="shscale01", ranges=["A1:H" + str(self.count)])
        elif tool in ("get_sheet_structure", "get_range_as_csv"):
            data["sheet_id"] = "shscale01"
        data.update(kwargs)
        if "sheet_name" in kwargs and "sheet_id" not in kwargs:
            data.pop("sheet_id", None)
        body = json.dumps({"tool_name": tool, "input": json.dumps(data)}).encode()
        reply = self.scenario.dispatch("POST", READ_PATH, body, {"X-Lark-Proxy-Identity": "user"})
        self.records.append({"route": reply.route, "status": reply.status, "proof": reply.proof,
                             "method": "POST", "uri": READ_PATH, "signature_valid": True, "returned_ids": []})
        return json.loads(reply.json["data"]["output"])

    def identity(self):
        reply = self.scenario.dispatch("GET", IDENTITY_PATH, b"", {"X-Lark-Proxy-Identity": "user"})
        self.records.append({"route": reply.route, "status": reply.status, "proof": reply.proof,
                             "method": "GET", "uri": IDENTITY_PATH, "signature_valid": True, "returned_ids": []})

    def receipt(self):
        return {"scenario": self.task["fixture"]["scenario"], "requests": deepcopy(self.records),
                "state": self.scenario.snapshot()}

    def check(self, payload=None, receipt=None, answer=None):
        if payload is not None:
            (self.workspace / "sheet.json").write_text(json.dumps(payload, ensure_ascii=False), encoding="utf-8")
        return sheets.verify(self.workspace, answer or {"artifact_path": "sheet.json"},
                             receipt if receipt is not None else self.receipt(), self.task["expected"])

    def bound(self, matrix):
        self.identity()
        return {"schema": "feishu-cli-next/result/v1", "binding": {
                    "environment": "feishu", "endpoint": "https://open.feishu.cn",
                    "app_id": "synthetic-evaluation-app", "identity": "user",
                    "user_id": "ou_scale_fixture_self", "tenant_key": "scale-fixture-tenant",
                    "user_verified": True, "tenant_verified": True},
                "result": {"ok": True, "data": {"schema": "feishu-cli-next/sheets-data/v1",
                    "spreadsheet_token": "shtScaleDev01", "sheet_id": "shscale01", "range": "A1:H" + str(self.count),
                    "count": self.count, "complete": True, "cells": matrix}}}

    def test_public_tasks_expose_only_extent_and_type_schema(self):
        for count in SIZES:
            task = sheets.task(count)
            fixture = task["fixture"]
            self.assertEqual(set(fixture), {"scenario", "row_count", "answer_schema"})
            self.assertEqual(fixture["scenario"], "scaling-sheets-v2")
            self.assertEqual(fixture["row_count"], count)
            self.assertEqual(fixture["answer_schema"], {"type": "object", "properties": {"artifact_path": {"type": "string"}},
                                                      "required": ["artifact_path"], "additionalProperties": False})
            self.assertIn("A1:H" + str(count), task["prompt"])
            for hidden in ("cells", "hash", "seed", "answer", "expected"):
                self.assertNotIn(hidden, fixture)

    def test_fixed_family_is_a_prefix_and_every_size_preserves_types(self):
        previous = None
        for count in SIZES:
            self.reset(count)
            payload = self.call(cell_limit=1000000000, max_chars=20000000)
            self.assertIs(payload["has_more"], False)
            self.assertEqual(payload["returned_cell_count"], count * 8)
            matrix = payload_matrix(payload)
            self.assertEqual(len(matrix), count)
            if previous:
                self.assertEqual(matrix[:len(previous)], previous)
            previous = matrix
            for index in range(count):
                row = matrix[index]
                self.assertEqual(len(row), 8)
                self.assertIs(type(row[0]["value"]), str)
                self.assertEqual(row[0]["value"], f"scale-native-v1:{index:06d}:你好\n✓")
                self.assertIn("你好", row[0]["value"])
                self.assertIn("\n", row[0]["value"])
                self.assertEqual(row[1]["value"], (0, -3.5, 0.125, 9007199254740991)[index % 4])
                self.assertIsNot(type(row[1]["value"]), bool)
                self.assertIs(row[2]["value"], index % 2 == 1)
                self.assertEqual(row[3]["value"], "")
                self.assertIs(row[4]["value"], None)
                self.assertNotIn("value", row[5])
                self.assertEqual(row[6]["value"], f"{index:08d}" if index % 2 == 0 else f"=literal-{index:06d}")
                self.assertNotIn("formula", row[6])
                self.assertEqual(row[7]["formula"], f"=B{index + 1}+1")
                self.assertEqual(Decimal(str(row[7]["value"])), Decimal(str(row[1]["value"])) + 1)
            self.assertEqual(len({row[0]["value"] for row in matrix}), count)
            self.assertEqual(matrix, sheets.source_rows(count))
            self.assertEqual(independent_hash(matrix), sheets.semantic_hash(matrix))
            self.assertTrue(self.check(payload))

    def test_formula_and_raw_artifact_hashes_agree_across_containers(self):
        raw = self.call()
        formula = self.call(value_render_option="formula", include_styles=True, include_truncation_info=True)
        raw_matrix, formula_matrix = payload_matrix(raw), payload_matrix(formula)
        self.assertNotEqual(json.dumps(raw), json.dumps(formula))
        self.assertEqual(independent_hash(raw_matrix), independent_hash(formula_matrix))
        self.assertEqual(formula_matrix[1][6]["value"], "=literal-000001")
        self.assertEqual(formula_matrix[-1][7]["value"], "=B100+1")
        self.assertTrue(self.check(formula))
        self.assertTrue(self.check({"ok": True, "data": formula}))
        self.assertTrue(self.check(self.bound(raw_matrix)))

    def test_typed_presence_content_and_formula_corruption_fail(self):
        original = self.call()
        changes = [(0, 1, {"value": False}), (0, 2, {"value": 0}), (0, 3, {"value": None}),
                   (0, 4, {}), (0, 5, {"value": None}), (0, 6, {"value": 0}),
                   (1, 6, {"formula": "=literal-000001"}), (0, 0, {"value": "forged"}),
                   (99, 7, {"value": 1}), (99, 7, {"formula": "=B99+1"}),
                   (0, 7, {"formula": "=B1+1", "value": True}), (1, 7, {"formula": "=B2+1", "value": 1}),
                   (0, 0, {"value": original["ranges"][0]["cells"][0][0]["value"], "rich_text": [{"text": "foreign"}]})]
        for row, column, cell in changes:
            broken = deepcopy(original)
            broken["ranges"][0]["cells"][row][column] = cell
            with self.subTest(row=row, column=column, cell=cell):
                self.assertFalse(self.check(broken))
        original["ranges"][0]["cells"][0][7].pop("value")
        self.assertTrue(self.check(original), "formula cache omission is explicitly allowed")

    def test_structured_content_metadata_has_a_validated_empty_shape(self):
        full = self.call()
        for key in ("rich_text", "multiple_values"):
            for value in ("", {}, 0, False):
                broken = deepcopy(full)
                broken["ranges"][0]["cells"][0][0][key] = value
                with self.subTest(key=key, value=value):
                    self.assertFalse(self.check(broken))
            for value in (None, []):
                valid = deepcopy(full)
                valid["ranges"][0]["cells"][0][0][key] = value
                self.assertTrue(self.check(valid))

    def test_ranges_gaps_duplicates_reordering_and_truncation(self):
        first = self.call(ranges=["A1:D100"])
        second = self.call(ranges=["E1:H100"])
        composed = {"ranges": first["ranges"] + second["ranges"]}
        self.assertTrue(self.check(composed))
        self.assertFalse(self.check(first))
        full = self.call()
        for edit in (lambda p: p.update(has_more=True),
                     lambda p: p["ranges"][0].update(truncated=True),
                     lambda p: p["ranges"][0]["cells"][99][7].update(isColTruncated=True),
                     lambda p: p["ranges"][0]["cells"].pop(),
                     lambda p: p["ranges"][0]["cells"].append(deepcopy(p["ranges"][0]["cells"][-1])),
                     lambda p: p["ranges"][0]["cells"].reverse(),
                     lambda p: p["ranges"][0].update(row_indices=list(range(2, 102))),
                     lambda p: p["ranges"][0].update(col_indices=list("BACDEFGH"))):
            broken = deepcopy(full)
            edit(broken)
            self.assertFalse(self.check(broken))
        duplicate = deepcopy(full)
        duplicate["ranges"].append(deepcopy(full["ranges"][0]))
        self.assertTrue(self.check(duplicate))
        duplicate["ranges"][-1]["cells"][99][1]["value"] = 0
        self.assertFalse(self.check(duplicate))

    def test_partial_limits_and_missing_read_proof_cannot_be_forged(self):
        full = self.call()
        self.records.clear()
        self.assertFalse(self.check(full))
        self.call(tool="get_workbook_structure")
        self.identity()
        self.assertFalse(self.check(full))
        partial = self.call(cell_limit=9)
        self.assertTrue(partial["has_more"])
        self.assertEqual(partial["returned_cell_count"], 8, "partial budgets retain complete row coordinates")
        self.assertFalse(self.check(partial))
        self.assertFalse(self.check(full))
        limited = self.call(max_chars=1)
        self.assertTrue(limited["has_more"])
        self.assertLess(limited["returned_cell_count"], 800)
        self.assertFalse(self.check(full))
        self.call(ranges=["A1:H50"])
        self.call(ranges=["A51:H100"])
        self.assertTrue(self.check(full))
        receipt = self.receipt()
        for record in receipt["requests"]:
            if "typed_hash" in record.get("proof", {}):
                record["proof"]["typed_hash"] = "0" * 64
        self.assertFalse(self.check(full, receipt=receipt), "tampered independent observed hash must fail")

    def test_bound_identity_source_extent_and_claims_checked(self):
        full = self.call()
        artifact = self.bound(payload_matrix(full))
        self.assertTrue(self.check(artifact))
        for key in ("environment", "endpoint", "app_id", "identity", "user_id", "tenant_key"):
            broken = deepcopy(artifact)
            broken["binding"][key] = "foreign"
            with self.subTest(key=key):
                self.assertFalse(self.check(broken))
        for key, value in (("range", "A1:H99"), ("spreadsheet_token", "foreign"), ("sheet_id", "foreign"),
                           ("count", 99), ("count", True), ("complete", False)):
            broken = deepcopy(artifact)
            broken["result"]["data"][key] = value
            self.assertFalse(self.check(broken))

    def test_truncation_at_envelope_and_bound_data_levels_is_rejected(self):
        full = self.call()
        bound = self.bound(payload_matrix(full))
        for location in ((), ("result",), ("result", "data")):
            for flag in ("truncated", "has_more", "is_truncated"):
                broken = deepcopy(bound)
                target = broken
                for key in location:
                    target = target[key]
                target[flag] = True
                with self.subTest(location=location, flag=flag):
                    self.assertFalse(self.check(broken))
        self.assertFalse(self.check({"ok": True, "truncated": True, "data": full}))

    def test_missing_bytes_duplicate_keys_nonfinite_and_symlink_escape_fail(self):
        full = self.call()
        self.assertTrue(self.check(full))
        self.assertFalse(self.check(answer={"artifact_path": "missing.json"}))
        self.assertFalse(self.check(answer={"artifact_path": str(self.workspace / "sheet.json")}))
        with tempfile.TemporaryDirectory() as directory:
            outside = Path(directory) / "outside.json"
            outside.write_text(json.dumps(full))
            (self.workspace / "escape.json").symlink_to(outside)
            self.assertFalse(self.check(answer={"artifact_path": "escape.json"}))
        for contents in ('{"ranges":[],"ranges":[]}', '{"ranges":NaN}', '{"ranges":Infinity}', 'not json',
                         '{"ranges":1e9999999999999999999999999}', "[" * 2000 + "0" + "]" * 2000):
            (self.workspace / "sheet.json").write_text(contents)
            self.assertFalse(self.check())
        with (self.workspace / "sheet.json").open("wb") as stream:
            stream.truncate(32 * 1024 * 1024 + 1)
        self.assertFalse(self.check())

    def test_rejected_mutation_attempt_and_source_state_changes_fail(self):
        full = self.call()
        for key, value in (("revision", 2), ("row_count", 101), ("notifications", ["sent"]),
                           ("permissions", {"visibility": "public"})):
            original = deepcopy(self.scenario.current)
            self.scenario.current[key] = value
            self.assertFalse(self.check(full))
            self.scenario.current = original
        self.scenario.current["cells"][99][7]["formula"] = "=B99+1"
        self.assertFalse(self.check(full))
        self.scenario.current["cells"] = sheets.source_rows(self.count)
        self.assertTrue(self.check(full))
        receipt = self.receipt()
        # A rejected write may have failed signature verification before dispatch.
        receipt["requests"].append({"route": "rejected", "status": 403, "method": "POST",
                                    "uri": READ_PATH.replace("invoke_read", "invoke_write"), "signature_valid": False})
        self.assertFalse(self.check(full, receipt=receipt))
        with self.assertRaises(ValueError):
            self.scenario.dispatch("DELETE", "/open-apis/sheets/v3/spreadsheets/shtScaleDev01", b"",
                                   {"X-Lark-Proxy-Identity": "user"})
        self.assertFalse(self.check(full))

    def test_foreign_resources_and_invalid_read_inputs_rejected(self):
        for changes in ({"excel_id": "foreign"}, {"sheet_id": "foreign"}, {"sheet_name": "foreign"},
                        {"ranges": ["A1:I100"]}, {"ranges": ["A1:H101"]}, {"ranges": ["H2:A1"]},
                        {"ranges": []}, {"value_render_option": "display"}, {"cell_limit": True},
                        {"cell_limit": 0}, {"max_chars": -1}, {"skip_hidden": "true"}):
            with self.subTest(changes=changes), self.assertRaises(ValueError):
                self.call(**changes)

    def test_sheet_metadata_is_read_only_and_has_no_invented_merges(self):
        before = self.scenario.snapshot()
        for info_type in ("all", "merged_cells_infos"):
            metadata = self.call(tool="get_sheet_structure", info_type=info_type, range="A1:H2")
            self.assertEqual(metadata["sheet_id"], "shscale01")
            self.assertEqual(metadata["sheet_name"], "Scale")
            self.assertEqual(metadata["revision"], 1)
            self.assertEqual(metadata["row_count"], 100)
            self.assertEqual(metadata["column_count"], 8)
            self.assertEqual(metadata["frozen_row_count"], 0)
            self.assertEqual(metadata["frozen_column_count"], 0)
            if info_type in ("all", "merged_cells_infos"):
                self.assertEqual(metadata["merged_cells"], [])
        named = self.call(tool="get_sheet_structure", sheet_name="Scale", info_type="merged_cells_infos")
        self.assertEqual(named["merged_cells"], [])
        self.assertEqual(before, self.scenario.snapshot())
        for changes in ({"info_type": "unknown"}, {"info_type": "row_heights_column_widths"},
                        {"info_type": "hidden_infos"}, {"info_type": "group_infos"},
                        {"sheet_id": "foreign"}, {"sheet_name": "foreign"},
                        {"range": "A1:H101"}, {"extra": True}):
            with self.subTest(changes=changes), self.assertRaises(ValueError):
                self.call(tool="get_sheet_structure", **changes)

    def test_structure_enum_invalid_and_documented_unsupported_are_distinct(self):
        before = self.scenario.snapshot()
        for info in ("row_heights_column_widths", "hidden_infos", "group_infos"):
            with self.subTest(info=info), self.assertRaisesRegex(ValueError, "^unsupported strategy:"):
                self.call(tool="get_sheet_structure", info_type=info)
        for info in ("unknown", "", None, True, 1, [], {}, ["all"]):
            with self.subTest(info=info), self.assertRaisesRegex(ValueError, "^invalid sheet structure info_type$"):
                self.call(tool="get_sheet_structure", info_type=info)
        self.assertEqual(before, self.scenario.snapshot())

    def test_matching_discovered_id_and_name_can_be_sent_together(self):
        for tool, options in (("get_sheet_structure", {"info_type": "merged_cells_infos"}),
                              ("get_range_as_csv", {"range": "B1:B2"}), ("get_cell_ranges", {})):
            with self.subTest(tool=tool):
                result = self.call(tool=tool, sheet_id="shscale01", sheet_name="Scale", **options)
                self.assertEqual(result["sheet_id"], "shscale01")
                for selectors in ({"sheet_id": "foreign", "sheet_name": "Scale"},
                                  {"sheet_id": "shscale01", "sheet_name": "foreign"}):
                    with self.assertRaises(ValueError):
                        self.call(tool=tool, **selectors, **options)

    def test_csv_numeric_subset_whole_column_clipping_and_multiline_records(self):
        before = self.scenario.snapshot()
        selected = self.call(tool="get_range_as_csv", range="B1:B2")
        self.assertEqual(selected["range"], "B1:B2")
        self.assertEqual(selected["actual_range"], "B1:B2")
        self.assertEqual(selected["current_region"], "A1:C100")
        self.assertEqual(selected["row_indices"], [1, 2])
        self.assertEqual(selected["col_indices"], ["B"])
        self.assertEqual(csv_records(selected["annotated_csv"]), [["0"], ["-3.5"]])
        self.assertIs(selected["has_more"], False)
        full = self.call(tool="get_range_as_csv", sheet_name="Scale", range="A:ZZZ", max_chars=20000000)
        self.assertEqual(full["actual_range"], "A1:H100")
        self.assertEqual(full["current_region"], "A1:H100")
        records = csv_records(full["annotated_csv"])
        self.assertEqual(len(records), 100, "quoted embedded newlines are part of logical CSV records")
        self.assertTrue(all(len(record) == 8 for record in records))
        self.assertEqual(records[0][0], "scale-native-v1:000000:你好\n✓")
        self.assertEqual(records[-1][0], "scale-native-v1:000099:你好\n✓")
        with self.assertRaises(ValueError):
            self.call(tool="get_range_as_csv", range="F1:F100")
        self.assertEqual(before, self.scenario.snapshot())

    def test_csv_limits_and_csv_only_files_cannot_supply_native_proof(self):
        native = self.call()
        self.records.clear()
        for limits in ({"max_rows": 1}, {"max_chars": 1}):
            result = self.call(tool="get_range_as_csv", range="A1:H100", **limits)
            self.assertIs(result["has_more"], True)
            self.assertIs(result["truncated"], True)
            self.assertLessEqual(len(csv_records(result["annotated_csv"])), 1)
        full = self.call(tool="get_range_as_csv", range="A:ZZZ", max_chars=20000000)
        self.assertFalse(self.check(full), "CSV output cannot preserve all native cell identities")
        self.assertFalse(self.check(native), "CSV observations do not count as signed native-cell observations")
        self.call()
        self.assertTrue(self.check(native))
        for changes in ({"sheet_id": "foreign"}, {"sheet_name": "foreign"}, {"range": "bad"},
                        {"max_rows": 0}, {"max_rows": True}, {"max_chars": -1},
                        {"skip_hidden": "true"}, {"value_render_option": "formula"}):
            arguments = {"range": "B1:B2", **changes}
            with self.subTest(changes=changes), self.assertRaises(ValueError):
                self.call(tool="get_range_as_csv", **arguments)

    def test_csv_source_region_is_independent_of_subset_and_emission_limits(self):
        for count in SIZES:
            self.reset(count)
            before = self.scenario.snapshot()
            for limits, actual, emitted in (({}, "B1:B2", 2), ({"max_rows": 1}, "B1:B1", 1),
                                             ({"max_chars": 1}, "", 0)):
                with self.subTest(count=count, scope="subset", limits=limits):
                    result = self.call(tool="get_range_as_csv", range="B1:B2", **limits)
                    self.assertEqual(result["current_region"], f"A1:C{count}")
                    self.assertEqual(result["actual_range"], actual)
                    self.assertEqual(len(csv_records(result["annotated_csv"])), emitted)
            one_row = self.call(tool="get_range_as_csv", range="A1:H1")
            first_row_chars = len(one_row["annotated_csv"])
            for limits, actual, emitted in (({"max_rows": 1}, "A1:H1", 1),
                                             ({"max_chars": first_row_chars}, "A1:H1", 1),
                                             ({"max_chars": 1}, "", 0)):
                with self.subTest(count=count, scope="whole", limits=limits):
                    result = self.call(tool="get_range_as_csv", range=f"A1:H{count}", **limits)
                    self.assertEqual(result["current_region"], f"A1:H{count}")
                    self.assertEqual(result["actual_range"], actual)
                    self.assertEqual(len(csv_records(result["annotated_csv"])), emitted)
                    self.assertEqual(result["row_count"], emitted)
                    self.assertIs(result["has_more"], True)
                    self.assertIs(result["truncated"], True)
            self.assertEqual(before, self.scenario.snapshot())


@unittest.skipUnless(os.environ.get("EVAL_SCALING_SHEETS_BINARIES"), "set EVAL_SCALING_SHEETS_BINARIES to clean archive binary paths")
class ScalingSheetsBinaryTests(unittest.TestCase):
    def test_metadata_csv_and_table_reads_before_final_native_export(self):
        evidence = []
        for binary in json.loads(os.environ["EVAL_SCALING_SHEETS_BINARIES"]):
            for count in SIZES:
                with self.subTest(binary=binary["path"], count=count), tempfile.TemporaryDirectory() as directory:
                    workspace = Path(directory)
                    for name in ("home", "tmp", "feishu-config"):
                        (workspace / name).mkdir()
                    scenario, task = sheets.Scenario(count), sheets.task(count)
                    with ScenarioFixture(task["fixture"]["scenario"], scenario) as server:
                        environment = {**cli_environment(workspace), **server.environment()}

                        def execute(command, *flags):
                            result = subprocess.run([binary["path"], "sheets", command, "--spreadsheet-token", "shtScaleDev01",
                                                     "--as", "user", *flags], cwd=workspace, env=environment,
                                                    capture_output=True, text=True, timeout=90)
                            self.assertEqual(result.returncode, 0, result.stderr)
                            output = json.loads(result.stdout)
                            self.assertIs(output["ok"], True)
                            return output["data"]

                        workbook = execute("+workbook-info")
                        listed = execute("+sheet-list")
                        revision = execute("+revision-get")
                        self.assertEqual(listed, workbook["sheets"])
                        self.assertEqual(revision, {"revision": 1})
                        self.assertEqual(listed[0]["row_count"], count)
                        self.assertEqual(listed[0]["column_count"], 8)
                        metadata = execute("+sheet-info", "--sheet-id", "shscale01", "--include", "merges")
                        self.assertEqual(metadata["merged_cells"], [])
                        selected = execute("+csv-get", "--sheet-id", "shscale01", "--range", "B1:B2")
                        self.assertEqual(selected["actual_range"], "B1:B2")
                        self.assertEqual(selected["current_region"], f"A1:C{count}")
                        self.assertEqual(selected["row_indices"], [1, 2])
                        self.assertEqual(selected["col_indices"], ["B"])
                        self.assertEqual(csv_records(selected["annotated_csv"]), [["0"], ["-3.5"]])
                        plain = execute("+csv-get", "--sheet-id", "shscale01", "--range", "A1:H2", "--include-row-prefix=false")
                        plain_records = list(csv.reader(io.StringIO(plain["annotated_csv"])))
                        self.assertEqual(plain_records, [
                            ["scale-native-v1:000000:你好\n✓", "0", "false", "", "", "", "00000000", "1"],
                            ["scale-native-v1:000001:你好\n✓", "-3.5", "true", "", "", "", "=literal-000001", "-2.5"],
                        ], "prefix removal must preserve the exact quoted multiline first field without a leading space")
                        table_rows = []
                        for explicit in (True, False):
                            filename = "table-explicit.json" if explicit else "table-default.json"
                            flags = ["--sheet-id", "shscale01", "--no-header", "--output-path", filename]
                            if explicit:
                                flags += ["--range", "A1:H" + str(count)]
                            before = len(server.receipt()["requests"])
                            confirmation = execute("+table-get", *flags)
                            self.assertIs(confirmation["complete"], True)
                            table = json.loads((workspace / filename).read_text())
                            self.assertEqual(len(table["sheets"]), 1)
                            data = table["sheets"][0]
                            self.assertEqual(data["range"], "A1:H" + str(count))
                            self.assertEqual(len(data["columns"]), 8)
                            self.assertEqual(len(data["data"]), count)
                            self.assertEqual(data["data"][0][0], "scale-native-v1:000000:你好\n✓")
                            self.assertEqual(data["data"][-1][0], f"scale-native-v1:{count - 1:06d}:你好\n✓")
                            table_rows.append(len(data["data"]))
                            requests = server.receipt()["requests"][before:]
                            operations = [record.get("proof", {}).get("operation") for record in requests]
                            self.assertIn("get_cell_ranges", operations)
                            self.assertEqual("get_range_as_csv" in operations, not explicit)
                            self.assertFalse(sheets.verify(workspace, {"artifact_path": filename}, server.receipt(), task["expected"]),
                                             "typed table output is not a native-cell artifact")
                        final_flags = (["--save-result", "sheet.json"] if binary.get("next") else
                                       ["--include", "value,formula,truncation", "--output-path", "sheet.json"])
                        final_command = "+data-export" if binary.get("next") else "+cells-get"
                        execute(final_command, "--sheet-id", "shscale01", "--range", "A1:H" + str(count), *final_flags)
                        receipt = server.receipt()
                        self.assertTrue(sheets.verify(workspace, {"artifact_path": "sheet.json"}, receipt, task["expected"]))
                        self.assertTrue(all(record["status"] == 200 and record["signature_valid"] for record in receipt["requests"]))
                        evidence.append({"arm": "next" if binary.get("next") else "official", "rows": count,
                                         "table_rows": table_rows, "final_command": final_command,
                                         "api_reads": len(receipt["requests"])})
        print(json.dumps({"sheets_scaling_metadata_controlled": evidence}, sort_keys=True))

    def test_all_sizes_from_clean_official_and_next_cli(self):
        family_hashes = {}
        evidence = []
        for binary in json.loads(os.environ["EVAL_SCALING_SHEETS_BINARIES"]):
            strategies = [["+cells-get", "--include", "value,formula,truncation", "--output-path", "sheet.json"]]
            if binary.get("next"):
                strategies.append(["+data-export", "--save-result", "sheet.json"])
            for count in SIZES:
                for strategy in strategies:
                    with self.subTest(binary=binary["path"], count=count, strategy=strategy), tempfile.TemporaryDirectory() as directory:
                        workspace = Path(directory)
                        for name in ("home", "tmp", "feishu-config"):
                            (workspace / name).mkdir()
                        scenario = sheets.Scenario(count)
                        task = sheets.task(count)
                        with ScenarioFixture(task["fixture"]["scenario"], scenario) as server:
                            result = subprocess.run([binary["path"], "sheets", *strategy, "--spreadsheet-token", "shtScaleDev01",
                                                     "--sheet-id", "shscale01", "--range", "A1:H" + str(count), "--as", "user"],
                                                    cwd=workspace, env={**cli_environment(workspace), **server.environment()},
                                                    capture_output=True, text=True, timeout=90)
                            self.assertEqual(result.returncode, 0, result.stderr)
                            artifact = workspace / "sheet.json"
                            self.assertLess(artifact.stat().st_size, 32 * 1024 * 1024)
                            receipt = server.receipt()
                            self.assertTrue(sheets.verify(workspace, {"artifact_path": "sheet.json"}, receipt, task["expected"]))
                            pages = [record for record in receipt["requests"] if record.get("proof", {}).get("operation") == "get_cell_ranges"]
                            self.assertEqual(sum(record["proof"]["returned_cell_count"] for record in pages), count * 8)
                            evidence.append({"arm": "next" if binary.get("next") else "official", "rows": count,
                                             "command": strategy[0], "artifact_bytes": artifact.stat().st_size,
                                             "read_pages": len(pages)})
                            matrix = payload_matrix(json.loads(artifact.read_text(), parse_float=Decimal))
                            self.assertEqual(len(matrix), count)
                            observed = independent_hash(matrix)
                            if count in family_hashes:
                                self.assertEqual(observed, family_hashes[count])
                            family_hashes[count] = observed
                            stdout = json.loads(result.stdout)
                            self.assertNotIn("你好", result.stdout)
                            self.assertIsInstance(stdout, dict)
        print(json.dumps({"sheets_scaling_controlled": evidence}, sort_keys=True))


if __name__ == "__main__":
    unittest.main()
