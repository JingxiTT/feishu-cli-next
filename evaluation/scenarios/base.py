"""Bounded Base record mutation with independent typed preservation evidence."""
from copy import deepcopy
import json
import math
import threading
from urllib.parse import parse_qs, urlsplit

from scenario_protocol import Reply, ScenarioBase

SCENARIO = "base-zero-empty-preservation-v1"
BASE = "bas_eval_base_dev_01"
TABLE = "tbl_eval_base_dev_01"
TARGET = "rec_eval_target"
OTHER = "rec_eval_other"
PREFIX = f"/open-apis/base/v3/bases/{BASE}/tables/{TABLE}"
FIELDS = [
    {"id": "fld_amount", "name": "Amount", "type": "number"},
    {"id": "fld_text", "name": "Text", "type": "text"},
    {"id": "fld_checked", "name": "Checked", "type": "checkbox"},
    {"id": "fld_attachment", "name": "Attachment", "type": "attachment"},
]
INITIAL = {
    TARGET: {"fld_amount": 8, "fld_text": "old", "fld_checked": True,
             "fld_attachment": [{"file_token": "box_fixture_file", "name": "fixture.txt", "size": 12}]},
    OTHER: {"fld_amount": 21, "fld_text": "unrelated", "fld_checked": False, "fld_attachment": []},
}
FINAL = deepcopy(INITIAL)
FINAL[TARGET].update(fld_amount=0, fld_text="")
TASK = {
    "id": "base-dev-01", "section": "base",
    "prompt": f"In fixture Base {BASE}, table {TABLE}, update record {TARGET}: set numeric Amount (fld_amount) to 0 and Text (fld_text) to an empty string. Preserve Checked (fld_checked), Attachment (fld_attachment), all other fields, and every other row. Report the updated record ID and the resulting amount and text.",
    "fixture": {"scenario": SCENARIO, "allow_synthetic_mutations": True,
                "answer_schema": {"type": "object", "required": ["record_id", "amount", "text"],
                                  "properties": {"record_id": {"type": "string"},
                                                 "amount": {"type": "number"}, "text": {"type": "string"}},
                                  "additionalProperties": True}},
    "expected": {"answer": {"record_id": TARGET, "amount": 0, "text": ""},
                 "initial": deepcopy(INITIAL), "final": deepcopy(FINAL), "fields": deepcopy(FIELDS)},
}
READ_ROUTES = {"identity", "base_record_get", "base_record_list", "base_field_list", "base_field_get"}
WRITE_ROUTES = {"base_record_update", "base_record_batch_update"}


def typed_equal(left, right):
    """JSON booleans are never numbers; exact numeric values may use 0 or 0.0."""
    if isinstance(left, bool) or isinstance(right, bool):
        return type(left) is type(right) and left == right
    if isinstance(left, (int, float)) and isinstance(right, (int, float)):
        return _finite(left) and _finite(right) and left == right
    if type(left) is not type(right):
        return False
    if isinstance(left, dict):
        return left.keys() == right.keys() and all(typed_equal(left[k], right[k]) for k in left)
    if isinstance(left, list):
        return len(left) == len(right) and all(typed_equal(a, b) for a, b in zip(left, right))
    return left == right


def _finite(value):
    try:
        return math.isfinite(value)
    except OverflowError:
        return False


def _object(body):
    def pairs(items):
        result = {}
        for key, value in items:
            if key in result:
                raise ValueError("duplicate JSON key")
            result[key] = value
        return result
    def invalid(_):
        raise ValueError("nonfinite JSON number")
    result = json.loads(body or b"{}", object_pairs_hook=pairs, parse_constant=invalid)
    if not isinstance(result, dict):
        raise ValueError("object required")
    return result


def _field(value):
    for field in FIELDS:
        if value in (field["id"], field["name"]):
            return field["id"]
    raise ValueError("unknown field")


