#!/usr/bin/env python3
"""Small, non-forwarding synthetic sidecar; signed reads and an independent ledger."""
import hashlib
import hmac
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
from pathlib import Path
import threading
import time
from scenario_protocol import Reply
from read_fixture import native_read_label
from urllib.parse import parse_qs, urlsplit

PROXY_KEY = "synthetic-evaluation-proxy-key"
UNSUPPORTED_REASONS = {
    "unsupported strategy: historical native Docs revision is outside this fixture": "unsupported_docs_historical_revision",
    "unsupported strategy: native scoped Markdown-document fetch unobserved": "unsupported_markdown_fetch_scope",
    "unsupported strategy: native Sheets structure inspection unobserved": "unsupported_sheets_structure",
    "unsupported strategy: native Sheets CSV inspection unobserved": "unsupported_sheets_csv",
    "unsupported strategy: general reconstruction needs validated native identity semantics": "unsupported_reconstruction",
    "unsupported strategy: XML replacement fragments require native inline parsing": "unsupported_xml_fragment",
}
UNSUPPORTED_REASON_CODES = frozenset((*UNSUPPORTED_REASONS.values(), "unsupported_strategy_other"))
CONTACTS = [
    {"open_id": "ou_fixture_1", "name": "Alex", "email": "alex.one@example.invalid", "enterprise_email": "alex.one@example.invalid"},
    {"open_id": "ou_fixture_2", "name": "Alex", "email": "alex.two@example.invalid", "enterprise_email": "alex.two@example.invalid"},
]


def signed_headers(method, uri, body=b"", identity="user", timestamp=None):
    """Used by controlled tests; production CLI constructs its own signature."""
    stamp = str(int(time.time()) if timestamp is None else timestamp)
    digest = hashlib.sha256(body).hexdigest()
    fields = ["v1", method, "open.feishu.cn", uri, digest, stamp, identity, "Authorization"]
    return {"X-Lark-Proxy-Version": "v1", "X-Lark-Proxy-Target": "https://open.feishu.cn",
            "X-Lark-Proxy-Identity": identity, "X-Lark-Proxy-Auth-Header": "Authorization",
            "X-Lark-Proxy-Timestamp": stamp, "X-Lark-Body-SHA256": digest,
            "X-Lark-Proxy-Signature": hmac.new(PROXY_KEY.encode(), "\n".join(fields).encode(), hashlib.sha256).hexdigest()}


def valid_signature(method, uri, body, headers):
    target = headers.get("X-Lark-Proxy-Target", "")
    if target not in ("https://open.feishu.cn", "https://open.larksuite.com") or headers.get("Authorization"):
        return False
    if headers.get("X-Lark-Proxy-Version") != "v1" or headers.get("X-Lark-Proxy-Identity") not in ("user", "bot") or headers.get("X-Lark-Proxy-Auth-Header") != "Authorization":
        return False
    try:
        if abs(int(headers["X-Lark-Proxy-Timestamp"]) - time.time()) > 60:
            return False
        digest = hashlib.sha256(body).hexdigest()
        if headers.get("X-Lark-Body-SHA256") != digest:
            return False
        fields = ["v1", method, urlsplit(target).netloc, uri, digest,
                  headers["X-Lark-Proxy-Timestamp"], headers["X-Lark-Proxy-Identity"], "Authorization"]
        expected = hmac.new(PROXY_KEY.encode(), "\n".join(fields).encode(), hashlib.sha256).hexdigest()
        return hmac.compare_digest(expected, headers.get("X-Lark-Proxy-Signature", ""))
    except (ValueError, KeyError, TypeError):
        return False


def contact_read(method, uri, body):
    """Accept equivalent contact reads; every other route is rejected, never forwarded."""
    parsed = urlsplit(uri)
    query = parse_qs(parsed.query)
    path = parsed.path
    if method == "GET" and path == "/open-apis/authen/v1/user_info":
        return "identity", {"open_id": "fixture_self", "name": "Synthetic Fixture", "tenant_key": "fixture-tenant"}, []
    data = json.loads(body or "{}")
    if not isinstance(data, dict):
        raise ValueError("object body required")
    if query.get("user_id_type", ["open_id"]) != ["open_id"]:
        raise ValueError("scenario uses open_id")
    if method == "POST" and path == "/open-apis/contact/v3/users/basic_batch":
        ids = data.get("user_ids")
        if not isinstance(ids, list) or not all(isinstance(value, str) for value in ids):
            raise ValueError("invalid IDs")
        users = [dict(user) for user in CONTACTS if user["open_id"] in ids]
        return "contact_batch", {"users": users}, [user["open_id"] for user in users]
    if method == "GET" and path.startswith("/open-apis/contact/v3/users/"):
        users = [dict(user) for user in CONTACTS if user["open_id"] == path.rsplit("/", 1)[-1]]
        if users:
            return "contact_get", {"user": users[0]}, [users[0]["open_id"]]
    if method == "POST" and path == "/open-apis/contact/v3/users/search":
        criteria = data.get("filter") or {}
        if not isinstance(criteria, dict):
            raise ValueError("invalid filter")
        ids = criteria.get("user_ids")
        if ids is not None and (not isinstance(ids, list) or not all(isinstance(value, str) for value in ids)):
            raise ValueError("invalid IDs")
        needle = data.get("query", "")
        if not isinstance(needle, str) or (not needle and ids is None):
            raise ValueError("search criterion required")
        users = [user for user in CONTACTS if (ids is None or user["open_id"] in ids)
                 and (not needle or needle.casefold() in user["name"].casefold() or needle.casefold() in user["email"].casefold())]
        items = [{"id": user["open_id"], "display_info": "<h>Alex</h>\nSynthetic Department",
                  "meta_data": {"i18n_names": {"en_us": user["name"]}, "mail_address": user["email"],
                                "enterprise_mail_address": user["email"], "is_registered": True,
                                "chat_id": "", "is_cross_tenant": False}} for user in users]
        return "contact_search", {"items": items, "has_more": False, "page_token": ""}, [user["open_id"] for user in users]
    raise ValueError("unsupported route")


