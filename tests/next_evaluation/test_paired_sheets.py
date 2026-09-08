"""Fresh paired native render contract; no models or live network calls."""
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
from paired import sheets
from codex_runner import cli_environment


def merge(raw, formula):
    """Controlled local composition uses actual native responses, never expected data."""
    if type(raw.get("revision")) is not int or raw["revision"] != formula.get("revision"):
        raise ValueError("revision mismatch")
    result = deepcopy(raw)
    formulas = sheets.original._native_cells(formula) if len(formula["ranges"][0]["cells"][0]) == 5 else {
        col + "1": cell for region in formula["ranges"] for col, cell in zip(region["col_indices"], region["cells"][0])}
    for region in result["ranges"]:
        for col, cell in zip(region["col_indices"], region["cells"][0]):
            other = formulas.get(col + "1", {})
            if "formula" in other:
                if "value" in other and not sheets.original._cell_value_same(cell.get("value"), other["value"]):
                    raise ValueError("cache mismatch")
                cell["formula"] = other["formula"]
    return result


class PairedSheetsTests(unittest.TestCase):
    def setUp(self):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        self.workspace = Path(directory.name)
        self.scenario = sheets.Scenario()
        self.records = []

    def call(self, tool="get_cell_ranges", **inputs):
        data = {"excel_id": "shtSheetsDev01"}
        if tool == "get_cell_ranges":
            data.update(sheet_id="shdev01", ranges=["A1:E1"], value_render_option="raw_value")
        data.update(inputs)
        if data.get("value_render_option") == "OMIT":
            del data["value_render_option"]
        raw = json.dumps({"tool_name": tool, "input": json.dumps(data)}).encode()
        reply = self.scenario.dispatch("POST", sheets.READ_PATH, raw, {"X-Lark-Proxy-Identity": "user"})
        self.records.append({"route": reply.route, "status": reply.status, "proof": reply.proof,
                             "signature_valid": True, "method": "POST", "uri": sheets.READ_PATH})
        return json.loads(reply.json["data"]["output"])

    def pair(self, **formula_options):
        return merge(self.call(), self.call(value_render_option="formula", **formula_options))

    def check(self, payload=None, answer=None):
        if payload is not None:
            (self.workspace / "sheet.json").write_text(json.dumps(payload, ensure_ascii=False))
        return sheets.verify(self.workspace, answer or {"artifact_path": "sheet.json"},
                             {"scenario": sheets.SCENARIO, "requests": self.records,
                              "state": self.scenario.snapshot()}, sheets.TASK["expected"])

    def test_public_routing_contains_no_source_answers(self):
        self.assertEqual(sheets.TASK["id"], "paired-sheets")
        self.assertEqual(sheets.TASK["section"], "sheets")
        self.assertEqual(set(sheets.TASK["fixture"]), {"scenario", "scenario_section", "answer_schema"})
        self.assertEqual(sheets.TASK["fixture"]["scenario_section"], "paired-sheets")
        self.assertNotIn("你好", json.dumps(sheets.TASK["fixture"], ensure_ascii=False))

    def test_actual_wire_rendering_is_separate_and_revision_bound(self):
        raw, formula = self.call(), self.call(value_render_option="formula")
        self.assertEqual(raw["revision"], 17)
        self.assertEqual(formula["revision"], 17)
        for payload, mode in ((raw, "raw_value"), (formula, "formula")):
            self.assertEqual(payload["ranges"][0]["cells"][0], list(sheets.TASK["expected"]["rendered"][mode].values()))
        self.assertFalse(self.check(raw))
        self.assertFalse(self.check(formula))
        self.assertTrue(self.check(merge(raw, formula)))
        self.assertTrue(self.check({"ok": True, "data": merge(raw, formula)}))

    def test_partial_formula_read_and_matching_dual_selector_are_valid(self):
        self.assertTrue(self.check(self.pair(ranges=["E1"], sheet_name="Fixture")))
        with self.assertRaises(ValueError):
            self.call(sheet_name="Foreign")

    def test_supported_inspection_is_distinguished_from_invalid_requests(self):
        for mode in ("OMIT", "formatted_value"):
            with self.subTest(mode=mode), self.assertRaisesRegex(ValueError, "^unsupported strategy:"):
                self.call(value_render_option=mode)
        for tool in ("get_sheet_structure", "get_range_as_csv"):
            with self.subTest(tool=tool), self.assertRaisesRegex(ValueError, "^unsupported strategy:"):
                self.call(tool=tool, sheet_id="shdev01", range="A1:E1")
        for data in ({"value_render_option": None}, {"value_render_option": "bogus"},
                     {"sheet_id": "foreign"}, {"cell_limit": True}, {"include_styles": "true"},
                     {"tool": "invented"}, {"tool": "get_sheet_structure", "sheet_id": "shdev01", "info_type": []},
                     {"tool": "get_range_as_csv", "sheet_id": "foreign", "range": "A1:E1"}):
            with self.subTest(data=data), self.assertRaises(ValueError) as caught:
                self.call(**data)
            self.assertFalse(str(caught.exception).startswith("unsupported strategy:"))

    def test_read_proof_cannot_be_substituted_or_forged(self):
        payload = self.pair()
        valid = deepcopy(self.records)
        for index, changes in ((0, {"signature_valid": False}), (0, {"method": "GET"}),
                               (0, {"uri": "/foreign"})):
            self.records = deepcopy(valid)
            self.records[index].update(changes)
            self.assertFalse(self.check(payload))
        for key, value in (("revision", None), ("revision", True), ("revision", "17"), ("revision", 18),
                           ("identity", "bot"), ("operation", "other"), ("sheet_id", "foreign"),
                           ("spreadsheet_token", "foreign"), ("formula_coordinates", ["E1"]),
                           ("content_hashes", {"A1": "invented"})):
            self.records = deepcopy(valid)
            self.records[0]["proof"][key] = value
            with self.subTest(key=key, value=value):
                self.assertFalse(self.check(payload))
        self.records = valid[:1]
        self.assertFalse(self.check(payload), "raw-only proof cannot establish formula identity")
        self.records = valid[1:]
        self.assertFalse(self.check(payload), "display proof cannot establish raw scalars")
        self.records = []
        self.assertFalse(self.check(payload))

    def test_artifact_types_formula_cache_and_presence_are_independent(self):
        payload = self.pair()
        cases = [(0, {"value": False}), (0, {"value": "0"}), (1, {"value": 0}),
                 (2, {}), (2, {"value": None}), (3, {"value": "您好"}), (4, {"value": 1}),
                 (4, {"formula": "=A1+2", "value": 1}), (4, {"formula": "=A1+1"}),
                 (4, {"formula": "=A1+1", "value": True}), (4, {"formula": "=A1+1", "value": "1"})]
        for index, cell in cases:
            broken = deepcopy(payload)
            broken["ranges"][0]["cells"][0][index] = cell
            with self.subTest(index=index, cell=cell):
                self.assertFalse(self.check(broken))
        self.assertTrue(self.check(payload))

    def test_budget_proves_only_returned_coordinates(self):
        partial = self.call(cell_limit=2)
        self.assertTrue(partial["has_more"])
        self.assertEqual(self.records[-1]["proof"]["coordinates"], ["A1", "B1"])
        self.assertFalse(self.check(partial))
        full = self.call()
        formula = self.call(value_render_option="formula")
        self.assertTrue(self.check(merge(full, formula)))
        for limits in ({"max_chars": 1}, {"cell_limit": 1}):
            result = self.call(**limits)
            self.assertEqual(result["returned_cell_count"], len(self.records[-1]["proof"]["coordinates"]))

    def test_revision_and_cache_disagreement_cannot_merge(self):
        raw, formula = self.call(), self.call(value_render_option="formula")
        formula["revision"] += 1
        with self.assertRaises(ValueError):
            merge(raw, formula)
        self.scenario.current["revision"] += 1
        changed = self.call(value_render_option="formula")
        self.scenario.current["revision"] -= 1
        self.assertFalse(self.check(raw), "revision-drift proof fails even if final state restored")
        changed["revision"] = raw["revision"]
        changed["ranges"][0]["cells"][0][-1]["value"] = 2
        with self.assertRaises(ValueError):
            merge(raw, changed)

    def test_actual_file_extent_and_unmodified_backend_required(self):
        payload = self.pair()
        for edit in (lambda p: p.update(has_more=True), lambda p: p.update(spreadsheet_token="foreign"),
                     lambda p: p["ranges"][0].update(truncated=True),
                     lambda p: p["ranges"][0]["cells"][0][0].update(rich_text=[{"text": "foreign"}])):
            broken = deepcopy(payload)
            edit(broken)
            self.assertFalse(self.check(broken))
        for key, value in (("permissions", {"visibility": "public"}), ("notifications", ["sent"]),
                           ("revision", 18), ("cells", [[{"value": False}]])):
            self.scenario.current[key] = value
            self.assertFalse(self.check(payload))
            self.scenario.current = deepcopy(sheets.SOURCE)
        self.records.append({"method": "POST", "uri": sheets.READ_PATH.replace("invoke_read", "invoke_write"), "status": 403})
        self.assertFalse(self.check(payload))

    def test_paths_json_and_bound_identity_claims(self):
        payload = self.pair()
        self.assertFalse(self.check(payload, {"artifact_path": str(self.workspace / "sheet.json")}))
        for content in ('{"ranges":[],"ranges":[]}', '{"ranges":NaN}', 'not json'):
            (self.workspace / "sheet.json").write_text(content)
            self.assertFalse(self.check())
        with tempfile.TemporaryDirectory() as other:
            outside = Path(other) / "outside.json"
            outside.write_text(json.dumps(payload))
            (self.workspace / "escape.json").symlink_to(outside)
            self.assertFalse(self.check(answer={"artifact_path": "escape.json"}))
        bound = {"schema": "feishu-cli-next/result/v1", "binding": dict(sheets.BINDING, user_verified=True, tenant_verified=True),
                 "result": {"ok": True, "data": {"schema": "feishu-cli-next/sheets-data/v1", "spreadsheet_token": "shtSheetsDev01",
                 "sheet_id": "shdev01", "range": "A1:E1", "complete": True, "count": 1, "cells": payload["ranges"][0]["cells"]}}}
        self.assertFalse(self.check(bound))
        self.records.append({"route": "identity", "status": 200, "method": "GET", "uri": "/open-apis/authen/v1/user_info",
                             "signature_valid": True, "proof": deepcopy(sheets.BINDING)})
        self.assertTrue(self.check(bound))
        bound["binding"]["tenant_key"] = "foreign"
        self.assertFalse(self.check(bound))


