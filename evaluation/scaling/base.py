"""Development-only Base import size family; no model or live network calls."""
from copy import deepcopy
import hashlib
import json
import math
from pathlib import Path
import threading
from urllib.parse import parse_qs, urlsplit

from scenario_protocol import Reply, ScenarioBase

SCENARIO = "scaling-base"
SIZES = (100, 1000, 10000)
BASE, TABLE = "bas_scale_dev_2026", "tbl_scale_import_01"
PREFIX = f"/open-apis/base/v3/bases/{BASE}/tables/{TABLE}"
SENTINEL = "rec_scale_preserved"
FIELDS = [{"id": field_id, "name": name, "type": kind} for field_id, name, kind in (
    ("fld_scale_key", "Row key", "text"), ("fld_scale_number", "Number", "number"),
    ("fld_scale_checked", "Checked", "checkbox"), ("fld_scale_nullable", "Nullable", "text"),
    ("fld_scale_optional", "Optional", "text"), ("fld_scale_empty", "Empty", "text"),
    ("fld_scale_code", "Code", "text"))]
MAPPING = [{"column": i, "field_id": f["id"], "type": f["type"]} for i, f in enumerate(FIELDS)]
INITIAL = {SENTINEL: {"fld_scale_key": "preexisting-control", "fld_scale_number": 23.5,
                     "fld_scale_checked": True, "fld_scale_nullable": None,
                     "fld_scale_optional": "retain this cell", "fld_scale_empty": "preexisting nonempty",
                     "fld_scale_code": "009900"}}
BINDING = {"environment": "feishu", "endpoint": "https://open.feishu.cn", "app_id": "synthetic-evaluation-app",
           "identity": "user", "user_id": "fixture_self", "tenant_key": "fixture-tenant",
           "user_verified": True, "tenant_verified": True}
ROUTES = {
    "identity": ("GET", "/open-apis/authen/v1/user_info"),
    "base_scale_fields": ("GET", PREFIX + "/fields"),
    "base_scale_field": ("GET", PREFIX + "/fields/"),
    "base_scale_list": ("GET", PREFIX + "/records"),
    "base_scale_get": ("POST", PREFIX + "/records/batch_get"),
    "base_scale_create": ("POST", PREFIX + "/records"),
    "base_scale_batch_create": ("POST", PREFIX + "/records/batch_create"),
}
WRITES = {"base_scale_create", "base_scale_batch_create"}


def _size(n):
    if type(n) is not int or n not in SIZES:
        raise ValueError("row_count must be 100, 1000 or 10000")
    return n


def source_row(i):
    number = 9007199254740991 if i % 31 == 30 else (0 if i % 5 == 0 else -(i % 97) / 4)
    return [{"value": f"測定-{i:05d}-café-🧪"}, {"value": number}, {"value": i % 2 == 0},
            {"value": None if i % 3 == 0 else f"note-{i:05d}"},
            {} if i % 4 == 0 else {"value": f"optional-{i:05d}"}, {"value": ""}, {"value": f"{i:06d}"}]


def native_row(i):
    cells = source_row(i)
    return {m["field_id"]: cells[m["column"]]["value"] for m in MAPPING if "value" in cells[m["column"]]}


def _encode(value):
    return json.dumps(value, ensure_ascii=False, separators=(",", ":"), sort_keys=True, allow_nan=False).encode("utf-8")


def _digest(raw):
    return hashlib.sha256(raw).hexdigest()


def input_bytes(row_count):
    return _encode({"schema": "feishu-cli-next/result/v1", "binding": BINDING, "result": {"ok": True, "data": {
        "schema": "feishu-cli-next/sheets-data/v1", "spreadsheet_token": "sheet_scale_development_source",
        "sheet_id": "tab_scale_development", "range": f"A1:G{_size(row_count)}", "complete": True,
        "count": row_count, "cells": [source_row(i) for i in range(row_count)]}}})


