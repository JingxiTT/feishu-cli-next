"""Synthetic native Docs edits; no network or live-account access.

Inline identities belong only to the verifier ledger. Public API responses use
existing DocxXML/text-run fields. General block reconstruction is deliberately
unsupported until its native identity behavior can be modeled faithfully.
"""
import copy
import hashlib
from html import escape
import json
from pathlib import Path
import threading
from urllib.parse import parse_qs, urlsplit

from scenario_protocol import Reply, ScenarioBase

CONTRACT = json.loads((Path(__file__).resolve().parents[1] / "contracts/docs.json").read_text())
INITIAL = CONTRACT["initial_state"]
DOCUMENT = CONTRACT["valid_ids"]["document_id"]
TARGET = CONTRACT["valid_ids"]["target_block_id"]
UNRELATED = CONTRACT["valid_ids"]["unrelated_block_id"]
AI_PATH = "/open-apis/docs_ai/v1/documents/" + DOCUMENT
DOCX_PATH = "/open-apis/docx/v1/documents/" + DOCUMENT
EXPECTED_ANSWER = CONTRACT["expected_answer"]
TASK = {"id": CONTRACT["task_id"], "section": "docs", "prompt": CONTRACT["concrete_prompt"],
        "fixture": {"scenario": CONTRACT["scenario"], "answer_schema": CONTRACT["answer_schema"],
                    "allow_synthetic_mutations": True},
        "expected": {"answer": EXPECTED_ANSWER}}



def json_object(raw):
    def unique_pairs(pairs):
        result = {}
        for key, value in pairs:
            if key in result:
                raise ValueError("duplicate JSON field")
            result[key] = value
        return result

    def reject_constant(value):
        raise ValueError("nonfinite JSON value")

    value = json.loads(raw or b"{}", object_pairs_hook=unique_pairs, parse_constant=reject_constant)
    if not isinstance(value, dict):
        raise ValueError("JSON object required")
    return value


def canonical(value):
    return json.dumps(value, sort_keys=True, ensure_ascii=False, separators=(",", ":"))


def digest(value):
    return hashlib.sha256(canonical(value).encode()).hexdigest()


def text_of(block):
    return "".join(run["text"] for run in block.get("runs", [])) if "runs" in block else block.get("text", "")


def xml_block(block, ids):
    identity = ' block-id="' + escape(block["block_id"], quote=True) + '"' if ids else ""
    if block["kind"] == "image":
        return '<img' + identity + ' token="' + escape(block["token"], quote=True) + '"/>'
    tag = "title" if block["kind"] == "title" else "p"
    parts = []
    for run in block.get("runs", [{"text": block.get("text", ""), "style": {}}]):
        part = escape(run["text"], quote=False)
        style = run["style"]
        if style.get("bold"):
            part = "<b>" + part + "</b>"
        if style.get("link"):
            part = '<a href="' + escape(style["link"]["url"], quote=True) + '">' + part + "</a>"
        parts.append(part)
    return "<" + tag + identity + ">" + "".join(parts) + "</" + tag + ">"


def markdown_block(block):
    if block["kind"] == "image":
        return "![image](synthetic://media/" + block["token"] + ")"
    parts = []
    for run in block.get("runs", [{"text": block.get("text", ""), "style": {}}]):
        part = run["text"]
        if run["style"].get("bold"):
            part = "**" + part + "**"
        if run["style"].get("link"):
            part = "[" + part + "](" + run["style"]["link"]["url"] + ")"
        parts.append(part)
    return ("# " if block["kind"] == "title" else "") + "".join(parts)


def wire_block(block):
    result = {"block_id": block["block_id"], "parent_id": block["parent_id"]}
    if block["kind"] == "image":
        return dict(result, block_type=27, image={key: block[key] for key in ("token", "width", "height")})
    elements = []
    for run in block.get("runs", [{"text": block["text"], "style": {}}]):
        style = {}
        if run["style"].get("bold"):
            style["bold"] = True
        if run["style"].get("link"):
            style["link"] = {"url": run["style"]["link"]["url"]}
        elements.append({"text_run": {"content": run["text"], "text_element_style": style}})
    key = "heading1" if block["kind"] == "title" else "text"
    return dict(result, block_type=3 if key == "heading1" else 2, **{key: {"elements": elements}})


def integer(value, label, minimum=-1):
    if type(value) is int:
        result = value
    elif isinstance(value, str) and value.lstrip("-").isdigit():
        result = int(value)
    else:
        raise ValueError("invalid " + label)
    if result < minimum:
        raise ValueError("invalid " + label)
    return result


