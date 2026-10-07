# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Development-only native export scaling on the existing signed scenario protocol."""
from copy import deepcopy
from decimal import Decimal, InvalidOperation
import csv
import hashlib
import io
import json
from pathlib import Path
import re
from urllib.parse import parse_qs, urlsplit

from scenario_protocol import Reply, ScenarioBase

SCENARIO = "scaling-sheets-v2"
SIZES = (100, 1000, 10000)
TOKEN, SHEET = "shtScaleDev01", "shscale01"
READ_PATH = f"/open-apis/sheet_ai/v2/spreadsheets/{TOKEN}/tools/invoke_read"
BINDING = {"environment": "feishu", "endpoint": "https://open.feishu.cn",
           "app_id": "synthetic-evaluation-app", "identity": "user",
           "user_id": "ou_scale_fixture_self", "tenant_key": "scale-fixture-tenant"}
FLAGS = ("has_more", "truncated", "is_truncated", "isRowTruncated", "isColTruncated")


def _size(count):
    if type(count) is not int or count not in SIZES:
        raise ValueError("unsupported development row count")
    return count


def _bytes(value):
    return json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":"), allow_nan=False).encode()


def _load(raw):
    def pairs(items):
        result = {}
        for key, value in items:
            if key in result:
                raise ValueError("duplicate JSON key")
            result[key] = value
        return result
    def constant(_):
        raise ValueError("non-finite JSON number")
    return json.loads(raw, object_pairs_hook=pairs, parse_float=Decimal, parse_constant=constant)


def _row(index):
    number = (0, -3.5, 0.125, 9007199254740991)[index % 4]
    return [{"value": f"scale-native-v1:{index:06d}:你好\n✓"}, {"value": number},
            {"value": index % 2 == 1}, {"value": ""}, {"value": None}, {},
            {"value": f"{index:08d}" if index % 2 == 0 else f"=literal-{index:06d}"},
            {"formula": f"=B{index + 1}+1", "value": number + 1}]


def source_rows(row_count):
    return [_row(index) for index in range(_size(row_count))]


def _number(value):
    if type(value) not in (int, float, Decimal):
        raise ValueError("number required, without boolean coercion")
    decimal = Decimal(str(value))
    if not decimal.is_finite():
        raise ValueError("finite number required")
    if not decimal:
        return "0"
    # Generated values are bounded; reject malicious huge exponents before formatting.
    if decimal.adjusted() > 20 or decimal.adjusted() < -30:
        raise ValueError("number outside fixture precision")
    text = format(decimal, "f")
    return text.rstrip("0").rstrip(".") if "." in text else text


def _style(column):
    return ({"number_format": "General"} if column in (1, 7) else
            {"number_format": "@"} if column in (0, 3, 6) else {})


def _token(cell, row, column):
    if not isinstance(cell, dict) or set(cell) - {"value", "formula", "cell_styles", "rich_text", "multiple_values", *FLAGS}:
        raise ValueError("unexpected native cell shape")
    if any(cell.get(key, False) is not False for key in FLAGS):
        raise ValueError("truncated cell")
    if any(cell.get(key) not in (None, []) for key in ("rich_text", "multiple_values")):
        raise ValueError("conflicting content identity")
    if "cell_styles" in cell and cell["cell_styles"] != _style(column):
        raise ValueError("unexpected cell style")
    wanted = _row(row - 1)[column]
    if column == 7:
        if cell.get("formula") != wanted["formula"]:
            raise ValueError("formula identity lost")
        if "value" in cell and cell["value"] != wanted["formula"]:
            if _number(cell["value"]) != _number(wanted["value"]):
                raise ValueError("incorrect formula cache")
        return ["formula", wanted["formula"]]
    if cell.get("formula") not in (None, ""):
        raise ValueError("static cell became formula")
    if "value" not in wanted:
        if "value" in cell:
            raise ValueError("omitted value became present")
        return ["omitted"]
    if "value" not in cell:
        raise ValueError("present value omitted")
    value, expected = cell["value"], wanted["value"]
    if type(expected) in (int, float):
        if _number(value) != _number(expected):
            raise ValueError("numeric value changed")
        return ["number", _number(value)]
    if type(value) is not type(expected) or value != expected:
        raise ValueError("value or JSON type changed")
    return (["null"] if value is None else ["boolean", value] if type(value) is bool else ["string", value])


