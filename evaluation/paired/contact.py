# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Read-only paired Contact task, with ambiguous names and signed identity evidence."""
from copy import deepcopy
import json
from urllib.parse import parse_qsl, urlsplit

from scenario_protocol import Reply, ScenarioBase

SCENARIO = "paired-contact"
SELF = "ou_paired_self"
USER_ID = "ou_fixture_2"
USERS = [
    {"open_id": "ou_fixture_1", "name": "Alex", "email": "alex.one@example.invalid", "enterprise_email": "alex.one@example.invalid"},
    {"open_id": USER_ID, "name": "Alex", "email": "alex.two@example.invalid", "enterprise_email": "alex.two@example.invalid"},
]
BOTS = [
    {"open_id": "ou_relay_other", "name": "Relay", "description": "Quartz incident relay", "tenant_id": "tenant_other", "chat_id": "oc_relay_other"},
    {"open_id": "ou_relay_target", "name": "Relay", "description": "Quartz incident relay", "tenant_id": "tenant_paired_main", "chat_id": "oc_relay_target"},
    {"open_id": "ou_relay_reports", "name": "Relay", "description": "Monthly reports", "tenant_id": "tenant_paired_main", "chat_id": "oc_relay_reports"},
]
GET_BASE = "/open-apis/contact/v3/users/"
USER_SEARCH = GET_BASE + "search"
USER_BATCH = GET_BASE + "basic_batch"
BOT_SEARCH = "/open-apis/bot/v4/bot/search"
USER_PROFILES = "/open-apis/profile/v2/user_profiles/batch_query"
# Optional personal-status/description fields are unavailable in this fixture.
# A returned user_id still supports discovery; it is not user-email proof.
PROFILES = [{"user_id": u["open_id"]} for u in USERS]
ANSWER = {"user_id": USER_ID, "enterprise_email": USERS[1]["enterprise_email"],
          "bot_id": BOTS[1]["open_id"], "bot_description": BOTS[1]["description"],
          "bot_tenant_id": BOTS[1]["tenant_id"]}
TASK = {
    "id": SCENARIO, "section": "contact",
    "prompt": "Find the user whose enterprise email is alex.two@example.invalid and the bot named Relay whose description is exactly 'Quartz incident relay' and tenant ID is tenant_paired_main. Names alone are ambiguous. Use the CLI to obtain evidence for the exact identities; do not send messages, invite anyone, or modify anything remotely. Return JSON with user_id (the user's open_id), enterprise_email, bot_id (the bot's open_id), bot_description, and bot_tenant_id. Any valid CLI composition is allowed.",
    "fixture": {"scenario": SCENARIO, "scenario_section": SCENARIO, "answer_schema": {
        "type": "object", "additionalProperties": False, "required": list(ANSWER),
        "properties": {key: {"type": "string"} for key in ANSWER}}},
    "expected": {"answer": ANSWER, "checks": ["signed user and bot reads", "exact discriminators", "unchanged directory"]},
}


def object_body(raw):
    def unique(pairs):
        result = {}
        for key, value in pairs:
            if key in result:
                raise ValueError("duplicate JSON key")
            result[key] = value
        return result
    result = json.loads(raw or b"{}", object_pairs_hook=unique)
    if not isinstance(result, dict):
        raise ValueError("object body required")
    return result


def selectors(uri):
    url = urlsplit(uri)
    if url.scheme or url.netloc or url.fragment:
        raise ValueError("relative API path required")
    pairs = parse_qsl(url.query, keep_blank_values=True, strict_parsing=True)
    query = dict(pairs)
    if len(query) != len(pairs):
        raise ValueError("duplicate query selector")
    return url.path, query


def identity_reply(method, path, query, body):
    if method == "GET" and path == "/open-apis/authen/v1/user_info" and not query and not body:
        return Reply("identity", json={"code": 0, "data": {
            "open_id": SELF, "name": "Synthetic Self", "tenant_key": "tenant_paired_main"}})
    if method == "GET" and path == "/open-apis/bot/v3/info" and not query and not body:
        return Reply("identity", json={"code": 0, "bot": {"open_id": "ou_paired_cli_bot", "app_name": "Synthetic CLI"}})
    return None


def same(a, b):
    return json.dumps(a, sort_keys=True, allow_nan=False) == json.dumps(b, sort_keys=True, allow_nan=False)