def _projection(values):
    if values is None:
        return [f["id"] for f in FIELDS]
    if not isinstance(values, list) or not values or not all(isinstance(v, str) for v in values):
        raise ValueError("invalid projection")
    result = [_field(value) for value in values]
    if len(set(result)) != len(result):
        raise ValueError("duplicate projection")
    return result


def _pagination(query, maximum):
    values = []
    for key, default in (("offset", "0"), ("limit", str(maximum))):
        raw = query.get(key, [default])
        if len(raw) != 1 or not raw[0].isascii() or not raw[0].isdigit():
            raise ValueError("invalid pagination")
        values.append(int(raw[0]))
    if not 1 <= values[1] <= maximum:
        raise ValueError("invalid limit")
    return values


class Scenario(ScenarioBase):
    def __init__(self):
        self.records = deepcopy(INITIAL)
        self.effects = []
        self.lock = threading.Lock()

    def snapshot(self):
        with self.lock:
            return deepcopy({"initial": INITIAL, "records": self.records, "fields": FIELDS, "effects": self.effects})

    def dispatch(self, method, uri, body_bytes, headers):
        with self.lock:
            return self._dispatch(method, uri, body_bytes, headers)

    def _dispatch(self, method, uri, body_bytes, headers):
        normalized = {key.lower(): value for key, value in headers.items()}
        if normalized.get("x-lark-proxy-identity") != "user":
            raise ValueError("scenario requires original user identity")
        parsed = urlsplit(uri)
        if parsed.scheme or parsed.netloc or parsed.fragment:
            raise ValueError("relative API URI required")
        query = parse_qs(parsed.query, keep_blank_values=True)
        path = parsed.path
        if method == "GET" and body_bytes:
            raise ValueError("GET body unsupported")
        if method == "GET" and path == "/open-apis/authen/v1/user_info" and not query:
            return Reply("identity", json={"code": 0, "data": {"open_id": "fixture_self", "name": "Synthetic Fixture", "tenant_key": "fixture-tenant"}})
        if method == "GET" and path == PREFIX + "/fields":
            if set(query) - {"offset", "limit"}:
                raise ValueError("unsupported field query")
            offset, limit = _pagination(query, 1000)
            return Reply("base_field_list", json={"code": 0, "data": {"items": deepcopy(FIELDS[offset:offset+limit]), "has_more": offset+limit < len(FIELDS)}})
        if method == "GET" and path.startswith(PREFIX + "/fields/") and not query:
            field_id = path[len(PREFIX + "/fields/"):]
            field = next((f for f in FIELDS if f["id"] == field_id), None)
            if field is not None:
                return Reply("base_field_get", json={"code": 0, "data": deepcopy(field)})
        if method == "GET" and path == PREFIX + "/records":
            if set(query) - {"offset", "limit", "field_id"}:
                raise ValueError("unsupported record query")
            offset, limit = _pagination(query, 2000)
            ids = list(self.records)[offset:offset+limit]
            return self._matrix("base_record_list", ids, _projection(query.get("field_id")), offset+limit < len(self.records))
        data = _object(body_bytes)
        if method == "POST" and path == PREFIX + "/records/batch_get" and not query:
            if set(data) - {"record_id_list", "select_fields"}:
                raise ValueError("unsupported record read keys")
            ids = data.get("record_id_list")
            if not isinstance(ids, list) or not 1 <= len(ids) <= 200 or not all(isinstance(v, str) and v for v in ids) or len(set(ids)) != len(ids):
                raise ValueError("invalid record IDs")
            return self._matrix("base_record_get", ids, _projection(data.get("select_fields")))
        if method == "PATCH" and path == PREFIX + "/records/" + TARGET and not query:
            return self._update("base_record_update", {TARGET: data})
        if method == "POST" and path == PREFIX + "/records/batch_update" and not query:
            if set(data) != {"update_records"}:
                raise ValueError("unsupported batch update keys")
            return self._update("base_record_batch_update", data["update_records"])
        raise ValueError("unsupported Base route")

    def _matrix(self, route, requested, projected, has_more=False):
        ids = [record_id for record_id in requested if record_id in self.records]
        fields = [next(f for f in FIELDS if f["id"] == field_id) for field_id in projected]
        data = {"timezone": "UTC", "fields": [f["name"] for f in fields],
                "field_id_list": projected, "field_type_list": [f["type"] for f in fields],
                "record_id_list": ids, "data": [[deepcopy(self.records[r][f]) for f in projected] for r in ids],
                "record_not_found": [r for r in requested if r not in self.records], "has_more": has_more}
        return Reply(route, json={"code": 0, "data": data}, proof={"record_ids": ids, "field_ids": projected})

    def _update(self, route, updates):
        if not isinstance(updates, dict) or set(updates) != {TARGET}:
            raise ValueError("only the requested record may be updated")
        patch = updates[TARGET]
        if not isinstance(patch, dict) or not patch:
            raise ValueError("nonempty field map required")
        normalized = {}
        for key, value in patch.items():
            field_id = _field(key)
            if field_id in normalized:
                raise ValueError("duplicate field alias")
            if field_id == "fld_amount":
                if value is not None and (type(value) not in (int, float) or not _finite(value)):
                    raise ValueError("number or explicit null required")
            elif field_id == "fld_text":
                if value is not None and not isinstance(value, str):
                    raise ValueError("text or explicit null required")
            elif not typed_equal(value, self.records[TARGET][field_id]):
                raise ValueError("preserved field modification forbidden")
            normalized[field_id] = deepcopy(value)
        before = deepcopy(self.records[TARGET])
        self.records[TARGET].update(normalized)
        effect = {"sequence": len(self.effects)+1, "record_id": TARGET, "submitted": normalized,
                  "before": before, "after": deepcopy(self.records[TARGET])}
        self.effects.append(effect)
        data = {"record_id": TARGET} if route == "base_record_update" else {"ignored_fields": []}
        return Reply(route, json={"code": 0, "data": data}, proof=deepcopy(effect))