def _hash_cells(cells):
    digest = hashlib.sha256()
    for row, column, cell in cells:
        digest.update(_bytes([row, column, _token(cell, row, column)]) + b"\n")
    return digest.hexdigest()


def semantic_hash(matrix):
    _size(len(matrix))
    if any(not isinstance(row, list) or len(row) != 8 for row in matrix):
        raise ValueError("eight columns required")
    return _hash_cells((r + 1, c, cell) for r, row in enumerate(matrix) for c, cell in enumerate(row))


# Synthetic structure declaration: the frozen reference names merged_cells
# but does not publish its JSON schema; [] represents this fixture's no merges.
def _source(count):
    return {"spreadsheet_token": TOKEN, "sheet_id": SHEET, "range": f"A1:H{count}",
            "revision": 1, "row_count": count, "column_count": 8, "cells": source_rows(count),
            "permissions": {"visibility": "self_only", "owner_id": BINDING["user_id"], "collaborators": []},
            "structure": {"merged_cells": [], "frozen_row_count": 0, "frozen_column_count": 0},
            "notifications": [], "sentinels": {"outside_range": "preserve synthetic neighbour", "sheet_schema": "native-eight-v1"}}


def _snapshot(source):
    result = {key: deepcopy(value) for key, value in source.items() if key != "cells"}
    result["state_sha256"] = hashlib.sha256(_bytes(source)).hexdigest()
    try:
        result["typed_hash"] = semantic_hash(source["cells"])
    except (ValueError, TypeError, KeyError):
        result["typed_hash"] = None
    return result


def task(row_count):
    count = _size(row_count)
    return {"id": f"sheets-size-dev-{count}", "section": "sheets",
            "prompt": f"Read all {count} rows and all eight columns in A1:H{count} of synthetic spreadsheet {TOKEN}, sheet ID {SHEET} (Scale), into a local JSON artifact. Preserve native values and JSON types, including zero, false, empty text, null, omitted value fields, Unicode, leading-zero strings, literal text beginning =, and explicit formula fields. Keep the source unchanged. Use any available read/export commands or local processing. Keep the full cell matrix in the artifact rather than your final answer. Finish with JSON containing artifact_path, relative to your workspace.",
            "fixture": {"scenario": SCENARIO, "row_count": count, "answer_schema": {
                "type": "object", "properties": {"artifact_path": {"type": "string"}},
                "required": ["artifact_path"], "additionalProperties": False}},
            "expected": {"row_count": count, "source": _snapshot(_source(count)), "binding": deepcopy(BINDING)}}


def _range(value, count):
    if not isinstance(value, str):
        raise ValueError("range must be string")
    match = re.fullmatch(r"([A-H])([1-9][0-9]{0,4})(?::([A-H])([1-9][0-9]{0,4}))?", value)
    if not match:
        raise ValueError("invalid fixture range")
    first, last = int(match[2]), int(match[4] or match[2])
    left, right = ord(match[1]) - 65, ord(match[3] or match[1]) - 65
    if first > last or last > count or left > right:
        raise ValueError("range outside fixture")
    return first, last, left, right


def _extent(first, last, left, right):
    return f"{chr(65 + left)}{first}:{chr(65 + right)}{last}"


def _coordinates(region, count):
    first, last, left, right = _range(region, count)
    return ((row, col) for row in range(first, last + 1) for col in range(left, right + 1))


def _mutation(method, uri):
    return method not in ("GET", "HEAD", "OPTIONS") and not (method == "POST" and urlsplit(uri).path == READ_PATH)


