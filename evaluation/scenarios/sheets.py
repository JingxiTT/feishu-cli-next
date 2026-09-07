"""Five-cell native Sheets read; producer files and signed reads are checked separately."""
from copy import deepcopy
import json
from pathlib import Path
import re
from urllib.parse import parse_qs, urlsplit

from scenario_protocol import Reply, ScenarioBase

_CONTRACT = json.loads((Path(__file__).resolve().parents[1] / "contracts/sheets.json").read_text())
SOURCE = deepcopy(_CONTRACT["fixture"]["state"])
SOURCE["row_count"], SOURCE["column_count"] = 1, 5
BINDING = {"environment": "feishu", "endpoint": "https://open.feishu.cn",
           "app_id": "synthetic-evaluation-app", "identity": "user",
           "user_id": "ou_sheets_fixture_self", "tenant_key": "sheets-fixture-tenant"}
SCENARIO = _CONTRACT["scenario"]
TASK = {"id": "sheets-dev-01", "section": "sheets", "prompt": _CONTRACT["prompt"],
        "fixture": {"scenario": SCENARIO, "answer_schema": deepcopy(_CONTRACT["artifact_contract"]["final_answer_schema"])},
        "expected": {"source": deepcopy(SOURCE), "binding": deepcopy(BINDING)}}
READ_PATH = "/open-apis/sheet_ai/v2/spreadsheets/shtSheetsDev01/tools/invoke_read"
COORDS = {letter + "1" for letter in "ABCDE"}


def _json(text):
    def pairs(items):
        result = {}
        for key, value in items:
            if key in result:
                raise ValueError("duplicate JSON key")
            result[key] = value
        return result
    def constant(_):
        raise ValueError("non-finite JSON number")
    return json.loads(text, object_pairs_hook=pairs, parse_constant=constant)


def _same(left, right):
    # Python's equality otherwise treats False == 0 and True == 1.
    return json.dumps(left, sort_keys=True, ensure_ascii=False) == json.dumps(right, sort_keys=True, ensure_ascii=False)


def _cell_value_same(value, wanted):
    if type(wanted) in (int, float):
        return type(value) in (int, float) and value == wanted
    return type(value) is type(wanted) and value == wanted


def _columns(a1):
    if not isinstance(a1, str):
        raise ValueError("range must be a string")
    match = re.fullmatch(r"([A-E])1(?::([A-E])1)?", a1)
    if not match:
        raise ValueError("range outside fixture")
    first, last = match[1], match[2] or match[1]
    if first > last:
        raise ValueError("inverted range")
    return [chr(code) for code in range(ord(first), ord(last) + 1)]


def _extent(columns):
    return columns[0] + "1:" + columns[-1] + "1"


def _mutation(method, uri):
    return method not in ("GET", "HEAD", "OPTIONS") and not (method == "POST" and urlsplit(uri).path == READ_PATH)