class Scenario(ScenarioBase):
    def __init__(self):
        self.state = copy.deepcopy(INITIAL)
        self.mutations = []
        self.lock = threading.RLock()

    def snapshot(self):
        with self.lock:
            return copy.deepcopy({"initial": INITIAL, "final": self.state, "mutations": self.mutations})

    def _reply(self, route, data, proof=None):
        return Reply(route=route, json={"code": 0, "msg": "success", "data": data},
                     proof=proof or {"read": True, "document_id": DOCUMENT})

    def _revision(self, value):
        revision = integer(value, "revision")
        if revision != -1 and revision != self.state["revision_id"]:
            raise ValueError("synthetic revision conflict; no write applied")

    def _commit(self, candidate, route):
        if canonical(candidate) == canonical(self.state):
            return {"mutation": False, "document_id": DOCUMENT}
        before = copy.deepcopy(self.state)
        candidate["revision_id"] += 1
        candidate["full_xml"] = "".join(xml_block(block, True) for block in candidate["blocks"])
        event = {"before": before, "after": copy.deepcopy(candidate), "before_hash": digest(before),
                 "after_hash": digest(candidate), "changed_block_ids": [a["block_id"] for a, b in zip(candidate["blocks"], before["blocks"]) if canonical(a) != canonical(b)]}
        self.state = candidate
        self.mutations.append(event)
        return {"mutation": True, "document_id": DOCUMENT, "before_hash": event["before_hash"],
                "after_hash": event["after_hash"], "changed_block_ids": event["changed_block_ids"]}

    def _select(self, option):
        if not isinstance(option, dict):
            raise ValueError("read_option must be an object")
        if set(option) - {"read_mode", "start_block_id", "end_block_id", "keyword", "context_before", "context_after", "max_depth"}:
            raise ValueError("unsupported read option")
        mode = option.get("read_mode", "full")
        blocks = self.state["blocks"]
        index = {block["block_id"]: n for n, block in enumerate(blocks)}
        depth = integer(option.get("max_depth", -1), "depth")
        before = integer(option.get("context_before", 0), "context", 0)
        after = integer(option.get("context_after", 0), "context", 0)
        if before > 1000 or after > 1000:
            raise ValueError("context exceeds fixture bound")
        if mode == "full":
            return blocks, mode
        if mode == "outline":
            return [block for block in blocks if block["kind"] == "title" and (depth == -1 or depth >= 1)], mode
        if mode == "range":
            start, end = option.get("start_block_id"), option.get("end_block_id")
            if start is None and end is None:
                raise ValueError("range requires an endpoint")
            if start not in (None, "0") and start not in index or end not in (None, "-1") and end not in index:
                raise ValueError("unknown block endpoint")
            first, last = index.get(start, 0), index.get(end, len(blocks) - 1)
            if first > last:
                raise ValueError("reversed range")
            selected = list(range(first, last + 1))
        elif mode == "section":
            anchor = option.get("start_block_id")
            if anchor not in index:
                raise ValueError("unknown section anchor")
            first = index[anchor]
            last = next((n for n in range(first + 1, len(blocks)) if blocks[n]["kind"] == "title"), len(blocks))
            selected = list(range(first, last))
        elif mode == "keyword":
            keyword = option.get("keyword")
            if not isinstance(keyword, str) or not keyword or len(keyword) > 4096:
                raise ValueError("invalid keyword")
            needles = [part.casefold() for part in keyword.split("|") if part]
            selected = [n for n, block in enumerate(blocks) if any(needle in text_of(block).casefold() for needle in needles)]
        else:
            raise ValueError("unsupported read mode")
        selected = sorted({neighbor for n in selected for neighbor in range(max(0, n - before), min(len(blocks), n + after + 1))})
        return [blocks[n] for n in selected], mode

    def _fetch(self, data):
        if set(data) - {"format", "export_option", "read_option", "revision_id", "lang", "scene", "extra_param"}:
            raise ValueError("unsupported fetch input")
        self._revision(data.get("revision_id", -1))
        options = data.get("export_option", {})
        if not isinstance(options, dict) or set(options) - {"export_block_id", "export_style_attrs", "export_cite_extra_data"} or any(type(value) is not bool for value in options.values()):
            raise ValueError("invalid export options")
        if "extra_param" in data and (not isinstance(data["extra_param"], str) or not isinstance(json.loads(data["extra_param"]), dict)):
            raise ValueError("invalid extra_param")
        blocks, mode = self._select(data.get("read_option", {}))
        format_ = data.get("format", "xml")
        if format_ == "xml":
            content = "".join(xml_block(block, options.get("export_block_id", False)) for block in blocks)
            if mode != "full":
                content = '<fragment mode="' + mode + '">' + content + "</fragment>"
        elif format_ == "markdown":
            content = "\n".join(markdown_block(block) for block in blocks)
        else:
            raise ValueError("unsupported format")
        return self._reply("docs_fetch", {"document": {"document_id": DOCUMENT, "revision_id": self.state["revision_id"], "content": content, "reference_map": copy.deepcopy(self.state["reference_map"])}, "complete": mode == "full", "returned_block_ids": [block["block_id"] for block in blocks]})

    def _replace(self, data):
        if set(data) - {"command", "format", "pattern", "content", "revision_id", "scene", "reference_map"}:
            raise ValueError("unsupported precise edit input")
        if data.get("command") != "str_replace":
            raise ValueError("unsupported strategy: general reconstruction needs validated native identity semantics")
        if data.get("format", "xml") not in ("xml", "markdown") or data.get("reference_map", {}) != {}:
            raise ValueError("unsupported precise edit format/reference map")
        self._revision(data.get("revision_id", -1))
        pattern, content = data.get("pattern"), data.get("content", "")
        if not isinstance(pattern, str) or not pattern or not isinstance(content, str):
            raise ValueError("literal pattern/content required")
        if any(char in content for char in "<>"):
            raise ValueError("unsupported strategy: XML replacement fragments require native inline parsing")
        matches = [(n, text_of(block).find(pattern)) for n, block in enumerate(self.state["blocks"]) if block["kind"] != "image" for _ in range(text_of(block).count(pattern))]
        if len(matches) != 1:
            raise ValueError("missing or ambiguous literal match")
        candidate = copy.deepcopy(self.state)
        index, start = matches[0]
        block = candidate["blocks"][index]
        if "runs" not in block:
            block["text"] = block["text"].replace(pattern, content, 1)
        else:
            prefix = 0
            while prefix < min(len(pattern), len(content)) and pattern[prefix] == content[prefix]:
                prefix += 1
            suffix = 0
            while suffix < min(len(pattern), len(content)) - prefix and pattern[-suffix - 1] == content[-suffix - 1]:
                suffix += 1
            characters = [(char, n) for n, run in enumerate(block["runs"]) for char in run["text"]]
            first, last = start + prefix, start + len(pattern) - suffix
            owner = characters[min(first, len(characters) - 1)][1]
            inserted = content[prefix:len(content) - suffix if suffix else None]
            characters[first:last] = [(char, owner) for char in inserted]
            for n, run in enumerate(block["runs"]):
                run["text"] = "".join(char for char, owner in characters if owner == n)
            block["text"] = text_of(block)
        proof = self._commit(candidate, "docs_update")
        return self._reply("docs_update", {"result": "success", "document": {"document_id": DOCUMENT, "revision_id": self.state["revision_id"]}}, proof)

    def _patch(self, block_id, data, query):
        if set(data) != {"update_text_elements"} or not isinstance(data["update_text_elements"], dict) or set(data["update_text_elements"]) != {"elements"}:
            raise ValueError("unsupported patch operation")
        self._revision(query.get("document_revision_id", ["-1"])[0])
        elements = data["update_text_elements"]["elements"]
        if not isinstance(elements, list):
            raise ValueError("elements must be an array")
        candidate = copy.deepcopy(self.state)
        block = next((item for item in candidate["blocks"] if item["block_id"] == block_id), None)
        if block is None or block["kind"] == "image":
            raise ValueError("unknown or non-text block")
        previous = block.get("runs", [])
        runs = []
        for n, element in enumerate(elements):
            if not isinstance(element, dict) or set(element) != {"text_run"}:
                raise ValueError("unsupported text element")
            value = element["text_run"]
            if not isinstance(value, dict) or set(value) - {"content", "text_element_style"} or not isinstance(value.get("content"), str):
                raise ValueError("invalid text run")
            style = value.get("text_element_style", {})
            if not isinstance(style, dict) or set(style) - {"bold", "link"} or type(style.get("bold", False)) is not bool:
                raise ValueError("unsupported style")
            link = style.get("link")
            if link is not None and (not isinstance(link, dict) or set(link) != {"url"} or not isinstance(link["url"], str)):
                raise ValueError("invalid link")
            old = previous[n] if len(previous) == len(elements) else None
            identity = old["native_identity"] if old else "generated_run_" + str(self.state["revision_id"]) + "_" + str(n)
            native_link = None
            if link is not None:
                old_link = old["style"].get("link") if old else None
                native_link = {"url": link["url"], "native_identity": old_link["native_identity"] if old_link and old_link["url"] == link["url"] else "generated_link_" + str(self.state["revision_id"]) + "_" + str(n)}
            runs.append({"native_identity": identity, "text": value["content"], "style": {"bold": style.get("bold", False), "link": native_link}})
        block["runs"] = runs
        block["text"] = text_of(block)
        proof = self._commit(candidate, "docs_block_patch")
        return self._reply("docs_block_patch", {"block": wire_block(block), "document_revision_id": self.state["revision_id"]}, proof)

    def dispatch(self, method, uri, body_bytes, headers):
        if headers.get("X-Lark-Proxy-Identity") not in ("user", "bot"):
            raise ValueError("unsupported identity")
        parsed = urlsplit(uri)
        query = parse_qs(parsed.query, keep_blank_values=True)
        if any(len(values) != 1 for values in query.values()):
            raise ValueError("duplicate query parameter")
        data = json_object(body_bytes)
        with self.lock:
            if method == "GET" and parsed.path == "/open-apis/authen/v1/user_info" and not query:
                return self._reply("identity", CONTRACT["transport"]["authentication_route"]["data"])
            if method == "POST" and parsed.path == AI_PATH + "/fetch" and not query:
                return self._fetch(data)
            if method == "PUT" and parsed.path == AI_PATH and not query:
                return self._replace(data)
            if parsed.path == DOCX_PATH and method == "GET" and not query:
                return self._reply("docs_metadata", {"document": {key: self.state[key] for key in ("document_id", "revision_id", "title")}})
            if parsed.path == DOCX_PATH + "/raw_content" and method == "GET" and set(query) <= {"lang"}:
                return self._reply("docs_raw_content", {"content": "\n".join(text_of(block) for block in self.state["blocks"] if block["kind"] != "image")})
            if parsed.path == DOCX_PATH + "/blocks" and method == "GET" and set(query) <= {"page_size", "page_token", "document_revision_id", "user_id_type"}:
                self._revision(query.get("document_revision_id", ["-1"])[0])
                size = integer(query.get("page_size", ["500"])[0], "page size", 1)
                offset = integer(query.get("page_token", ["0"])[0], "page token", 0)
                if size > 500 or offset > len(self.state["blocks"]):
                    raise ValueError("page out of bounds")
                blocks = self.state["blocks"][offset:offset + size]
                end = offset + len(blocks)
                more = end < len(self.state["blocks"])
                return self._reply("docs_blocks", {"items": [wire_block(block) for block in blocks], "has_more": more, "page_token": str(end) if more else ""})
            prefix = DOCX_PATH + "/blocks/"
            if parsed.path.startswith(prefix) and "/" not in parsed.path[len(prefix):] and set(query) <= {"document_revision_id", "user_id_type", "client_token"}:
                block_id = parsed.path[len(prefix):]
                if method == "PATCH":
                    return self._patch(block_id, data, query)
                if method == "GET":
                    self._revision(query.get("document_revision_id", ["-1"])[0])
                    block = next((item for item in self.state["blocks"] if item["block_id"] == block_id), None)
                    if block:
                        return self._reply("docs_block_get", {"block": wire_block(block)})
            raise ValueError("unsupported route or foreign resource ID")