def task(row_count):
    n = _size(row_count)
    return {"id": f"base-scale-dev-{n:05d}", "section": "base", "prompt": (
        f"Import all {n} rows from input.json into Base {BASE}, table {TABLE}, using the column mapping in mapping.json. "
        "Preserve every supported value and its type, including Unicode, numeric zero, false, null, empty text, numeric-looking text and omitted cells. "
        "Append each source row once; preserve existing rows and table schema. Save a JSON report to import-report.json with created and record_ids for the imported records. "
        "Return imported_count and report_path. Do not include row contents in the final answer."),
        "fixture": {"scenario": SCENARIO, "row_count": n, "allow_synthetic_mutations": True, "answer_schema": {
            "type": "object", "required": ["imported_count", "report_path"], "properties": {
                "imported_count": {"type": "integer"}, "report_path": {"type": "string"}}, "additionalProperties": False}},
        "expected": {"row_count": n, "input_sha256": _digest(input_bytes(n)),
                     "mapping_sha256": _digest(_encode(MAPPING))}}


def _finite(value):
    try:
        return math.isfinite(value)
    except OverflowError:
        return False


def _equal(a, b):
    if isinstance(a, bool) or isinstance(b, bool):
        return type(a) is type(b) and a == b
    if type(a) in (int, float) and type(b) in (int, float):
        return _finite(a) and _finite(b) and a == b
    if type(a) is not type(b):
        return False
    if isinstance(a, dict):
        return a.keys() == b.keys() and all(_equal(a[k], b[k]) for k in a)
    if isinstance(a, list):
        return len(a) == len(b) and all(_equal(x, y) for x, y in zip(a, b))
    return a == b


def _object(raw):
    def pairs(items):
        value = {}
        for k, v in items:
            if k in value:
                raise ValueError("duplicate JSON key")
            value[k] = v
        return value
    def invalid(_):
        raise ValueError("nonfinite JSON number")
    value = json.loads(raw or b"{}", object_pairs_hook=pairs, parse_constant=invalid)
    if not isinstance(value, dict):
        raise ValueError("JSON object required")
    return value


def _field(name):
    for f in FIELDS:
        if name in (f["id"], f["name"]):
            return f
    raise ValueError("unknown field")


def _projection(values):
    if values is None:
        return FIELDS
    if not isinstance(values, list) or not values or not all(isinstance(v, str) for v in values):
        raise ValueError("invalid field projection")
    fields = [_field(v) for v in values]
    if len({f["id"] for f in fields}) != len(fields):
        raise ValueError("duplicate field projection")
    return fields


def _page(query, maximum):
    result = []
    for key, default in (("offset", "0"), ("limit", str(maximum))):
        values = query.get(key, [default])
        if len(values) != 1 or not values[0].isascii() or not values[0].isdigit():
            raise ValueError("invalid pagination")
        result.append(int(values[0]))
    if not 1 <= result[1] <= maximum:
        raise ValueError("invalid limit")
    return result


