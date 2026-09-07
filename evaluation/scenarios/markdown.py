"""Synthetic Markdown import contract, not evidence of live conversion fidelity.

The product delegates Markdown-to-DocxXML conversion to Docs. This bounded
backend recognizes the predeclared input and returns native fixture structure;
the independent verifier checks observed import, returned IDs and local XML.
"""
import copy
import hashlib
import json
import re
import threading
from pathlib import Path
from urllib.parse import urlsplit
import xml.etree.ElementTree as ET

from scenario_protocol import Reply, ScenarioBase

SCENARIO = "markdown-native-conversion-v1"
SOURCE = "# 你好\n\n| Name | Value |\n| --- | --- |\n| A | |\n"
DOCUMENT_ID = "docxMarkdownDevelopment01"
NATIVE_XML = ('<h1 block-id="heading_1">你好</h1><table block-id="table_1">'
              '<tr block-id="row_header"><td block-id="cell_name"><p>Name</p></td>'
              '<td block-id="cell_value"><p>Value</p></td></tr>'
              '<tr block-id="row_a"><td block-id="cell_a"><p>A</p></td>'
              '<td block-id="cell_empty"><p></p></td></tr></table>')
CREATE_PATH = "/open-apis/docs_ai/v1/documents"
FETCH_PATH = CREATE_PATH + "/" + DOCUMENT_ID + "/fetch"
# Frozen 7fd6ef3 shortcuts/drive/drive_import.go:appendDriveImportUploadDryRun
# and drive_import_common.go:createDriveImportTask/getDriveImportStatus.
DRIVE_IMPORT_POST_PATHS = frozenset({
    "/open-apis/drive/v1/medias/upload_all",
    "/open-apis/drive/v1/medias/upload_prepare",
    "/open-apis/drive/v1/medias/upload_part",
    "/open-apis/drive/v1/medias/upload_finish",
    "/open-apis/drive/v1/import_tasks",
})

TASK = {
    "id": "markdown-dev-01", "section": "markdown",
    "prompt": "Convert the synthetic Markdown in input.md to the supported native document representation. Preserve its heading, Unicode, table structure and empty cell. Save the returned native XML including block IDs as converted.xml. Report the document ID, artifact path and any unsupported conversion. Do not share or change unrelated resources.",
    "fixture": {"scenario": SCENARIO, "allow_synthetic_mutations": True,
                "answer_schema": {"type": "object", "additionalProperties": False,
                                  "required": ["document_id", "artifact_path", "unsupported"],
                                  "properties": {"document_id": {"type": "string"},
                                                 "artifact_path": {"type": "string"},
                                                 "unsupported": {"type": "array", "items": {"type": "string"}}}}},
    "expected": {"checks": ["heading and Unicode preserved", "table structure and empty cell preserved", "unsupported conversions reported"]},
}


def digest(value):
    return hashlib.sha256(value.encode("utf-8")).hexdigest()


def object_body(raw):
    def unique(pairs):
        result = {}
        for key, value in pairs:
            if key in result:
                raise ValueError("duplicate JSON key")
            result[key] = value
        return result
    value = json.loads(raw or b"{}", object_pairs_hook=unique)
    if not isinstance(value, dict):
        raise ValueError("object body required")
    return value


class Scenario(ScenarioBase):
    def __init__(self, unsupported=False):
        self._lock = threading.Lock()
        self.unsupported = unsupported
        self.created = False
        self.create_attempts = 0
        self.fetches = 0

    def setup(self, workspace):
        self.workspace = Path(workspace)
        (self.workspace / "input.md").write_bytes(SOURCE.encode("utf-8"))

    def dispatch(self, method, uri, body_bytes, headers):
        with self._lock:
            return self._dispatch(method, uri, body_bytes, headers)

    def _dispatch(self, method, uri, body_bytes, headers):
        url = urlsplit(uri)
        if url.query or url.fragment or url.scheme or url.netloc:
            raise ValueError("unexpected URI components")
        identity = next((v for k, v in headers.items() if k.lower() == "x-lark-proxy-identity"), None)
        if identity != "user":
            raise ValueError("user identity required")
        ticket = re.fullmatch(r"/open-apis/drive/v1/import_tasks/([^/%?#\\\x00-\x20]+)", url.path)
        if ((method == "POST" and url.path in DRIVE_IMPORT_POST_PATHS)
                or (method == "GET" and ticket and ticket[1] not in (".", ".."))):
            # Detect the known strategy before JSON decoding: upload_all/part
            # carry multipart bytes. This reports an adapter gap, never success.
            raise ValueError("unsupported strategy: Drive import not modeled")
        body = object_body(body_bytes)
        if method == "GET" and url.path == "/open-apis/authen/v1/user_info" and not body:
            return Reply("identity", json={"code": 0, "data": {
                "open_id": "ou_markdown_development_self", "name": "Synthetic Self",
                "tenant_key": "markdown-development-tenant"}})
        if method == "POST" and url.path == CREATE_PATH:
            if (set(body) - {"format", "content", "scene"} or body.get("format") != "markdown"
                    or body.get("content") != SOURCE or self.create_attempts):
                raise ValueError("only one exact Markdown import is supported")
            if "scene" in body and not isinstance(body["scene"], str):
                raise ValueError("scene must be text")
            self.create_attempts += 1
            if self.unsupported:
                return Reply("markdown_unsupported", json={"code": 0, "data": {
                    "result": "failed", "warnings": ["synthetic Markdown table conversion unsupported"]}},
                    proof={"source_sha256": digest(SOURCE), "created": False})
            self.created = True
            return Reply("markdown_create", json={"code": 0, "data": {"document": {
                "document_id": DOCUMENT_ID, "url": "https://fixture.example/docx/" + DOCUMENT_ID}}},
                proof={"document_id": DOCUMENT_ID, "source_sha256": digest(SOURCE), "created": True})
        if method == "POST" and url.path == FETCH_PATH and self.created:
            if set(body) - {"format", "export_option", "extra_param", "lang", "revision_id", "scene"}:
                raise ValueError("unknown or partial fetch")
            if body.get("format") != "xml":
                raise ValueError("native XML fetch required")
            for key in ("extra_param", "lang", "scene"):
                if key in body and not isinstance(body[key], str):
                    raise ValueError("text parameter required")
            if "revision_id" in body and (type(body["revision_id"]) is not int or body["revision_id"] not in (-1, 1)):
                raise ValueError("unknown revision")
            options = body.get("export_option", {})
            if (not isinstance(options, dict) or set(options) - {
                    "export_block_id", "export_style_attrs", "export_cite_extra_data"}
                    or any(type(value) is not bool for value in options.values())):
                raise ValueError("invalid export options")
            # Equivalent raw API requests and either with-ids/full preserve IDs.
            # Default/simple fetch legitimately returns the same structure without IDs.
            with_ids = options.get("export_block_id", False)
            content = NATIVE_XML
            if not with_ids:
                content = re.sub(r' block-id="[^"]+"', "", content)
            self.fetches += 1
            return Reply("markdown_fetch", json={"code": 0, "data": {"document": {
                "document_id": DOCUMENT_ID, "revision_id": 1, "content": content}}},
                proof={"document_id": DOCUMENT_ID, "native_sha256": digest(content), "with_ids": with_ids})
        raise ValueError("unsupported route")

    def snapshot(self):
        with self._lock:
            return self._snapshot()

    def _snapshot(self):
        return copy.deepcopy({"initial": {"created": False}, "final": {
            "created": self.created, "create_attempts": self.create_attempts,
            "fetches": self.fetches, "document_id": DOCUMENT_ID if self.created else None,
            "source_sha256": digest(SOURCE) if self.created else None,
            "native_sha256": digest(NATIVE_XML) if self.created else None}})


