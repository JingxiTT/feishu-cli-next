#!/usr/bin/env python3
"""Small, non-forwarding synthetic sidecar; signed reads and an independent ledger."""
import hashlib
import hmac
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
from pathlib import Path
import threading
import time
from urllib.parse import parse_qs, urlsplit

PROXY_KEY = "synthetic-evaluation-proxy-key"
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
    def __init__(self, scenario):
        if scenario != "contact-exact-id-v1":
            raise ValueError("unsupported fixture scenario")
        self.records = []
        self.lock = threading.Lock()
        fixture = self

        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *_):
                pass

            def handle_request(self):
                status, route, ids = 403, "rejected", []
                try:
                    length = int(self.headers.get("Content-Length", "0"))
                    if length < 0 or length > 65536 or self.headers.get("Transfer-Encoding"):
                        raise ValueError("invalid length")
                    body = self.rfile.read(length)
                    if not valid_signature(self.command, self.path, body, self.headers):
                        raise ValueError("invalid signature")
                    route, data, ids = contact_read(self.command, self.path, body)
                    status = 200
                    response = {"code": 0, "msg": "success", "data": data}
                except (ValueError, KeyError, TypeError):
                    response = {"code": 403, "msg": "synthetic fixture rejected request"}
                with fixture.lock:
                    fixture.records.append({"route": route, "status": status, "returned_ids": ids})
                encoded = json.dumps(response).encode()
                self.send_response(status)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(encoded)))
                self.end_headers()
                self.wfile.write(encoded)

            do_GET = do_POST = do_PUT = do_PATCH = do_DELETE = do_HEAD = do_OPTIONS = handle_request

        self.server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
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
            return {"scenario": "contact-exact-id-v1", "requests": list(self.records)}

    def write_receipt(self, workspace):
        # Called only after the agent process has ended. Never trust an agent-written ledger.
        path = Path(workspace) / "backend-receipt.json"
        path.unlink(missing_ok=True)
        path.write_text(json.dumps(self.receipt()), encoding="utf-8")