@unittest.skipUnless(os.environ.get("EVAL_PAIRED_SHEETS_BINARIES"), "optional clean CLI paths required")
class PairedSheetsBinaryTests(unittest.TestCase):
    def test_both_binaries_native_compositions(self):
        from scenario_fixture import ScenarioFixture
        for binary in json.loads(os.environ["EVAL_PAIRED_SHEETS_BINARIES"]):
            strategies = ["api-pair", "api-shortcut"] + (["data-export"] if binary.get("next") else [])
            for strategy in strategies:
                with self.subTest(binary=binary["path"], strategy=strategy), tempfile.TemporaryDirectory() as directory:
                    workspace = Path(directory)
                    for name in ("home", "tmp", "feishu-config"):
                        (workspace / name).mkdir()
                    scenario = sheets.Scenario()
                    with ScenarioFixture(sheets.SCENARIO, scenario) as server:
                        env = {**cli_environment(workspace), **server.environment()}
                        def run(*args):
                            result = subprocess.run([binary["path"], *args, "--as", "user"], cwd=workspace,
                                                    env=env, capture_output=True, text=True, timeout=30)
                            self.assertEqual(result.returncode, 0, result.stderr)
                            return result
                        def api(mode):
                            body = {"tool_name": "get_cell_ranges", "input": json.dumps({"excel_id": "shtSheetsDev01",
                                    "sheet_id": "shdev01", "ranges": ["A1:E1"], "value_render_option": mode})}
                            result = run("api", "POST", sheets.READ_PATH, "--data", json.dumps(body))
                            outer = json.loads(result.stdout)
                            return json.loads(outer["data"]["output"])
                        flags = ("--spreadsheet-token", "shtSheetsDev01", "--sheet-id", "shdev01", "--range", "A1:E1")
                        if strategy == "data-export":
                            run("sheets", "+data-export", *flags, "--save-result", "sheet.json")
                        else:
                            raw = api("raw_value")
                            if strategy == "api-pair":
                                formula = api("formula")
                            else:
                                run("sheets", "+cells-get", *flags, "--include", "value,formula", "--output-path", "formula.json")
                                formula = json.loads((workspace / "formula.json").read_text())
                            (workspace / "sheet.json").write_text(json.dumps(merge(raw, formula), ensure_ascii=False))
                        receipt = server.receipt()
                        self.assertTrue(sheets.verify(workspace, {"artifact_path": "sheet.json"}, receipt, sheets.TASK["expected"]), receipt)
                        self.assertEqual(receipt["state"]["initial"], receipt["state"]["final"])
                        self.assertEqual(receipt["state"]["mutation_attempts"], 0)


if __name__ == "__main__":
    unittest.main()