class Scenario(ScenarioBase):
    def __init__(self):
        self.initial = deepcopy(SOURCE)
        self.current = deepcopy(SOURCE)
        self.mutation_attempts = 0

    def dispatch(self, method, uri, body_bytes, headers):
        parsed = urlsplit(uri)
        path, query = parsed.path, parse_qs(parsed.query, keep_blank_values=True)
        if _mutation(method, uri):
            self.mutation_attempts += 1
        if parsed.scheme or parsed.netloc or parsed.fragment:
            raise ValueError("local API path required")
        if headers.get("X-Lark-Proxy-Identity") != "user":
            raise ValueError("fixture owner identity required")
        if method == "GET" and path == "/open-apis/authen/v1/user_info":
            if query or body_bytes:
                raise ValueError("unexpected identity input")
            return Reply("identity", {"code": 0, "data": {
                "open_id": BINDING["user_id"], "tenant_key": BINDING["tenant_key"],
                "name": "Synthetic Sheets Fixture"}}, proof=deepcopy(BINDING))
        if method == "GET" and path == "/open-apis/sheets/v3/spreadsheets/shtSheetsDev01":
            if body_bytes or set(query) - {"user_id_type"} or query.get("user_id_type", ["open_id"]) != ["open_id"]:
                raise ValueError("unexpected metadata input")
            return Reply("workbook_metadata", {"code": 0, "data": {"spreadsheet": {
                "title": "Synthetic Sheets Fixture", "token": SOURCE["spreadsheet_token"],
                "owner_id": BINDING["user_id"], "url": "https://example.feishu.cn/sheets/shtSheetsDev01"}}})
        if method != "POST" or path != READ_PATH or query:
            raise ValueError("unsupported route")
        body = _json(body_bytes)
        if not isinstance(body, dict) or set(body) != {"tool_name", "input"} or not isinstance(body["input"], str):
            raise ValueError("encoded tool input required")
        data = _json(body["input"])
        if not isinstance(data, dict) or data.get("excel_id") != SOURCE["spreadsheet_token"]:
            raise ValueError("wrong workbook")
        tool = body["tool_name"]
        if tool == "get_workbook_structure":
            if set(data) != {"excel_id"}:
                raise ValueError("unexpected workbook selector")
            output = {"spreadsheet_token": SOURCE["spreadsheet_token"], "title": "Synthetic Sheets Fixture",
                      "revision": SOURCE["revision"], "sheets": [{"sheet_id": "shdev01", "sheet_name": "Fixture",
                      "title": "Fixture", "index": 0, "row_count": 1, "column_count": 5, "hidden": False}]}
            proof = {"operation": tool, "identity": "user"}
            route = "workbook_structure"
        elif tool == "get_cell_ranges":
            output, proof = self._cells(data)
            route = "sheets_cells_read"
        else:
            raise ValueError("unsupported read tool")
        return Reply(route, {"code": 0, "msg": "success", "data": {
            "output": json.dumps(output, ensure_ascii=False)}}, proof=proof)

    def _cells(self, data):
        optional = {"value_render_option", "include_styles", "include_truncation_info",
                    "include_conditional_format_style", "skip_hidden", "cell_limit", "max_chars"}
        if set(data) - ({"excel_id", "sheet_id", "sheet_name", "ranges"} | optional):
            raise ValueError("unknown cell input")
        selectors = set(data) & {"sheet_id", "sheet_name"}
        if selectors not in ({"sheet_id"}, {"sheet_name"}):
            raise ValueError("exactly one sheet selector required")
        if data.get("sheet_id", "shdev01") != "shdev01" or data.get("sheet_name", "Fixture") != "Fixture":
            raise ValueError("foreign sheet")
        render = data.get("value_render_option", "raw_value")
        if render not in ("raw_value", "formula"):
            raise ValueError("unsupported rendering")
        for flag in optional - {"value_render_option", "cell_limit", "max_chars"}:
            if flag in data and type(data[flag]) is not bool:
                raise ValueError("boolean switch required")
        for flag in ("cell_limit", "max_chars"):
            if flag in data and (type(data[flag]) is not int or data[flag] < 1):
                raise ValueError("positive integer limit required")
        requested = data.get("ranges")
        if not isinstance(requested, list) or not requested or len(requested) > 100:
            raise ValueError("bounded nonempty ranges required")
        selections = [_columns(value) for value in requested]
        cell_budget, char_budget = data.get("cell_limit", 1000000000), data.get("max_chars", 500000)
        ranges, coordinates = [], []
        partial = False
        for a1, columns in zip(requested, selections):
            row, returned = [], []
            for column in columns:
                cell = deepcopy(self.current["cells"][0][ord(column) - ord("A")])
                if column == "E" and render == "formula":
                    cell["value"] = cell["formula"]
                if data.get("include_styles"):
                    cell["cell_styles"] = ({"number_format": "General"} if column in "AE" else
                                           {"number_format": "@"} if column in "CD" else {})
                if data.get("include_truncation_info"):
                    cell.update(isRowTruncated=False, isColTruncated=False)
                size = len(json.dumps(cell, ensure_ascii=False))
                if cell_budget < 1 or char_budget < size:
                    break
                cell_budget -= 1
                char_budget -= size
                row.append(cell)
                returned.append(column)
                coordinates.append(column + "1")
            truncated = len(returned) != len(columns)
            partial |= truncated
            ranges.append({"range": a1, "actual_range": _extent(returned) if returned else "",
                           "row_indices": [1] if returned else [], "col_indices": returned,
                           "truncated": truncated, "cells": [row] if returned else []})
        output = {"spreadsheet_token": SOURCE["spreadsheet_token"], "sheet_id": SOURCE["sheet_id"],
                  "has_more": partial, "returned_cell_count": len(coordinates), "ranges": ranges}
        proof = {"operation": "get_cell_ranges", "spreadsheet_token": SOURCE["spreadsheet_token"],
                 "sheet_id": SOURCE["sheet_id"], "identity": "user", "coordinates": coordinates,
                 "ranges": deepcopy(requested), "render": render, "selector": {key: data[key] for key in selectors}}
        return output, proof

    def snapshot(self):
        return {"initial": deepcopy(self.initial), "final": deepcopy(self.current),
                "mutation_attempts": self.mutation_attempts}