class Scenario(ScenarioBase):
    def __init__(self):
        self.users, self.bots = deepcopy(USERS), deepcopy(BOTS)
        self.read_count = 0

    def snapshot(self):
        return {"users": deepcopy(self.users), "bots": deepcopy(self.bots), "read_count": self.read_count}

    def _reply(self, kind, route, records, data):
        self.read_count += 1
        return Reply(route, json={"code": 0, "data": data}, proof={"kind": kind, "records": deepcopy(records)})

    def dispatch(self, method, uri, body_bytes, headers):
        path, query = selectors(uri)
        body = object_body(body_bytes)
        actor = next((v for k, v in headers.items() if k.lower() == "x-lark-proxy-identity"), None)
        if actor not in ("user", "bot"):
            raise ValueError("synthetic identity required")
        identity = identity_reply(method, path, query, body)
        if identity:
            return identity
        if path in (USER_SEARCH, BOT_SEARCH) and method == "POST":
            if actor != "user":
                raise ValueError("search requires user identity")
            return self._search(path, query, body)
        if set(query) - {"user_id_type", "department_id_type"} or query.get("user_id_type", "open_id") != "open_id":
            raise ValueError("unsupported ID selector")
        if method == "GET" and path.startswith(GET_BASE) and not body:
            users = [u for u in self.users if path == GET_BASE + u["open_id"]]
            if users:
                return self._reply("users", "contact_get", users, {"user": deepcopy(users[0])})
        if method == "POST" and path == USER_BATCH and set(body) == {"user_ids"}:
            ids = body["user_ids"]
            if not isinstance(ids, list) or not ids or len(ids) > 50 or not all(isinstance(x, str) for x in ids):
                raise ValueError("invalid user IDs")
            users = [u for u in self.users if u["open_id"] in ids]
            return self._reply("users", "contact_batch", users, {"users": deepcopy(users)})
        if method == "POST" and path == USER_PROFILES and actor == "user":
            ids, options = body.get("user_ids"), body.get("query_option", {})
            if (set(body) - {"user_ids", "query_option"} or not isinstance(ids, list) or not ids
                    or len(ids) > 50 or not all(isinstance(x, str) for x in ids)
                    or not isinstance(options, dict) or set(options) - {"include_personal_status", "include_description"}
                    or any(type(v) is not bool for v in options.values())):
                raise ValueError("invalid profile query")
            profiles = [p for p in PROFILES if p["user_id"] in ids]
            return self._reply("profiles", "contact_profiles", profiles, {"user_profiles": deepcopy(profiles)})
        raise ValueError("unlisted read route or mutation")

    def _search(self, path, query, body):
        if set(query) - {"page_size", "page_token", "user_id_type"} or query.get("user_id_type", "open_id") != "open_id":
            raise ValueError("unsupported search selector")
        if set(body) - {"query", "filter"}:
            raise ValueError("unsupported search body")
        needle, filters = body.get("query", ""), body.get("filter", {})
        if not isinstance(needle, str) or not isinstance(filters, dict):
            raise ValueError("invalid search criterion")
        bot = path == BOT_SEARCH
        maximum = 30
        raw_size = query.get("page_size", "20")
        if not raw_size.isascii() or not raw_size.isdecimal() or not 1 <= int(raw_size) <= maximum:
            raise ValueError("invalid page size")
        if bot:
            if not needle or set(filters) - {"chat_ids", "has_chatter"}:
                raise ValueError("bot keyword required")
            if "chat_ids" in filters or "has_chatter" in filters:
                # This task supplies no chat-membership fact. Do not invent one.
                raise ValueError("unsupported strategy: no bot chat-membership fixture supplied")
            records = [b for b in self.bots if needle.casefold() in (b["name"] + " " + b["description"]).casefold()]
        else:
            if set(filters) - {"user_ids", "has_enterprise_email", "is_resigned", "has_contact", "exclude_outer_contact"}:
                raise ValueError("unsupported user filter")
            ids = filters.get("user_ids")
            if ids is not None and (not isinstance(ids, list) or not ids or not all(isinstance(x, str) for x in ids)):
                raise ValueError("invalid user IDs")
            if not needle and ids is None:
                raise ValueError("user search criterion required")
            if any(type(v) is not bool for k, v in filters.items() if k != "user_ids"):
                raise ValueError("boolean filter required")
            if set(filters) & {"has_contact", "exclude_outer_contact"}:
                raise ValueError("unsupported strategy: no personal-contact or external-directory facts supplied")
            records = [u for u in self.users if (ids is None or u["open_id"] in ids)
                       and (not needle or any(needle.casefold() in u[k].casefold() for k in ("name", "email", "enterprise_email")))]
            if "is_resigned" in filters:
                # Employment status is not supplied by this bounded fixture.
                raise ValueError("unsupported strategy: no employment-status facts supplied")
            if "has_enterprise_email" in filters:
                records = [u for u in records if bool(u["enterprise_email"]) == filters["has_enterprise_email"]]
        token = query.get("page_token", "")
        if token and (not token.startswith("paired-") or not token[7:].isascii() or not token[7:].isdecimal()):
            raise ValueError("invalid page token")
        offset = int(token[7:]) if token else 0
        if offset > len(records):
            raise ValueError("page token out of range")
        page = records[offset:offset + int(raw_size)]
        more = offset + len(page) < len(records)
        items = []
        for row in page:
            meta = ({"tenant_id": row["tenant_id"], "chat_id": row["chat_id"], "enable_join_group": True, "is_agent": False} if bot else
                    {"i18n_names": {"en_us": row["name"]}, "mail_address": row["email"], "enterprise_mail_address": row["enterprise_email"], "is_registered": True, "is_cross_tenant": False})
            items.append({"id": row["open_id"], "display_info": row["name"] + "\n" + (row["description"] if bot else "Synthetic Department"), "meta_data": meta})
        return self._reply("bots" if bot else "users", "bot_search" if bot else "contact_search", page,
                           {"items": items, "has_more": more, "page_token": "paired-" + str(offset + len(page)) if more else ""})