class Scenario(ScenarioBase):
    def __init__(self, row_count):
        self.row_count = _size(row_count)
        self.records = deepcopy(INITIAL)
        self.effects = []
        self.keys = {INITIAL[SENTINEL]["fld_scale_key"]}
        self.lock = threading.Lock()
        self.input = input_bytes(row_count)
        self.mapping = _encode(MAPPING)

    def setup(self, workspace):
        self.workspace = Path(workspace)
        for name, raw in (("input.json", self.input), ("mapping.json", self.mapping)):
            with (self.workspace / name).open("xb") as stream:
                stream.write(raw)

    def snapshot(self):
        with self.lock:
            return deepcopy({"row_count": self.row_count, "initial": INITIAL, "fields": FIELDS,
                             "records": self.records, "effects": self.effects,
                             "input_sha256": _digest(self.input), "mapping_sha256": _digest(self.mapping)})

    def dispatch(self, method, uri, body_bytes, headers):
        with self.lock:
            return self._dispatch(method, uri, body_bytes, headers)

    def _dispatch(self, method, uri, body_bytes, headers):
        identity = {k.lower(): v for k, v in headers.items()}.get("x-lark-proxy-identity")
        if identity != "user":
            raise ValueError("original user identity required")
        parsed = urlsplit(uri)
        if parsed.scheme or parsed.netloc or parsed.fragment:
            raise ValueError("relative API path required")
        path, query = parsed.path, parse_qs(parsed.query, keep_blank_values=True)
        if method == "GET" and body_bytes:
            raise ValueError("unexpected GET body")
        if method == "GET" and path == ROUTES["identity"][1] and not query:
            return Reply("identity", json={"code": 0, "data": {"open_id": "fixture_self", "name": "Synthetic Fixture", "tenant_key": "fixture-tenant"}})
        if method == "GET" and path == PREFIX + "/fields":
            if set(query) - {"offset", "limit"}:
                raise ValueError("unsupported field query")
            offset, limit = _page(query, 1000)
            return Reply("base_scale_fields", json={"code": 0, "data": {"items": deepcopy(FIELDS[offset:offset+limit]), "has_more": offset+limit < len(FIELDS)}})
        if method == "GET" and path.startswith(PREFIX + "/fields/") and not query:
            field = _field(path[len(PREFIX + "/fields/"):])
            if path != PREFIX + "/fields/" + field["id"]:
                raise ValueError("field detail requires ID")
            return Reply("base_scale_field", json={"code": 0, "data": deepcopy(field)})
        if method == "GET" and path == PREFIX + "/records":
            if set(query) - {"offset", "limit", "field_id"}:
                raise ValueError("unsupported record query")
            offset, limit = _page(query, 2000)
            return self._matrix("base_scale_list", list(self.records)[offset:offset+limit], _projection(query.get("field_id")), offset+limit < len(self.records))
        body = _object(body_bytes)
        if method == "POST" and path == PREFIX + "/records/batch_get" and not query:
            ids = body.get("record_id_list")
            if set(body) - {"record_id_list", "select_fields"} or not isinstance(ids, list) or not 1 <= len(ids) <= 200 or not all(isinstance(v, str) and v for v in ids) or len(set(ids)) != len(ids):
                raise ValueError("invalid record read")
            return self._matrix("base_scale_get", ids, _projection(body.get("select_fields")))
        if method == "POST" and path == PREFIX + "/records/batch_create" and not query:
            if set(body) != {"create_records"}:
                raise ValueError("invalid create body")
            return self._create("base_scale_batch_create", body["create_records"])
        if method == "POST" and path == PREFIX + "/records" and not query:
            return self._create("base_scale_create", [body])
        raise ValueError("unsupported Base scaling route")

    def _matrix(self, route, requested, fields, has_more=False):
        ids = [r for r in requested if r in self.records]
        data = {"timezone": "UTC", "fields": [f["name"] for f in fields], "field_id_list": [f["id"] for f in fields],
                "field_type_list": [f["type"] for f in fields], "record_id_list": ids,
                "data": [[deepcopy(self.records[r].get(f["id"])) for f in fields] for r in ids],
                "record_not_found": [r for r in requested if r not in self.records], "has_more": has_more}
        # Native matrices represent absent cells as null; actual stored key
        # presence remains in the independent snapshot and is never inferred here.
        return Reply(route, json={"code": 0, "data": data}, proof={"record_ids": ids})

    def _create(self, route, rows):
        if not isinstance(rows, list) or not 1 <= len(rows) <= 200 or len(self.records)-1+len(rows) > self.row_count:
            raise ValueError("invalid bounded create count")
        normalized, new_keys = [], set()
        for row in rows:
            if not isinstance(row, dict):
                raise ValueError("native field map required")
            cells = {}
            for name, value in row.items():
                field = _field(name)
                if field["id"] in cells:
                    raise ValueError("duplicate field alias")
                if value is not None:
                    kind = field["type"]
                    if (kind == "text" and not isinstance(value, str)) or (kind == "checkbox" and type(value) is not bool) or (kind == "number" and (type(value) not in (int, float) or not _finite(value))):
                        raise ValueError("invalid native cell type")
                cells[field["id"]] = deepcopy(value)
            key = cells.get("fld_scale_key")
            if not isinstance(key, str) or not key or key in new_keys or key in self.keys:
                raise ValueError("missing or duplicate row key")
            new_keys.add(key)
            normalized.append(cells)
        start = len(self.records)-1
        ids = [f"rec_scale_{start+i:05d}" for i in range(len(rows))]
        self.records.update(zip(ids, normalized))
        self.keys.update(new_keys)
        effect = {"sequence": len(self.effects)+1, "record_ids": ids, "rows": deepcopy(normalized)}
        self.effects.append(effect)
        proof = {"sequence": effect["sequence"], "record_ids": ids, "rows_sha256": _digest(_encode(normalized)), "identity": "user"}
        data = {"record_id_list": ids} if route == "base_scale_batch_create" else {"record_id": ids[0]}
        return Reply(route, json={"code": 0, "data": data}, proof=proof)