class Scenario(ScenarioBase):
    def __init__(self, row_count):
        self.row_count = _size(row_count)
        self.current = _source(row_count)
        self.initial = _snapshot(self.current)
        self.mutation_attempts = 0

    def dispatch(self, method, uri, body_bytes, headers):
        if _mutation(method, uri):
            self.mutation_attempts += 1
        parsed = urlsplit(uri)
        query = parse_qs(parsed.query, keep_blank_values=True)
        if parsed.scheme or parsed.netloc or parsed.fragment or headers.get("X-Lark-Proxy-Identity") != "user":
            raise ValueError("local fixture owner identity required")
        if method == "GET" and parsed.path == "/open-apis/authen/v1/user_info" and not query and not body_bytes:
            return Reply("identity", {"code": 0, "data": {"open_id": BINDING["user_id"],
                         "tenant_key": BINDING["tenant_key"], "name": "Synthetic Scale Fixture"}}, proof=deepcopy(BINDING))
        if method == "GET" and parsed.path == f"/open-apis/sheets/v3/spreadsheets/{TOKEN}" and not body_bytes:
            if set(query) - {"user_id_type"} or query.get("user_id_type", ["open_id"]) != ["open_id"]:
                raise ValueError("invalid metadata query")
            return Reply("workbook_metadata", {"code": 0, "data": {"spreadsheet": {
                "token": TOKEN, "title": "Synthetic Scale Fixture", "owner_id": BINDING["user_id"]}}})
        if method != "POST" or parsed.path != READ_PATH or query:
            raise ValueError("unsupported route")
        body = _load(body_bytes)
        if not isinstance(body, dict) or set(body) != {"tool_name", "input"} or not isinstance(body["input"], str):
            raise ValueError("encoded tool input required")
        data = _load(body["input"])
        if not isinstance(data, dict) or data.get("excel_id") != TOKEN:
            raise ValueError("wrong workbook")
        if body["tool_name"] == "get_workbook_structure" and set(data) == {"excel_id"}:
            output = {"spreadsheet_token": TOKEN, "title": "Synthetic Scale Fixture", "revision": 1,
                      "sheets": [{"sheet_id": SHEET, "sheet_name": "Scale", "title": "Scale", "index": 0,
                                  "row_count": self.row_count, "column_count": 8, "hidden": False}]}
            route, proof = "workbook_structure", {"operation": "get_workbook_structure"}
        elif body["tool_name"] == "get_cell_ranges":
            output, proof = self._cells(data)
            route = "sheets_cells_read"
        elif body["tool_name"] == "get_sheet_structure":
            output = self._structure(data)
            route, proof = "sheet_structure", {"operation": "get_sheet_structure", "identity": "user"}
        elif body["tool_name"] == "get_range_as_csv":
            output = self._csv(data)
            route, proof = "sheets_csv_preview", {"operation": "get_range_as_csv", "identity": "user"}
        else:
            raise ValueError("unsupported read operation")
        return Reply(route, {"code": 0, "msg": "success", "data": {"output": json.dumps(output, ensure_ascii=False)}}, proof=proof)

    def _selector(self, data):
        selectors = set(data) & {"sheet_id", "sheet_name"}
        if not selectors or data.get("sheet_id", SHEET) != SHEET or data.get("sheet_name", "Scale") != "Scale":
            raise ValueError("wrong sheet selector")

    def _structure(self, data):
        self._selector(data)
        if set(data) - {"excel_id", "sheet_id", "sheet_name", "range", "info_type"}:
            raise ValueError("unknown structure input")
        if "range" in data:
            _range(data["range"], self.row_count)
        info = data.get("info_type", "all")
        if type(info) is not str or info not in ("all", "merged_cells_infos", "row_heights_column_widths", "hidden_infos", "group_infos"):
            raise ValueError("invalid sheet structure info_type")
        if info in ("row_heights_column_widths", "hidden_infos", "group_infos"):
            # The frozen reference names other categories but supplies no native
            # response shape for their overrides. Do not invent empty objects.
            raise ValueError("unsupported strategy: sheet structure category has no sourced fixture representation")
        return {"sheet_id": SHEET, "sheet_name": "Scale", "revision": self.current["revision"],
                "row_count": self.row_count, "column_count": 8, **deepcopy(self.current["structure"])}

    def _csv(self, data):
        self._selector(data)
        if set(data) - {"excel_id", "sheet_id", "sheet_name", "range", "max_rows", "max_chars", "skip_hidden"}:
            raise ValueError("unknown CSV input")
        if "skip_hidden" in data and type(data["skip_hidden"]) is not bool:
            raise ValueError("boolean skip_hidden required")
        for key in ("max_rows", "max_chars"):
            if key in data and (type(data[key]) is not int or data[key] < 1):
                raise ValueError("positive CSV limit required")
        region = data.get("range")
        whole = re.fullmatch(r"([A-Z]{1,3}):([A-Z]{1,3})", region) if isinstance(region, str) else None
        if whole:
            def column(label):
                result = 0
                for letter in label:
                    result = result * 26 + ord(letter) - 64
                return result - 1
            left, requested_right = column(whole[1]), column(whole[2])
            if left > requested_right or left > 7:
                raise ValueError("CSV columns outside fixture")
            first, last, right = 1, self.row_count, min(7, requested_right)
        else:
            first, last, left, right = _range(region, self.row_count)
        remaining = data.get("max_chars", 500000)
        max_rows = data.get("max_rows", 1000000000)
        used = self._current_region(first, last, left, right)
        records = []
        for row in range(first, last + 1):
            if len(records) >= max_rows:
                break
            values = []
            for col in range(left, right + 1):
                cell = self.current["cells"][row - 1][col]
                value = cell.get("value")
                if value is None:
                    text = ""
                elif type(value) is bool:
                    text = "true" if value else "false"
                elif type(value) in (int, float):
                    text = _number(value)
                else:
                    text = value
                values.append(text)
            stream = io.StringIO(newline="")
            csv.writer(stream, lineterminator="\n").writerow(values)
            # Match frozen stripRowPrefixFromCsvOutput, which removes a comma.
            rendered = f"[row={row}]," + stream.getvalue()
            if len(rendered) > remaining:
                break
            remaining -= len(rendered)
            records.append(rendered)
        actual = _extent(first, first + len(records) - 1, left, right) if records else ""
        truncated = len(records) != last - first + 1
        return {"spreadsheet_token": TOKEN, "sheet_id": SHEET, "range": region,
                "actual_range": actual, "current_region": used, "row_count": len(records),
                "row_indices": list(range(first, first + len(records))),
                "col_indices": [chr(65 + col) for col in range(left, right + 1)] if records else [],
                "warning_message": "Read truncated; use actual_range and narrower ranges to continue." if truncated else "Use row_indices and col_indices for source coordinates.",
                "annotated_csv": "".join(records), "has_more": truncated, "truncated": truncated}

    def _current_region(self, first, last, left, right):
        # Frozen read-data reference: expand the requested source region to
        # surrounding blank rows/columns. Preview budgets do not limit it.
        def occupied(row, col):
            cell = self.current["cells"][row - 1][col]
            return "formula" in cell or cell.get("value") is not None and cell.get("value") != ""

        used = [(row, col) for row in range(first, last + 1)
                for col in range(left, right + 1) if occupied(row, col)]
        if not used:
            raise ValueError("unsupported strategy: empty-only CSV source region has no sourced fixture representation")
        first, last = min(row for row, _ in used), max(row for row, _ in used)
        left, right = min(col for _, col in used), max(col for _, col in used)
        while True:
            previous = first, last, left, right
            while first > 1 and any(occupied(first - 1, col) for col in range(left, right + 1)):
                first -= 1
            while last < self.row_count and any(occupied(last + 1, col) for col in range(left, right + 1)):
                last += 1
            while left > 0 and any(occupied(row, left - 1) for row in range(first, last + 1)):
                left -= 1
            while right < 7 and any(occupied(row, right + 1) for row in range(first, last + 1)):
                right += 1
            if previous == (first, last, left, right):
                return _extent(first, last, left, right)

    def _cells(self, data):
        optional = {"value_render_option", "include_styles", "include_truncation_info",
                    "include_conditional_format_style", "skip_hidden", "cell_limit", "max_chars"}
        if set(data) - ({"excel_id", "sheet_id", "sheet_name", "ranges"} | optional):
            raise ValueError("unknown cell input")
        self._selector(data)
        render = data.get("value_render_option", "raw_value")
        if render not in ("raw_value", "formula"):
            raise ValueError("unsupported rendering")
        for key in optional - {"value_render_option", "cell_limit", "max_chars"}:
            if key in data and type(data[key]) is not bool:
                raise ValueError("boolean switch required")
        for key in ("cell_limit", "max_chars"):
            if key in data and (type(data[key]) is not int or data[key] < 1):
                raise ValueError("positive integer limit required")
        requested = data.get("ranges")
        if not isinstance(requested, list) or not requested or len(requested) > 100:
            raise ValueError("bounded nonempty ranges required")
        selections = [_range(region, self.row_count) for region in requested]
        cells_left, chars_left = data.get("cell_limit", 1000000000), data.get("max_chars", 500000)
        ranges, returned, hash_cells = [], [], []
        partial, total = False, 0
        for region, (first, last, left, right) in zip(requested, selections):
            matrix = []
            for row in range(first, last + 1):
                native = deepcopy(self.current["cells"][row - 1][left:right + 1])
                for column, cell in zip(range(left, right + 1), native):
                    if column == 7 and render == "formula":
                        cell["value"] = cell["formula"]
                    if data.get("include_styles") or data.get("include_conditional_format_style"):
                        cell["cell_styles"] = _style(column)
                    if data.get("include_truncation_info"):
                        cell.update(isRowTruncated=False, isColTruncated=False)
                size = len(json.dumps(native, ensure_ascii=False))
                if cells_left < len(native) or chars_left < size:
                    break
                cells_left -= len(native)
                chars_left -= size
                total += len(native)
                matrix.append(native)
                hash_cells.extend((row, col, cell) for col, cell in zip(range(left, right + 1), native))
            truncated = len(matrix) != last - first + 1
            partial |= truncated
            actual = _extent(first, first + len(matrix) - 1, left, right) if matrix else ""
            if actual:
                returned.append(actual)
            ranges.append({"range": region, "actual_range": actual, "truncated": truncated,
                           "row_indices": list(range(first, first + len(matrix))),
                           "col_indices": [chr(65 + col) for col in range(left, right + 1)] if matrix else [], "cells": matrix})
        proof = {"operation": "get_cell_ranges", "identity": "user", "spreadsheet_token": TOKEN,
                 "sheet_id": SHEET, "ranges": requested, "returned_ranges": returned,
                 "returned_cell_count": total, "typed_hash": _hash_cells(hash_cells), "render": render}
        return {"spreadsheet_token": TOKEN, "sheet_id": SHEET, "has_more": partial,
                "returned_cell_count": total, "ranges": ranges}, proof

    def snapshot(self):
        return {"initial": deepcopy(self.initial), "final": _snapshot(self.current), "mutation_attempts": self.mutation_attempts}