def _native_cells(payload):
    if not isinstance(payload, dict) or any(payload.get(key, False) is not False for key in ("has_more", "truncated", "is_truncated")):
        raise ValueError("partial or invalid cell payload")
    for key in ("spreadsheet_token", "sheet_id"):
        if key in payload and payload[key] != SOURCE[key]:
            raise ValueError("wrong source metadata")
    ranges = payload.get("ranges")
    if not isinstance(ranges, list) or not ranges:
        raise ValueError("native ranges required")
    cells = {}
    for region in ranges:
        if not isinstance(region, dict) or any(region.get(key, False) is not False for key in ("has_more", "truncated", "is_truncated")):
            raise ValueError("truncated region")
        columns = _columns(region.get("actual_range", region.get("range")))
        if "range" in region:
            _columns(region["range"])
        rows = region.get("row_indices", [1])
        if not _same(rows, [1]) or region.get("col_indices", columns) != columns:
            raise ValueError("coordinate mismatch")
        matrix = region.get("cells")
        if not isinstance(matrix, list) or len(matrix) != 1 or not isinstance(matrix[0], list) or len(matrix[0]) != len(columns):
            raise ValueError("grid extent mismatch")
        for column, cell in zip(columns, matrix[0]):
            if not isinstance(cell, dict):
                raise ValueError("native cell object required")
            if any(cell.get(key, False) is not False for key in ("isRowTruncated", "isColTruncated", "truncated", "is_truncated")):
                raise ValueError("truncated native cell")
            if any(cell.get(key) for key in ("rich_text", "multiple_values")):
                raise ValueError("unexpected native content identity")
            coordinate = column + "1"
            if coordinate in cells and not _same(cells[coordinate], cell):
                raise ValueError("conflicting duplicate cell")
            cells[coordinate] = cell
    if set(cells) != COORDS:
        raise ValueError("incomplete target coverage")
    return cells


def _bound_payload(artifact, records, binding):
    actual = artifact.get("binding")
    if not isinstance(actual, dict):
        raise ValueError("binding required")
    for key in ("environment", "app_id", "identity", "user_id"):
        if actual.get(key) != binding[key]:
            raise ValueError("wrong artifact binding")
    for key in ("endpoint", "tenant_key"):
        if key in actual and actual[key] != binding[key]:
            raise ValueError("wrong known binding")
    for key in ("tenant_verified", "user_verified"):
        if key in actual and type(actual[key]) is not bool:
            raise ValueError("invalid verification claim")
        if actual.get(key):
            if key == "tenant_verified" and actual.get("tenant_key") != binding["tenant_key"]:
                raise ValueError("missing verified tenant")
            if not any(record.get("route") == "identity" and record.get("status") == 200 and
                       _same(record.get("proof"), binding) for record in records):
                raise ValueError("unobserved verified identity")
    result = artifact.get("result")
    if not isinstance(result, dict) or result.get("ok") is not True:
        raise ValueError("successful result required")
    payload = result.get("data")
    if not isinstance(payload, dict) or payload.get("schema") != "feishu-cli-next/sheets-data/v1":
        raise ValueError("native data schema required")
    if any(payload.get(key) != SOURCE[key] for key in ("spreadsheet_token", "sheet_id", "range")):
        raise ValueError("wrong source extent")
    if payload.get("complete") is not True or type(payload.get("count")) is not int or payload["count"] != 1:
        raise ValueError("incomplete export")
    return {"ranges": [{"range": payload["range"], "cells": payload.get("cells")}]}


def verify(workspace, answer, receipt, expected):
    try:
        if not isinstance(answer, dict) or set(answer) != {"artifact_path"}:
            return False
        relative = answer["artifact_path"]
        if not isinstance(relative, str) or not relative or Path(relative).is_absolute():
            return False
        root = Path(workspace).resolve()
        artifact_path = (root / relative).resolve(strict=True)
        if not artifact_path.is_relative_to(root) or not artifact_path.is_file() or artifact_path.stat().st_size > 1000000:
            return False
        if receipt.get("scenario") != SCENARIO:
            return False
        state = receipt.get("state", {})
        if not _same(state.get("initial"), expected["source"]) or not _same(state.get("final"), expected["source"]) or type(state.get("mutation_attempts")) is not int or state["mutation_attempts"] != 0:
            return False
        records, observed = receipt.get("requests", []), set()
        for record in records:
            if "method" in record and _mutation(record["method"], record.get("uri", "")):
                return False
            proof = record.get("proof", {})
            if record.get("route") == "sheets_cells_read" and record.get("status") == 200:
                if proof.get("operation") != "get_cell_ranges" or proof.get("identity") != "user" or any(proof.get(key) != SOURCE[key] for key in ("spreadsheet_token", "sheet_id")):
                    return False
                observed.update(proof.get("coordinates", []))
        if observed != COORDS:
            return False
        artifact = _json(artifact_path.read_text(encoding="utf-8"))
        if not isinstance(artifact, dict):
            return False
        if artifact.get("schema") == "feishu-cli-next/result/v1":
            payload = _bound_payload(artifact, records, expected["binding"])
        elif "schema" in artifact:
            return False
        elif "ok" in artifact:
            if artifact["ok"] is not True:
                return False
            payload = artifact.get("data")
        else:
            payload = artifact
        cells = _native_cells(payload)
        for column, wanted in zip("ABCD", expected["source"]["cells"][0][:4]):
            cell = cells[column + "1"]
            if "value" not in cell or not _cell_value_same(cell["value"], wanted["value"]):
                return False
            if cell.get("formula") not in (None, ""):
                return False
        formula = cells["E1"]
        if formula.get("formula") != "=A1+1":
            return False
        if "value" in formula and not (_cell_value_same(formula["value"], 1) or _cell_value_same(formula["value"], "=A1+1")):
            return False
        return True
    except (OSError, UnicodeError, ValueError, TypeError, KeyError, AttributeError):
        return False