class SectionFixture:
    def __init__(self, scenario, dispatch=None, snapshot=None):
        if scenario != "contact-exact-id-v1" and dispatch is None:
            raise ValueError("unsupported fixture scenario")
        self.scenario = scenario
        self.dispatch = dispatch
        self.snapshot = snapshot
        self.records = []
        self.lock = threading.Lock()
        fixture = self

        class Handler(BaseHTTPRequestHandler):
            def setup(self):
                self.request.settimeout(2)
                super().setup()

            def log_message(self, *_):
                pass

            def handle_request(self):
                status, route, ids, proof = 403, "rejected", [], {}
                signature_valid, rejection, reason = False, None, None
                content_type, extra_headers, encoded = "application/json", {}, None
                with fixture.lock:
                    try:
                        length = int(self.headers.get("Content-Length", "0"))
                        if length < 0 or length > 65536 or self.headers.get("Transfer-Encoding"):
                            raise ValueError("invalid length")
                        body = self.rfile.read(length)
                        signature_valid = valid_signature(self.command, self.path, body, self.headers)
                        if not signature_valid:
                            raise ValueError("invalid signature")
                        if fixture.dispatch is None:
                            route, data, ids = contact_read(self.command, self.path, body)
                            response = {"code": 0, "msg": "success", "data": data}
                            status = 200
                        else:
                            result = fixture.dispatch(self.command, self.path, body, self.headers)
                            if isinstance(result, Reply):
                                route, response, status, proof = result.route, result.json, result.status, result.proof
                                encoded, content_type, extra_headers = result.body, result.content_type, result.headers
                            else:
                                route, response, ids = result
                                status = 200
                    except (ValueError, KeyError, TypeError) as error:
                        rejection = "unsupported_strategy" if str(error).startswith("unsupported strategy:") else "rejected_request"
                        if rejection == "unsupported_strategy":
                            reason = UNSUPPORTED_REASONS.get(str(error), "unsupported_strategy_other")
                        status, route = 403, "rejected"
                        response = {"code": 403, "msg": "synthetic fixture rejected request"}
                    except Exception:
                        status, route = 500, "fixture_error"
                        response = {"code": 500, "msg": "synthetic fixture implementation error"}
                    record = {"route": route, "status": status, "returned_ids": ids,
                              "method": self.command, "uri": urlsplit(self.path).path,
                              "signature_valid": signature_valid}
                    native_route = native_read_label(self.command, self.path)
                    if native_route:
                        record["native_read"] = native_route
                    if rejection:
                        record["rejection"] = rejection
                    if reason:
                        record["unsupported_reason"] = reason
                    if proof:
                        record["proof"] = proof
                    fixture.records.append(record)
                if encoded is None:
                    encoded = json.dumps(response, allow_nan=False).encode()
                self.send_response(status)
                self.send_header("Content-Type", content_type)
                self.send_header("Content-Length", str(len(encoded)))
                for name, value in extra_headers.items():
                    if name.lower() not in ("content-type", "content-length"):
                        self.send_header(name, str(value))
                self.end_headers()
                self.wfile.write(encoded)

            do_GET = do_POST = do_PUT = do_PATCH = do_DELETE = do_HEAD = do_OPTIONS = handle_request

        self.server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        # Drain bounded in-flight handlers before taking a final state snapshot.
        self.server.daemon_threads = False
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)

    def __enter__(self):
        self.thread.start()
        return self

    def __exit__(self, *_):
        self.server.shutdown()
        self.server.server_close()
        self.thread.join()

    def environment(self):
        return {"LARKSUITE_CLI_AUTH_PROXY": "http://127.0.0.1:" + str(self.server.server_port),
                "LARKSUITE_CLI_PROXY_KEY": PROXY_KEY, "LARKSUITE_CLI_APP_ID": "synthetic-evaluation-app",
                "LARKSUITE_CLI_BRAND": "feishu", "LARKSUITE_CLI_DEFAULT_AS": "user"}

    def receipt(self):
        with self.lock:
            result = {"scenario": self.scenario, "requests": list(self.records)}
            if self.snapshot is not None:
                result["state"] = self.snapshot()
            return result

    def write_receipt(self, workspace, extra=None):
        # Called only after the agent process has ended. Never trust an agent-written ledger.
        path = Path(workspace) / "backend-receipt.json"
        path.unlink(missing_ok=True)
        path.write_text(json.dumps({**self.receipt(), **(extra or {})}), encoding="utf-8")