def valid_native_xml(payload):
    """Assert native node structure and values independently of emitted XML text."""
    if len(payload) > 65536 or b"<!" in payload:
        return False
    try:
        text = payload.decode("utf-8")
        root = ET.fromstring("<root>" + text + "</root>")
    except (UnicodeError, ET.ParseError):
        return False
    if [node.tag for node in root] != ["h1", "table"]:
        return False
    heading, table = root
    if heading.attrib != {"block-id": "heading_1"} or heading.text != "你好" or len(heading):
        return False
    if table.attrib != {"block-id": "table_1"} or [row.tag for row in table] != ["tr", "tr"]:
        return False
    for row, row_id, cells in zip(table, ("row_header", "row_a"), (
            (("cell_name", "Name"), ("cell_value", "Value")),
            (("cell_a", "A"), ("cell_empty", "")))):
        if row.attrib != {"block-id": row_id} or [cell.tag for cell in row] != ["td", "td"]:
            return False
        for cell, (cell_id, text_value) in zip(row, cells):
            if cell.attrib != {"block-id": cell_id} or [child.tag for child in cell] != ["p"]:
                return False
            paragraph = cell[0]
            if paragraph.attrib or len(paragraph) or (paragraph.text or "") != text_value:
                return False
    # Only indentation may appear outside leaf text; extra tail/root content fails.
    for node in root.iter():
        if node.tail and node.tail.strip():
            return False
        if node.tag not in ("h1", "p") and node.text and node.text.strip():
            return False
    return True


def verify(workspace, answer, receipt, expected):
    del expected  # The predeclared task's semantic invariants are checked below.
    if not isinstance(answer, dict) or answer.get("document_id") != DOCUMENT_ID or answer.get("unsupported") != []:
        return False
    if not isinstance(receipt, dict) or receipt.get("scenario") != SCENARIO:
        return False
    state = receipt.get("state", {})
    if not isinstance(state, dict) or state.get("initial") != {"created": False}:
        return False
    final = state.get("final", {})
    if (not isinstance(final, dict) or final.get("created") is not True
            or type(final.get("create_attempts")) is not int or final["create_attempts"] != 1
            or type(final.get("fetches")) is not int or final["fetches"] < 1
            or final.get("document_id") != DOCUMENT_ID or final.get("source_sha256") != digest(SOURCE)
            or final.get("native_sha256") != digest(NATIVE_XML)):
        return False
    requests = receipt.get("requests")
    if not isinstance(requests, list) or any(not isinstance(r, dict) or r.get("status") != 200
            or r.get("route") not in {"identity", "markdown_create", "markdown_fetch"} for r in requests):
        return False
    creates = [r for r in requests if r["route"] == "markdown_create"]
    fetches = [r for r in requests if r["route"] == "markdown_fetch"]
    if (len(creates) != 1 or len(fetches) != final["fetches"]
            or creates[0].get("proof") != {"document_id": DOCUMENT_ID, "source_sha256": digest(SOURCE), "created": True}
            or not any(r.get("proof") == {"document_id": DOCUMENT_ID, "native_sha256": digest(NATIVE_XML), "with_ids": True} for r in fetches)):
        return False
    try:
        base = Path(workspace).resolve()
        path_value = answer.get("artifact_path")
        if not isinstance(path_value, str):
            return False
        artifact = Path(path_value)
        if not artifact.is_absolute():
            artifact = base / artifact
        source = base / "input.md"
        if (artifact.is_symlink() or source.is_symlink() or artifact.resolve() != base / "converted.xml"
                or source.read_bytes() != SOURCE.encode("utf-8")):
            return False
        if artifact.stat().st_size > 65536:
            return False
        return valid_native_xml(artifact.read_bytes())
    except (OSError, ValueError):
        return False