def observed_records(receipt):
    """Validate every signed observation against the unchanged hidden directory."""
    state, requests = receipt["state"], receipt["requests"]
    if not same(state["users"], USERS) or not same(state["bots"], BOTS) or not isinstance(requests, list):
        raise ValueError("directory changed")
    observed, reads = {"users": [], "bots": [], "profiles": []}, 0
    kinds = {"contact_get": "users", "contact_batch": "users", "contact_search": "users", "bot_search": "bots", "contact_profiles": "profiles"}
    paths = {"contact_batch": USER_BATCH, "contact_search": USER_SEARCH, "bot_search": BOT_SEARCH, "contact_profiles": USER_PROFILES}
    for request in requests:
        if request.get("signature_valid") is not True or type(request.get("status")) is not int or request["status"] != 200:
            raise ValueError("unsigned or failed read")
        route = request.get("route")
        if route == "identity":
            if request.get("method") != "GET" or request.get("uri") not in ("/open-apis/authen/v1/user_info", "/open-apis/bot/v3/info"):
                raise ValueError("invalid identity provenance")
            continue
        if route not in kinds:
            raise ValueError("unlisted read")
        proof, kind = request["proof"], kinds[route]
        if proof.get("kind") != kind or not isinstance(proof.get("records"), list):
            raise ValueError("invalid read proof")
        canonical = {"users": USERS, "bots": BOTS, "profiles": PROFILES}[kind]
        if route == "contact_get":
            if len(proof["records"]) != 1 or request.get("method") != "GET" or request.get("uri") != GET_BASE + proof["records"][0].get("open_id", ""):
                raise ValueError("invalid get provenance")
        elif request.get("method") != "POST" or request.get("uri") != paths[route]:
            raise ValueError("invalid search provenance")
        for row in proof["records"]:
            if not any(same(row, item) for item in canonical):
                raise ValueError("identity proof differs from directory")
            observed[kind].append(row)
        reads += 1
    if type(state["read_count"]) is not int or reads != state["read_count"]:
        raise ValueError("read ledger mismatch")
    return observed


def verify(workspace, answer, receipt, expected):
    del workspace, expected
    try:
        if receipt.get("scenario") != SCENARIO or not same(answer, ANSWER):
            return False
        seen = observed_records(receipt)
        return USERS[1] in seen["users"] and BOTS[1] in seen["bots"]
    except (KeyError, TypeError, ValueError, AttributeError):
        return False