def _complete(value):
    if not isinstance(value, dict) or any(value.get(key, False) is not False for key in FLAGS):
        raise ValueError("incomplete native data")


def _payload(artifact, records, expected):
    _complete(artifact)
    count, binding = expected["row_count"], expected["binding"]
    if artifact.get("schema") == "feishu-cli-next/result/v1":
        actual = artifact.get("binding", {})
        for key in ("environment", "app_id", "identity", "user_id"):
            if actual.get(key) != binding[key]:
                raise ValueError("wrong bound identity")
        for key in ("endpoint", "tenant_key"):
            if key in actual and actual[key] != binding[key]:
                raise ValueError("wrong known binding")
        for key in ("user_verified", "tenant_verified"):
            if key in actual and type(actual[key]) is not bool:
                raise ValueError("invalid verification claim")
            if actual.get(key):
                if key == "tenant_verified" and actual.get("tenant_key") != binding["tenant_key"]:
                    raise ValueError("missing verified tenant")
                if not any(record.get("route") == "identity" and record.get("status") == 200 and record.get("proof") == binding for record in records):
                    raise ValueError("unobserved verified binding")
        result = artifact["result"]
        _complete(result)
        if result.get("ok") is not True:
            raise ValueError("unsuccessful artifact")
        data = result["data"]
        _complete(data)
        if data.get("schema") != "feishu-cli-next/sheets-data/v1" or data.get("complete") is not True or type(data.get("count")) is not int or data["count"] != count:
            raise ValueError("incomplete bound cells")
        if data.get("spreadsheet_token") != TOKEN or data.get("sheet_id") != SHEET or data.get("range") != f"A1:H{count}":
            raise ValueError("wrong bound source extent")
        return {"ranges": [{"range": data["range"], "cells": data["cells"]}]}
    if "schema" in artifact:
        raise ValueError("unknown artifact schema")
    if "ok" in artifact:
        if artifact["ok"] is not True:
            raise ValueError("unsuccessful native result")
        return artifact.get("data")
    return artifact