def preserved_except_prefix(state):
    value = copy.deepcopy(state)
    value.pop("revision_id")
    value.pop("full_xml")
    block = value["blocks"][1]
    block.pop("text")
    block["runs"][0].pop("text")
    return canonical(value)


def verify(workspace, answer, receipt, expected):
    """Trust only framework-owned final state, native identities and proof chain."""
    try:
        if canonical(answer) != canonical(EXPECTED_ANSWER) or receipt.get("scenario") != TASK["fixture"]["scenario"]:
            return False
        snapshot = receipt["state"]
        initial, final, mutations = snapshot["initial"], snapshot["final"], snapshot["mutations"]
        if canonical(initial) != canonical(INITIAL) or not mutations:
            return False
        preserved = preserved_except_prefix(INITIAL)
        previous = initial
        for event in mutations:
            before, after = event["before"], event["after"]
            if canonical(before) != canonical(previous) or digest(before) != event["before_hash"] or digest(after) != event["after_hash"]:
                return False
            if type(after["revision_id"]) is not int or after["revision_id"] != before["revision_id"] + 1:
                return False
            if preserved_except_prefix(after) != preserved or event["changed_block_ids"] != [TARGET]:
                return False
            if not any(request.get("status") == 200 and request.get("route") in ("docs_update", "docs_block_patch") and request.get("proof", {}).get("mutation") is True and request["proof"].get("document_id") == DOCUMENT and request["proof"].get("before_hash") == event["before_hash"] and request["proof"].get("after_hash") == event["after_hash"] for request in receipt["requests"]):
                return False
            previous = after
        if canonical(final) != canonical(previous):
            return False
        desired = copy.deepcopy(INITIAL)
        desired["revision_id"] = final["revision_id"]
        desired["blocks"][1]["text"] = EXPECTED_ANSWER["paragraph_text"]
        desired["blocks"][1]["runs"][0]["text"] = "New "
        desired["full_xml"] = INITIAL["full_xml"].replace("Old <a", "New <a", 1)
        return canonical(final) == canonical(desired)
    except (KeyError, TypeError, ValueError, IndexError, AttributeError):
        return False
