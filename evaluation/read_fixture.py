"""Execute bounded read-only response tables reviewed by each section owner."""
import json
import re
from pathlib import Path
from urllib.parse import parse_qs, urlsplit

RUNTIME = Path(__file__).resolve().parent / "runtime"


def typed_equal(left, right):
    return json.dumps(left, sort_keys=True, ensure_ascii=False) == json.dumps(right, sort_keys=True, ensure_ascii=False)


def subset(value, expected):
    if isinstance(expected, dict):
        return isinstance(value, dict) and all(key in value and subset(value[key], wanted) for key, wanted in expected.items())
    return typed_equal(value, expected)


def matches(request, query, body, identity):
    if "identities" in request and identity not in request["identities"]:
        return False
    if "query_keys" in request and set(query) - set(request["query_keys"]):
        return False
    if set(query) & set(request.get("forbidden_query_keys", [])) or set(body) & set(request.get("forbidden_body_keys", [])):
        return False
    for optional in (False, True):
        for key, allowed in request.get("optional_query" if optional else "query", {}).items():
            if optional and key not in query:
                continue
            if key not in query or len(query[key]) != 1 or query[key][0] not in (allowed if isinstance(allowed, list) else [allowed]):
                return False
    for key, allowed in request.get("optional_body", {}).items():
        if key in body and not any(typed_equal(body[key], item) for item in allowed):
            return False
    return subset(body, request.get("body_contains", {}))


def load_runtime(section):
    if not isinstance(section, str) or not section.replace("-", "").isalnum():
        raise ValueError("invalid runtime section")
    data = json.loads((RUNTIME / (section + ".json")).read_text(encoding="utf-8"))
    if data["section"] != section or data["schema_version"] != 1:
        raise ValueError("invalid runtime data")
    for route in data["routes"]:
        if route["method"] not in ("GET", "POST") or route["response"]["http_status"] != 200 or "json" not in route["response"]:
            raise ValueError("only reviewed successful JSON read routes supported")
    return data


def dispatcher(runtime):
    def dispatch(method, uri, raw_body, headers):
        parsed = urlsplit(uri)
        query = parse_qs(parsed.query, keep_blank_values=True)
        body = json.loads(raw_body or "{}")
        if not isinstance(body, dict):
            raise ValueError("object body required")
        for route in runtime["routes"]:
            if route["method"] == method and route["path"] == parsed.path and matches(route.get("request", {}), query, body, headers.get("X-Lark-Proxy-Identity")):
                return route["id"], route["response"]["json"], []
        raise ValueError("unmatched synthetic read")
    return dispatch


def verify(request):
    runtime = load_runtime(request["fixture"]["runtime_section"])
    workspace = Path(request["workspace"])
    expected = request["expected"]
    answer = json.loads((workspace / "answer.json").read_text(encoding="utf-8"))
    desired = json.loads(json.dumps(expected["answer"]))
    for key in expected.get("unordered_answer_paths", []):
        if not isinstance(answer.get(key), list):
            return False
        answer[key] = sorted(answer[key], key=lambda item: json.dumps(item, sort_keys=True))
        desired[key] = sorted(desired[key], key=lambda item: json.dumps(item, sort_keys=True))
    if not typed_equal(answer, desired):
        return False
    receipt = json.loads((workspace / "backend-receipt.json").read_text(encoding="utf-8"))
    if receipt.get("scenario") != runtime["task"]["fixture"]["scenario"]:
        return False
    records = receipt.get("requests")
    if not isinstance(records, list) or not records:
        return False
    allowed = {route["id"] for route in runtime["routes"]}
    if any(record.get("status") != 200 or record.get("route") not in allowed for record in records):
        return False
    seen = {record["route"] for record in records}
    if not all(seen.intersection(group) for group in expected["required_reads"]):
        return False
    # Read-oriented raw POSTs can create conservative local attempt journals.
    # These are not auth/profile changes or remote mutations; route receipts decide
    # remote behavior. Keep every other configuration write forbidden.
    config = workspace / "feishu-config"
    return all(re.fullmatch(r"next/receipts/[0-9a-f]{32}\.json(?:\.lock)?", path.relative_to(config).as_posix())
               for path in config.rglob("*") if path.is_file())
