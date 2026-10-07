# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Paired Base update: numeric zero and an explicit native null clear.

Reuses the reviewed development state machine, never its empty-string task.
Native null clearing was observed in the private supported lifecycle. Empty-text
UPDATE storage semantics remain unverified (normalization was observed on CREATE).
"""
from copy import deepcopy
from urllib.parse import parse_qs, urlsplit, urlencode

from scenario_protocol import Reply
from scenarios import base as native

SCENARIO = "paired-base"
BASE, TABLE, TARGET, OTHER = native.BASE, native.TABLE, native.TARGET, native.OTHER
PREFIX, FIELDS, INITIAL = native.PREFIX, deepcopy(native.FIELDS), deepcopy(native.INITIAL)
BASE_PREFIX = f"/open-apis/base/v3/bases/{BASE}"
BASE_META = {"base_token": BASE, "name": "Paired Base fixture"}
TABLE_META = {"id": TABLE, "name": "Typed update"}
VIEW = {"id": "vew_paired_base_grid", "name": "All records", "type": "grid"}
FINAL = deepcopy(INITIAL)
FINAL[TARGET].update(fld_amount=0, fld_text=None)
TASK = {
    "id": SCENARIO, "section": "base",
    "prompt": f"In Base {BASE}, table {TABLE}, update record {TARGET}: set numeric Amount (fld_amount) to 0 and explicitly clear Text (fld_text) with JSON null. Preserve Checked (fld_checked), Attachment (fld_attachment), all other fields, and every other row exactly. Report the updated record ID, resulting amount, and resulting text (null).",
    "fixture": {"scenario": SCENARIO, "scenario_section": SCENARIO,
                "allow_synthetic_mutations": True,
                "answer_schema": {"type": "object", "required": ["record_id", "amount", "text"],
                                  "properties": {"record_id": {"type": "string"},
                                                 "amount": {"type": "number"}, "text": {"type": "null"}},
                                  "additionalProperties": False}},
    "expected": {"answer": {"record_id": TARGET, "amount": 0, "text": None},
                 "initial": deepcopy(INITIAL), "final": deepcopy(FINAL), "fields": deepcopy(FIELDS)},
}
ROUTES = {
    "identity": ("GET", "/open-apis/authen/v1/user_info"),
    "base_record_get": ("POST", PREFIX + "/records/batch_get"),
    "base_record_list": ("GET", PREFIX + "/records"),
    "base_field_list": ("GET", PREFIX + "/fields"),
    "base_field_get": ("GET", PREFIX + "/fields/"),
    "base_record_update": ("PATCH", PREFIX + "/records/" + TARGET),
    "base_record_batch_update": ("POST", PREFIX + "/records/batch_update"),
    "paired_base_get": ("GET", BASE_PREFIX),
    "paired_base_table": ("GET", PREFIX),
    "paired_base_tables": ("GET", BASE_PREFIX + "/tables"),
    "paired_base_views": ("GET", PREFIX + "/views"),
    "paired_base_view": ("GET", PREFIX + "/views/" + VIEW["id"]),
}
METADATA_ROUTES = {r for r in ROUTES if r.startswith("paired_")}


class Scenario(native.Scenario):
    def snapshot(self):
        state = super().snapshot()
        state.update(base=deepcopy(BASE_META), table=deepcopy(TABLE_META), views=[deepcopy(VIEW)])
        return state

    def _dispatch(self, method, uri, body_bytes, headers):
        parsed = urlsplit(uri)
        query = parse_qs(parsed.query, keep_blank_values=True)
        if {k.lower(): v for k, v in headers.items()}.get("x-lark-proxy-identity") != "user":
            raise ValueError("user identity required")
        if parsed.scheme or parsed.netloc or parsed.fragment or (method == "GET" and body_bytes):
            raise ValueError("relative native URI and bodyless GET required")
        # Source: Base table_ops.go and helpers.go listAllTables/Fields/Views.
        if method == "GET" and parsed.path in (BASE_PREFIX, PREFIX, PREFIX + "/views/" + VIEW["id"]):
            if query:
                raise ValueError("metadata detail does not accept selectors")
            route, data = {BASE_PREFIX: ("paired_base_get", BASE_META), PREFIX: ("paired_base_table", TABLE_META),
                           PREFIX + "/views/" + VIEW["id"]: ("paired_base_view", VIEW)}[parsed.path]
            return Reply(route, json={"code": 0, "data": deepcopy(data)})
        lists = {BASE_PREFIX + "/tables": ("paired_base_tables", "tables", [TABLE_META], 100),
                 PREFIX + "/views": ("paired_base_views", "views", [VIEW], 200),
                 PREFIX + "/fields": ("base_field_list", "fields", FIELDS, 1000)}
        if method == "GET" and parsed.path in lists:
            if set(query) - {"offset", "limit"}:
                raise ValueError("unsupported metadata selector")
            route, key, items, maximum = lists[parsed.path]
            offset, limit = native._pagination(query, maximum)
            return Reply(route, json={"code": 0, "data": {key: deepcopy(items[offset:offset + limit]),
                                                          "total": len(items), "has_more": offset + limit < len(items)}})
        if method == "GET" and parsed.path == PREFIX + "/records" and "view_id" in query:
            if query.pop("view_id") != [VIEW["id"]]:
                raise ValueError("foreign view selector")
            uri = parsed.path + "?" + urlencode(query, doseq=True)
        return super()._dispatch(method, uri, body_bytes, headers)

    def _update(self, route, updates):
        if isinstance(updates, dict) and set(updates) == {TARGET}:
            patch = updates[TARGET]
            if isinstance(patch, dict) and any(patch.get(key) == "" for key in ("Text", "fld_text")):
                # Only CREATE empty-text normalization was observed live. Do not
                # invent UPDATE effects or silently substitute requested null.
                raise ValueError("unsupported strategy: native Base empty-text update storage semantics are unverified")
        return super()._update(route, updates)


def verify(workspace, answer, receipt, expected):
    """Require signed effect history and complete independent native state."""
    try:
        if receipt.get("scenario") != SCENARIO:
            return False
        state = receipt["state"]
        for key, wanted in (("base", BASE_META), ("table", TABLE_META), ("views", [VIEW])):
            if not native.typed_equal(state[key], wanted):
                return False
        records = receipt["requests"]
        if not isinstance(records, list):
            return False
        for record in records:
            route = record.get("route")
            if route not in ROUTES or record.get("signature_valid") is not True or record.get("status") != 200:
                return False
            method, path = ROUTES[route]
            allowed_paths = [path + field["id"] for field in FIELDS] if route == "base_field_get" else [path]
            if record.get("method") != method or record.get("uri") not in allowed_paths:
                return False
        adapted = dict(receipt, scenario=native.SCENARIO,
                       requests=[record for record in records if record["route"] not in METADATA_ROUTES])
        return native.verify(workspace, answer, adapted, expected)
    except (KeyError, TypeError, ValueError, AttributeError):
        return False
