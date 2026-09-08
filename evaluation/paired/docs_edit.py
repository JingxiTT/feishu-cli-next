"""Paired native Docs edit, with live-observed replacement-span semantics.

The old development fixture remains untouched. Its rendering, routing, typed
native PATCH, paragraph replacement and independent verifier are reused here.
"""
import copy
from html import escape

from scenarios import docs as native

TASK = copy.deepcopy(native.TASK)
TASK.update(id="paired-docs-edit", section="docs")
TASK["fixture"].update(scenario="paired-docs-edit-v1", scenario_section="paired-docs-edit")
TASK["prompt"] = (
    f"In document {native.DOCUMENT}, change only the exact phrase Old status to New status. "
    "Preserve native bold formatting and the existing hyperlink on status, the Unicode "
    "text, and every other block/resource. Do not create, share or delete resources. "
    "Return JSON with document_id and paragraph_text (the complete changed paragraph)."
)


class Scenario(native.Scenario):
    def _fetch(self, data):
        reply = super()._fetch(data)
        reply.json["data"].pop("returned_block_ids", None)
        return reply

    def _replace(self, data):
        if data.get("command") != "str_replace":
            return super()._replace(data)
        data = copy.deepcopy(data)
        content = data.get("content", "")
        if not isinstance(content, str):
            raise ValueError("replacement content must be text")
        format_ = data.get("format", "xml")
        if format_ == "xml":
            # The complete matched span is replaced by the supplied native
            # representation. Plain XML has no marks; unchanged suffix text
            # inside the match does not inherit the old bold/link formatting.
            native.parse_inline_xml(content)
        elif format_ == "markdown":
            if any(char in content for char in "*_`[]<>\\"):
                raise ValueError(native.UNSUPPORTED_XML)
            content = escape(content, quote=False)
            data["format"] = "xml"
        else:
            raise ValueError("unsupported replacement format")
        data["content"] = "<span>" + content + "</span>"
        return super()._replace(data)


def verify(workspace, answer, receipt, expected):
    if not isinstance(receipt, dict) or receipt.get("scenario") != TASK["fixture"]["scenario"]:
        return False
    adapted = dict(receipt, scenario=native.TASK["fixture"]["scenario"])
    return native.verify(workspace, answer, adapted, expected)