def verify(workspace, answer, receipt, expected):
    try:
        n = _size(expected["row_count"])
        root = Path(workspace).resolve()
        if not isinstance(answer, dict) or type(answer.get("imported_count")) is not int or answer["imported_count"] != n or not isinstance(answer.get("report_path"), str):
            return False
        report_path = root / "import-report.json"
        if (root / answer["report_path"]).resolve() != report_path or report_path.is_symlink() or not report_path.is_file():
            return False
        report = _object(report_path.read_bytes())
        if type(report.get("created")) is not int or report["created"] != n or report.get("status", "complete") != "complete":
            return False
        state = receipt["state"]
        if receipt.get("scenario") != SCENARIO or type(state.get("row_count")) is not int or state["row_count"] != n or not _equal(state["initial"], INITIAL) or not _equal(state["fields"], FIELDS):
            return False
        for name, key in (("input.json", "input_sha256"), ("mapping.json", "mapping_sha256")):
            if (root / name).is_symlink() or _digest((root / name).read_bytes()) != expected[key] or state[key] != expected[key]:
                return False
        records = state["records"]
        if not isinstance(records, dict) or len(records) != n+1 or not _equal(records.get(SENTINEL), INITIAL[SENTINEL]):
            return False
        created = {k: v for k, v in records.items() if k != SENTINEL}
        ids = report.get("record_ids")
        if not isinstance(ids, list) or len(ids) != n or not all(isinstance(i, str) and i for i in ids) or len(set(ids)) != n or set(ids) != set(created):
            return False
        wanted = {native_row(i)["fld_scale_key"]: native_row(i) for i in range(n)}
        seen = set()
        for cells in created.values():
            if not isinstance(cells, dict) or not isinstance(cells.get("fld_scale_key"), str):
                return False
            key = cells["fld_scale_key"]
            if key in seen or key not in wanted or not _equal(cells, wanted[key]):
                return False
            seen.add(key)
        requests = receipt["requests"]
        if not isinstance(requests, list) or not requests:
            return False
        mutations = []
        for r in requests:
            if not isinstance(r, dict) or r.get("route") not in ROUTES or r.get("status") != 200 or r.get("signature_valid") is not True:
                return False
            method, uri = ROUTES[r["route"]]
            if r.get("method") != method:
                return False
            if r["route"] == "base_scale_field":
                if r.get("uri") not in [uri+f["id"] for f in FIELDS]:
                    return False
            elif r.get("uri") != uri:
                return False
            if r["route"] in WRITES:
                mutations.append(r)
        effects = state["effects"]
        if not isinstance(effects, list) or not effects or len(effects) != len(mutations):
            return False
        observed = {}
        for index, (effect, request) in enumerate(zip(effects, mutations), 1):
            batch_ids, rows = effect["record_ids"], effect["rows"]
            if type(effect.get("sequence")) is not int or effect["sequence"] != index or not isinstance(batch_ids, list) or not isinstance(rows, list) or not 1 <= len(rows) <= 200 or len(rows) != len(batch_ids):
                return False
            if request["route"] == "base_scale_create" and len(rows) != 1:
                return False
            proof = {"sequence": index, "record_ids": batch_ids, "rows_sha256": _digest(_encode(rows)), "identity": "user"}
            if not _equal(request.get("proof"), proof):
                return False
            for record_id, cells in zip(batch_ids, rows):
                if not isinstance(record_id, str) or record_id in observed or record_id not in created or not _equal(cells, created[record_id]):
                    return False
                observed[record_id] = cells
        if set(observed) != set(created):
            return False
        for key, value in (("requested", n), ("first_unconfirmed_row", n), ("completed_batches", len(effects)), ("base_token", BASE), ("table_id", TABLE)):
            if key in report and not _equal(report[key], value):
                return False
        return "binding" not in report or all(_equal(report["binding"].get(k), v) for k, v in BINDING.items())
    except (ValueError, TypeError, KeyError, OSError, OverflowError, AttributeError):
        return False