def verify(workspace, answer, receipt, expected):
    try:
        count = _size(expected["row_count"])
        if not isinstance(answer, dict) or set(answer) != {"artifact_path"}:
            return False
        relative = answer["artifact_path"]
        if not isinstance(relative, str) or not relative or Path(relative).is_absolute():
            return False
        root = Path(workspace).resolve()
        path = (root / relative).resolve(strict=True)
        if not path.is_relative_to(root) or not path.is_file() or path.stat().st_size > 32 * 1024 * 1024:
            return False
        state = receipt.get("state", {})
        if receipt.get("scenario") != SCENARIO or type(state.get("mutation_attempts")) is not int or state["mutation_attempts"] != 0:
            return False
        if _bytes(state.get("initial")) != _bytes(expected["source"]) or _bytes(state.get("final")) != _bytes(expected["source"]):
            return False
        records, observed = receipt.get("requests", []), set()
        for record in records:
            if "method" in record and _mutation(record["method"], record.get("uri", "")):
                return False
            if record.get("route") != "sheets_cells_read" or record.get("status") != 200:
                continue
            proof = record.get("proof", {})
            if proof.get("operation") != "get_cell_ranges" or proof.get("identity") != "user" or proof.get("spreadsheet_token") != TOKEN or proof.get("sheet_id") != SHEET:
                return False
            coordinates = [coordinate for region in proof["returned_ranges"] for coordinate in _coordinates(region, count)]
            if type(proof.get("returned_cell_count")) is not int or len(coordinates) != proof["returned_cell_count"]:
                return False
            if proof.get("typed_hash") != _hash_cells((row, col, _row(row - 1)[col]) for row, col in coordinates):
                return False
            observed.update(coordinates)
        if len(observed) != count * 8:
            return False
        artifact = _load(path.read_text(encoding="utf-8"))
        if not isinstance(artifact, dict):
            return False
        payload = _payload(artifact, records, expected)
        _complete(payload)
        if any(key in payload and payload[key] != value for key, value in (("spreadsheet_token", TOKEN), ("sheet_id", SHEET))):
            return False
        regions = payload.get("ranges")
        if not isinstance(regions, list) or not regions:
            return False
        cells = {}
        for region in regions:
            _complete(region)
            first, last, left, right = _range(region.get("actual_range", region.get("range")), count)
            if "range" in region:
                _range(region["range"], count)
            rows, columns = list(range(first, last + 1)), [chr(65 + col) for col in range(left, right + 1)]
            if region.get("row_indices", rows) != rows or any(type(row) is not int for row in region.get("row_indices", rows)) or region.get("col_indices", columns) != columns:
                return False
            matrix = region["cells"]
            if not isinstance(matrix, list) or len(matrix) != len(rows):
                return False
            for row, native in zip(rows, matrix):
                if not isinstance(native, list) or len(native) != len(columns):
                    return False
                for column, cell in zip(range(left, right + 1), native):
                    token = _token(cell, row, column)
                    if (row, column) in cells and cells[row, column] != token:
                        return False
                    cells[row, column] = token
        if set(cells) != observed:
            return False
        digest = hashlib.sha256()
        for (row, column), token in sorted(cells.items()):
            digest.update(_bytes([row, column, token]) + b"\n")
        return digest.hexdigest() == expected["source"]["typed_hash"]
    except (OSError, UnicodeError, ValueError, TypeError, KeyError, AttributeError, OverflowError, RecursionError, InvalidOperation):
        return False
