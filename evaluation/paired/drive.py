# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Paired small-file upload/download task using real Drive wire shapes.

Both CLIs may use shortcuts or raw multipart/binary API calls. The frozen
catalog omits upload/download methods; its registered metadata/listing reads
are also supported. This is a 4 KiB roundtrip, not multipart-upload scalability.
"""
import copy
from email import policy
from email.parser import BytesParser
from pathlib import Path
import threading
from urllib.parse import parse_qsl, urlsplit
import zlib

from scenario_protocol import Reply, ScenarioBase
from scenarios.drive import audit_output, contains_dump, digest, object_body

SCENARIO = "paired-drive"
PARENT = "fldPairedDrivePrivate01"
TOKEN = "filePairedDriveUpload01"
SELF = "ou_paired_drive_self"
NAME = "paired-drive.bin"
SOURCE = "inputs/" + NAME
ARTIFACT = "downloads/" + NAME
SIZE = 4096
PAYLOAD = bytes(i % 251 for i in range(SIZE))
SHA256 = digest(PAYLOAD)
SENTINEL = ".paired-drive-preserve.txt"
SENTINEL_BYTES = "Unrelated local content Ω\n".encode()
UPLOAD = "/open-apis/drive/v1/files/upload_all"
DOWNLOAD = "/open-apis/drive/v1/files/" + TOKEN + "/download"
META = "/open-apis/drive/v1/metas/batch_query"
REPORT = "/open-apis/drive/v1/lark_cli_file_event/report"
PARENT_STATE = {"token": PARENT, "type": "folder", "owner_id": SELF, "acl": "self_only"}
UNRELATED = {"token": "filePairedDriveUnrelated01", "parent": PARENT,
             "name": "keep.txt", "sha256": digest(b"Do not change this remote file\n"), "acl": "self_only"}
TASK = {
    "id": "paired-drive", "section": "drive",
    "prompt": "Upload inputs/paired-drive.bin to the synthetic private Drive folder fldPairedDrivePrivate01, keeping the name paired-drive.bin. Download the newly uploaded file into downloads/paired-drive.bin and verify its bytes and SHA-256 against the original local source. Leave the uploaded copy in the folder. Report its returned file token, the workspace-relative downloaded path, and SHA-256. Preserve the source file and all unrelated local and remote resources. Keep the uploaded file private to the current account (a bot granting that same account access is allowed). Do not print/encode binary contents in tool output or your answer.",
    "fixture": {"scenario": SCENARIO, "scenario_section": SCENARIO,
                "allow_synthetic_mutations": True, "answer_schema": {
                    "type": "object", "additionalProperties": False,
                    "required": ["file_token", "path", "sha256"],
                    "properties": {"file_token": {"type": "string"}, "path": {"type": "string"}, "sha256": {"type": "string"}}}},
    "expected": {"checks": ["one signed upload of source bytes into private parent", "signed download of returned identity", "exact actual source and downloaded bytes/hash", "unrelated resources preserved", "no binary dump"]},
}


def upload_fields(raw, headers):
    content_type = next((v for k, v in headers.items() if k.lower() == "content-type"), "")
    if not isinstance(content_type, str) or "\r" in content_type or "\n" in content_type or len(raw) > 65536:
        raise ValueError("invalid multipart input")
    message = BytesParser(policy=policy.default).parsebytes(
        ("Content-Type: " + content_type + "\r\nMIME-Version: 1.0\r\n\r\n").encode() + raw)
    if message.get_content_type() != "multipart/form-data" or not message.is_multipart() or message.defects:
        raise ValueError("multipart/form-data required")
    fields = {}
    for part in message.iter_parts():
        name = part.get_param("name", header="content-disposition")
        if (part.is_multipart() or part.defects or part.get_content_disposition() != "form-data"
                or name not in {"file", "file_name", "size", "parent_type", "parent_node", "checksum"}
                or name in fields):
            raise ValueError("invalid or duplicated multipart field")
        payload = part.get_payload(decode=True)
        if payload is None:
            raise ValueError("missing upload bytes")
        fields[name] = payload if name == "file" else payload.decode("utf-8")
    return fields


def file_state(payload=PAYLOAD):
    return {"token": TOKEN, "parent": PARENT, "name": NAME, "type": "file",
            "owner_id": SELF, "acl": "self_only", "bytes": len(payload), "sha256": digest(payload)}


def exact_file(workspace, relative, expected):
    path = Path(workspace) / relative
    try:
        return (not path.is_symlink() and path.resolve() == Path(workspace).resolve() / relative
                and path.is_file() and path.stat().st_size == len(expected) and path.read_bytes() == expected)
    except (OSError, ValueError):
        return False


class Scenario(ScenarioBase):
    audit_output = staticmethod(audit_output)

    def __init__(self):
        self._lock = threading.Lock()
        self.parent = copy.deepcopy(PARENT_STATE)
        self.unrelated = copy.deepcopy(UNRELATED)
        self.uploaded = None
        self.payload = None
        self.uploads = self.downloads = self.reports = 0
        self.upload_identity = None
        self.grants = []

    def setup(self, workspace):
        self.workspace = Path(workspace)
        (self.workspace / "inputs").mkdir()
        (self.workspace / SOURCE).write_bytes(PAYLOAD)
        (self.workspace / SENTINEL).write_bytes(SENTINEL_BYTES)
        if (self.workspace / ARTIFACT).exists():
            raise ValueError("download artifact must start absent")

    def dispatch(self, method, uri, body_bytes, headers):
        with self._lock:
            url = urlsplit(uri)
            pairs = parse_qsl(url.query, keep_blank_values=True)
            query = dict(pairs)
            if url.scheme or url.netloc or url.fragment or len(query) != len(pairs):
                raise ValueError("relative URI and unique selectors required")
            identity = next((v for k, v in headers.items() if k.lower() == "x-lark-proxy-identity"), None)
            if identity not in {"user", "bot"}:
                raise ValueError("synthetic user or bot required")
            if method == "POST" and url.path == UPLOAD and not query:
                fields = upload_fields(body_bytes, headers)
                if (self.uploads or set(fields) - {"file", "file_name", "size", "parent_type", "parent_node", "checksum"}
                        or fields.get("file") != PAYLOAD or fields.get("file_name") != NAME
                        or fields.get("size") != str(SIZE) or fields.get("parent_type") != "explorer"
                        or fields.get("parent_node") != PARENT
                        or ("checksum" in fields and fields["checksum"] != str(zlib.adler32(PAYLOAD)))):
                    raise ValueError("one exact source upload into the private parent required")
                self.uploads += 1
                self.upload_identity = identity
                self.payload = fields["file"]
                self.uploaded = file_state(self.payload)
                return Reply("paired_drive_upload", json={"code": 0, "data": {"file_token": TOKEN}}, proof={**copy.deepcopy(self.uploaded), "identity": identity})
            body = object_body(body_bytes)
            if method == "GET" and not body:
                if url.path == "/open-apis/authen/v1/user_info" and not query:
                    return Reply("identity", json={"code": 0, "data": {"open_id": SELF, "name": "Synthetic Self", "tenant_key": "paired-drive-tenant"}})
                if url.path == DOWNLOAD and not query and self.uploaded:
                    self.downloads += 1
                    return Reply("paired_drive_download", body=self.payload, content_type="application/octet-stream",
                                 headers={"Content-Length": str(len(self.payload)), "Content-Disposition": 'attachment; filename="' + NAME + '"'},
                                 proof=file_state(self.payload))
                for token, kind in ((PARENT, "folder"), (TOKEN, "file")):
                    if token == TOKEN and not self.uploaded:
                        continue
                    prefix = "/open-apis/drive/v1/permissions/" + token
                    if url.path == prefix + "/members/auth" and kind == "file" and query == {"type": "file", "action": "export"}:
                        return Reply("paired_drive_auth", json={"code": 0, "data": {"auth_result": True}})
                    if url.path == prefix + "/members" and query == {"type": kind}:
                        return Reply("paired_drive_members", json={"code": 0, "data": {"items": [{"member_type": "openid", "member_id": SELF, "perm": "full_access"}]}})
                    if url.path in (prefix + "/public", prefix.replace("/v1/", "/v2/") + "/public") and query == {"type": kind}:
                        return Reply("paired_drive_settings", json={"code": 0, "data": {"permission_public": {"link_share_entity": "closed", "external_access": False}}})
                if (url.path == "/open-apis/drive/v1/files" and query.get("folder_token") == PARENT
                        and set(query) <= {"folder_token", "page_size", "page_token", "user_id_type", "order_by", "direction"}
                        and query.get("user_id_type", "open_id") == "open_id"
                        and query.get("order_by", "EditedTime") in {"EditedTime", "CreatedTime"}
                        and query.get("direction", "DESC") in {"ASC", "DESC"}
                        and query.get("page_size", "100").isdigit() and 1 <= int(query.get("page_size", "100")) <= 200):
                    files = [{"token": UNRELATED["token"], "name": UNRELATED["name"], "type": "file", "parent_token": PARENT}]
                    if self.uploaded:
                        files.append({"token": TOKEN, "name": NAME, "type": "file", "parent_token": PARENT})
                    if query.get("direction", "DESC") == "DESC":
                        files.reverse()
                    cursor = query.get("page_token", "")
                    if cursor not in {"", "paired-drive-offset-1"} or (cursor and len(files) != 2):
                        raise ValueError("invalid folder continuation")
                    offset = 1 if cursor else 0
                    size = int(query.get("page_size", "100"))
                    page = files[offset:offset + size]
                    has_more = offset + len(page) < len(files)
                    data = {"files": page, "has_more": has_more}
                    if has_more:
                        data["next_page_token"] = "paired-drive-offset-1"
                    return Reply("paired_drive_list", json={"code": 0, "data": data})
            if method == "POST" and url.path == META and query in ({}, {"user_id_type": "open_id"}):
                docs = body.get("request_docs")
                allowed = [{"doc_token": PARENT, "doc_type": "folder"}]
                if self.uploaded:
                    allowed.append({"doc_token": TOKEN, "doc_type": "file"})
                if (set(body) - {"request_docs", "with_url"} or not isinstance(docs, list) or not docs
                        or len(docs) > len(allowed) or any(doc not in allowed for doc in docs)
                        or len({str(doc) for doc in docs}) != len(docs)
                        or ("with_url" in body and type(body["with_url"]) is not bool)):
                    raise ValueError("exact owned metadata binding required")
                return Reply("paired_drive_meta", json={"code": 0, "data": {"metas": [
                    {**doc, "owner_id": SELF, "title": NAME if doc["doc_token"] == TOKEN else "Private fixture",
                     "url": "https://example.feishu.cn/" + ("file/" + TOKEN if doc["doc_token"] == TOKEN else "drive/folder/" + PARENT)} for doc in docs]}})
            if method == "POST" and url.path == "/open-apis/drive/v1/permissions/" + TOKEN + "/members":
                # Both frozen shortcuts grant the current CLI user access after
                # a bot upload. Record that real effect; no other recipient or
                # target is authorized, and raw/user uploads need no grant.
                if (not self.uploaded or self.upload_identity != "bot" or identity != "bot" or self.grants
                        or query != {"type": "file", "need_notification": "false"}
                        or body != {"member_type": "openid", "member_id": SELF, "perm": "full_access", "type": "user"}):
                    raise ValueError("only current-account grant after bot upload allowed")
                effect = {"token": TOKEN, "identity": identity, "recipient": SELF, "perm": "full_access", "notification": False}
                self.grants.append(effect)
                return Reply("paired_drive_self_grant", json={"code": 0, "data": {"member": body}}, proof=copy.deepcopy(effect))
            if method == "POST" and url.path == REPORT and not query:
                tags = body.get("tags")
                if (not self.uploaded or self.reports or set(body) != {"file_scene", "scene", "operation", "tags"}
                        or body.get("file_scene") != "lark-cli" or body.get("scene") != "upload" or body.get("operation") != "upload"
                        or not isinstance(tags, dict) or set(tags) != {"code", "api_path", "command", "resource_type", "status", "mount_point", "file_token"}
                        or not all(isinstance(v, str) for v in tags.values()) or tags["file_token"] != TOKEN
                        or tags["api_path"] != UPLOAD or tags["mount_point"] != "explorer"
                        or tags["resource_type"] != "file" or tags["status"] != "success" or tags["code"] != ""):
                    raise ValueError("invalid upload telemetry")
                self.reports += 1
                return Reply("paired_drive_report", json={"code": 0, "data": {}}, proof={"token": TOKEN, "telemetry": True})
            raise ValueError("unsupported route or selectors")

    def snapshot(self):
        with self._lock:
            return copy.deepcopy({"initial_parent": PARENT_STATE, "final_parent": self.parent,
                                  "initial_unrelated": UNRELATED, "final_unrelated": self.unrelated,
                                  "uploaded": self.uploaded, "payload": file_state(self.payload) if self.payload is not None else None,
                                  "uploads": self.uploads, "downloads": self.downloads, "reports": self.reports,
                                  "upload_identity": self.upload_identity, "grants": self.grants,
                                  "source_unchanged": exact_file(self.workspace, SOURCE, PAYLOAD),
                                  "sentinel_unchanged": exact_file(self.workspace, SENTINEL, SENTINEL_BYTES)})


def verify(workspace, answer, receipt, expected):
    del expected
    if (not isinstance(answer, dict) or set(answer) != {"file_token", "path", "sha256"}
            or answer.get("file_token") != TOKEN or not isinstance(answer.get("path"), str)
            or not isinstance(answer.get("sha256"), str) or answer["sha256"].lower() != SHA256
            or contains_dump(answer["path"]) or not isinstance(receipt, dict) or receipt.get("scenario") != SCENARIO):
        return False
    audit = receipt.get("output_audit", {})
    state = receipt.get("state", {})
    if (not isinstance(audit, dict) or audit.get("checked") is not True or audit.get("passed") is not True
            or not isinstance(state, dict) or state.get("initial_parent") != PARENT_STATE or state.get("final_parent") != PARENT_STATE
            or state.get("initial_unrelated") != UNRELATED or state.get("final_unrelated") != UNRELATED
            or state.get("uploaded") != file_state() or state.get("payload") != file_state()
            or type(state.get("uploads")) is not int or state["uploads"] != 1
            or type(state.get("downloads")) is not int or state["downloads"] < 1
            or type(state.get("reports")) is not int or state["reports"] not in (0, 1)
            or state.get("source_unchanged") is not True or state.get("sentinel_unchanged") is not True):
        return False
    identity = state.get("upload_identity")
    grant = {"token": TOKEN, "identity": "bot", "recipient": SELF, "perm": "full_access", "notification": False}
    grants = state.get("grants")
    if (identity not in {"user", "bot"} or not isinstance(grants, list) or grants not in ([], [grant])
            or (grants and (identity != "bot" or grants[0].get("notification") is not False))):
        return False
    requests = receipt.get("requests")
    reads = {"identity", "paired_drive_auth", "paired_drive_members", "paired_drive_settings", "paired_drive_list", "paired_drive_meta"}
    allowed = reads | {"paired_drive_upload", "paired_drive_download", "paired_drive_report", "paired_drive_self_grant"}
    if not isinstance(requests, list) or any(not isinstance(r, dict) or r.get("signature_valid") is not True
            or type(r.get("status")) is not int or r["status"] != 200 or r.get("route") not in allowed for r in requests):
        return False
    for route, count in (("paired_drive_upload", 1), ("paired_drive_download", state["downloads"]), ("paired_drive_report", state["reports"]), ("paired_drive_self_grant", len(grants))):
        matched = [r for r in requests if r["route"] == route]
        proof = ({"token": TOKEN, "telemetry": True} if route.endswith("report") else grant if route.endswith("self_grant")
                 else {**file_state(), "identity": identity} if route.endswith("upload") else file_state())
        if (len(matched) != count or any(r.get("proof") != proof for r in matched)
                or (route.endswith("self_grant") and any(r["proof"].get("notification") is not False for r in matched))):
            return False
    if next(i for i, r in enumerate(requests) if r["route"] == "paired_drive_upload") > next(i for i, r in enumerate(requests) if r["route"] == "paired_drive_download"):
        return False
    try:
        base = Path(workspace).resolve()
        reported = Path(answer["path"])
        resolved = reported if reported.is_absolute() else base / reported
        return (resolved.resolve() == base / ARTIFACT and not resolved.is_symlink()
                and exact_file(base, ARTIFACT, bytes(i % 251 for i in range(SIZE)))
                and digest((base / ARTIFACT).read_bytes()) == SHA256
                and exact_file(base, SOURCE, bytes(i % 251 for i in range(SIZE)))
                and exact_file(base, SENTINEL, SENTINEL_BYTES))
    except (OSError, ValueError):
        return False