def verify(workspace, answer, receipt, expected):
    """Trust only the supervisor's in-memory snapshot/ledger, never workspace claims."""
    del workspace
    try:
        wanted = expected["answer"]
        if not isinstance(answer, dict) or any(key not in answer or not typed_equal(answer[key], wanted[key]) for key in wanted):
            return False
        if receipt.get("scenario") != SCENARIO:
            return False
        state = receipt["state"]
        if not typed_equal(state["initial"], expected["initial"]) or not typed_equal(state["fields"], expected["fields"]) or not typed_equal(state["records"], expected["final"]):
            return False
        records = receipt["requests"]
        if not isinstance(records, list) or not records or any(not isinstance(r, dict) or r.get("status") != 200 or r.get("route") not in READ_ROUTES | WRITE_ROUTES for r in records):
            return False
        mutations = [r for r in records if r["route"] in WRITE_ROUTES]
        effects = state["effects"]
        if not isinstance(effects, list) or not effects or len(effects) != len(mutations):
            return False
        previous = deepcopy(expected["initial"][TARGET])
        for sequence, (effect, request) in enumerate(zip(effects, mutations), 1):
            if effect.get("sequence") != sequence or effect.get("record_id") != TARGET or not typed_equal(effect, request.get("proof")) or not typed_equal(effect["before"], previous):
                return False
            submitted = effect["submitted"]
            if not isinstance(submitted, dict) or not submitted or set(submitted) - set(previous):
                return False
            after = deepcopy(previous)
            for key, value in submitted.items():
                if not typed_equal(value, previous[key]) and not typed_equal(value, expected["final"][TARGET][key]):
                    return False
                after[key] = value
            if not typed_equal(after, effect["after"]):
                return False
            previous = after
        return typed_equal(previous, expected["final"][TARGET])
    except (KeyError, TypeError, ValueError, OverflowError):
        return False
