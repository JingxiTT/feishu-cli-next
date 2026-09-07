"""Bounded synthetic binary download; no live or memory-scalability claim."""
import base64
import hashlib
import json
from pathlib import Path
import re
import threading
from urllib.parse import parse_qsl, urlsplit

from scenario_protocol import Reply, ScenarioBase

SCENARIO = "drive-binary-download-v1"
TOKEN = "fileDriveDev01"
TITLE = "drive-dev-01.bin"
ARTIFACT = "downloads/" + TITLE
SIZE = 1048576
PAYLOAD = bytes(range(251)) * (SIZE // 251) + bytes(range(SIZE % 251))
SHA256 = "631b84027d6b9e52b539c4e8373622d23032dfadc64d60af87339c9037e4f769"
DOWNLOAD = "/open-apis/drive/v1/files/" + TOKEN + "/download"
AUTH = "/open-apis/drive/v1/permissions/" + TOKEN + "/members/auth"
META = "/open-apis/drive/v1/metas/batch_query"
SENTINEL = ".drive-preserve.txt"
SENTINEL_BYTES = "Unrelated workspace content Ω\n".encode()
TASK = {
    "id": "drive-dev-01", "section": "drive",
    "prompt": "Download the synthetic Drive file fileDriveDev01 (https://example.feishu.cn/file/fileDriveDev01) into this workspace as downloads/drive-dev-01.bin. Report its workspace-relative path and SHA-256 checksum. Preserve the remote source and do not print, encode, or include the binary contents in your answer or tool output.",
    "fixture": {"scenario": SCENARIO, "answer_schema": {
        "type": "object", "additionalProperties": False, "required": ["path", "sha256"],
        "properties": {"path": {"type": "string"}, "sha256": {"type": "string"}}}},
    "expected": {"checks": ["download observed", "actual bytes and checksum exact", "source and unrelated file preserved", "no binary dump"]},
}


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def object_body(raw):
    def unique(pairs):
        value = {}
        for key, item in pairs:
            if key in value:
                raise ValueError("duplicate JSON key")
            value[key] = item
        return value
    body = json.loads(raw or b"{}", object_pairs_hook=unique)
    if not isinstance(body, dict):
        raise ValueError("object body required")
    return body


def source_state(payload=PAYLOAD):
    return {"token": TOKEN, "type": "file", "title": TITLE, "bytes": len(payload),
            "sha256": digest(payload), "acl": "self_only"}


def contains_dump(text):
    """Detect substantial literal/common-encoded payload runs, not arbitrary encodings.

    This qualitative check is separate from deterministic artifact verification.
    JSON protocol escaping is decoded by audit_output before checking strings.
    Short checksum/path output is allowed; this is not a steganography detector.
    """
    if not isinstance(text, str):
        return True
    compact = re.sub(r"\s+", "", text)
    for offset in range(251):
        chunk = (bytes(range(251)) * 2)[offset:offset + 96]
        candidates = (chunk.decode("latin1"), chunk.decode("utf-8", "replace"),
                      chunk.hex(), base64.b64encode(chunk).decode(),
                      repr(chunk)[2:-1], ",".join(map(str, chunk)), ", ".join(map(str, chunk)))
        if (any(candidate in text for candidate in candidates)
                or chunk.hex() in compact or base64.b64encode(chunk).decode() in compact):
            return True
    return False


def audit_output(raw_jsonl):
    # Inspect decoded string values, never syntax/escaping in raw provider JSON.
    def dumped(value):
        if isinstance(value, str):
            return contains_dump(value)
        if isinstance(value, list):
            if len(value) >= 96 and all(type(x) is int for x in value):
                return any(value[i:i + 96] == [(value[i] + n) % 251 for n in range(96)]
                           for i in range(len(value) - 95))
            return any(dumped(item) for item in value)
        if isinstance(value, dict):
            # Commands may legitimately describe generators or patterns; audit the
            # actual answer and tool output instead of invocation source code.
            return any(dumped(item) for key, item in value.items()
                       if key not in {"command", "arguments", "input", "code"})
        return False
    if not isinstance(raw_jsonl, str):
        return False
    for line in raw_jsonl.splitlines():
        try:
            value = json.loads(line)
        except (ValueError, TypeError):
            value = line
        if dumped(value):
            return False
    return True


class Scenario(ScenarioBase):
    audit_output = staticmethod(audit_output)

    def __init__(self):
        self._lock = threading.Lock()
        self.payload = PAYLOAD
        self.downloads = 0
        self.metadata = source_state()

    def setup(self, workspace):
        self.workspace = Path(workspace)
        if (self.workspace / ARTIFACT).exists():
            raise ValueError("download destination must start absent")
        with (self.workspace / SENTINEL).open("xb") as stream:
            stream.write(SENTINEL_BYTES)

    def dispatch(self, method, uri, body_bytes, headers):
        with self._lock:
            url = urlsplit(uri)
            if url.scheme or url.netloc or url.fragment:
                raise ValueError("relative API URI required")
            pairs = parse_qsl(url.query, keep_blank_values=True)
            query = dict(pairs)
            if len(query) != len(pairs):
                raise ValueError("duplicate query selector")
            identity = next((v for k, v in headers.items() if k.lower() == "x-lark-proxy-identity"), None)
            if identity not in {"user", "bot"}:
                raise ValueError("synthetic user or bot required")
            body = object_body(body_bytes)
            if method == "GET" and not body:
                if url.path == "/open-apis/authen/v1/user_info" and not query:
                    return Reply("identity", json={"code": 0, "data": {
                        "open_id": "ou_drive_development_self", "name": "Synthetic Self",
                        "tenant_key": "drive-development-tenant"}})
                if url.path == AUTH and query == {"action": "export", "type": "file"}:
                    return Reply("drive_auth", json={"code": 0, "data": {"auth_result": True}})
                if url.path == DOWNLOAD and not query:
                    self.downloads += 1
                    return Reply("drive_download", body=self.payload, content_type="application/octet-stream",
                                 headers={"Content-Length": str(len(self.payload)),
                                          "Content-Disposition": 'attachment; filename="' + TITLE + '"'},
                                 proof={"token": TOKEN, "bytes": len(self.payload), "sha256": digest(self.payload)})
            if method == "POST" and url.path == META and not query:
                if (set(body) - {"request_docs", "with_url"}
                        or body.get("request_docs") != [{"doc_token": TOKEN, "doc_type": "file"}]
                        or ("with_url" in body and type(body["with_url"]) is not bool)):
                    raise ValueError("exact metadata binding required")
                return Reply("drive_meta", json={"code": 0, "data": {"metas": [{
                    "doc_token": TOKEN, "doc_type": "file", "title": TITLE,
                    "url": "https://example.feishu.cn/file/" + TOKEN}]}})
            raise ValueError("unsupported route or selectors")

    def snapshot(self):
        with self._lock:
            sentinel = self.workspace / SENTINEL
            try:
                unchanged = not sentinel.is_symlink() and sentinel.is_file() and sentinel.read_bytes() == SENTINEL_BYTES
            except OSError:
                unchanged = False
            final = {**self.metadata, "bytes": len(self.payload), "sha256": digest(self.payload)}
            return {"initial_source": source_state(), "final_source": final,
                    "downloads": self.downloads, "sentinel_unchanged": unchanged}


def verify(workspace, answer, receipt, expected):
    del expected
    if (not isinstance(answer, dict) or set(answer) != {"path", "sha256"}
            or not isinstance(answer["path"], str) or not isinstance(answer["sha256"], str)
            or not re.fullmatch(r"[0-9a-fA-F]{64}", answer["sha256"])
            or answer["sha256"].lower() != SHA256 or contains_dump(answer["path"])):
        return False
    if not isinstance(receipt, dict) or receipt.get("scenario") != SCENARIO:
        return False
    audit = receipt.get("output_audit", {})
    if not isinstance(audit, dict) or audit.get("checked") is not True or audit.get("passed") is not True:
        return False
    state = receipt.get("state", {})
    if (not isinstance(state, dict) or state.get("initial_source") != source_state()
            or state.get("final_source") != source_state() or state.get("sentinel_unchanged") is not True
            or type(state.get("downloads")) is not int or state["downloads"] < 1):
        return False
    requests = receipt.get("requests")
    if not isinstance(requests, list) or any(not isinstance(r, dict)
            or type(r.get("status")) is not int or r["status"] != 200
            or r.get("route") not in {"identity", "drive_auth", "drive_meta", "drive_download"} for r in requests):
        return False
    downloads = [r for r in requests if r["route"] == "drive_download"]
    proof = {"token": TOKEN, "bytes": SIZE, "sha256": SHA256}
    if len(downloads) != state["downloads"] or any(r.get("proof") != proof for r in downloads):
        return False
    try:
        base = Path(workspace).resolve()
        reported = Path(answer["path"])
        path = reported if reported.is_absolute() else base / reported
        if path.is_symlink() or path.resolve() != base / ARTIFACT or not path.is_file() or path.stat().st_size != SIZE:
            return False
        sentinel = base / SENTINEL
        if sentinel.is_symlink() or sentinel.read_bytes() != SENTINEL_BYTES:
            return False
        actual = path.read_bytes()
        # Regenerate independently rather than trusting mutable backend payload.
        return actual == bytes(i % 251 for i in range(SIZE)) and digest(actual) == SHA256
    except (OSError, ValueError):
        return False
