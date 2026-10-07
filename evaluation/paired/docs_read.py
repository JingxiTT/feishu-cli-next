# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Read one exact document fact and its native block reference; no mutations."""
import copy
import json
from urllib.parse import urlsplit

from scenarios import docs as native
from paired.docs_revision import reject_unmodeled_history

FACT = "APR-0042"
BLOCK = native.UNRELATED
INITIAL = copy.deepcopy(native.INITIAL)
INITIAL["blocks"][2]["text"] = "Launch approval code: APR-0042. Keep résumé 😀 and 001 unchanged."
INITIAL["full_xml"] = "".join(native.xml_block(block, True) for block in INITIAL["blocks"])
ANSWER = {"approval_code": FACT, "block_id": BLOCK}
TASK = {
    "id": "paired-docs-read", "section": "docs",
    "prompt": f"Read document {native.DOCUMENT}. Find the launch approval code and the native block ID containing it. Return JSON with approval_code and block_id. Do not modify the document.",
    "fixture": {"scenario": "paired-docs-read-v1", "scenario_section": "paired-docs-read",
                "answer_schema": {"type": "object", "additionalProperties": False,
                    "required": ["approval_code", "block_id"], "properties": {
                        "approval_code": {"type": "string"}, "block_id": {"type": "string"}}}},
    "expected": {"answer": ANSWER},
}


class Scenario(native.Scenario):
    def __init__(self):
        super().__init__()
        self.state = copy.deepcopy(INITIAL)
        self.observations = []

    def dispatch(self, method, uri, body_bytes, headers):
        if method != "GET" and not (method == "POST" and urlsplit(uri).path == native.AI_PATH + "/fetch"):
            raise ValueError("read-only task rejects document mutations")
        with self.lock:
            reject_unmodeled_history(self, method, uri, body_bytes, headers)
            reply = super().dispatch(method, uri, body_bytes, headers)
            # Native fetch exposes block references inside requested XML, not
            # the historical fixture's synthetic returned_block_ids sidecar.
            if reply.route == "docs_fetch":
                reply.json["data"].pop("returned_block_ids", None)
            rendered = json.dumps(reply.json, ensure_ascii=False)
            observed = {"document_id": native.DOCUMENT, "fact_and_block_returned": FACT in rendered and BLOCK in rendered,
                        "response_sha256": native.digest(reply.json)}
            self.observations.append(observed)
            reply.proof = observed
            return reply

    def snapshot(self):
        with self.lock:
            return copy.deepcopy({"initial": INITIAL, "final": self.state,
                                  "observations": self.observations, "mutations": self.mutations})


def verify(workspace, answer, receipt, expected):
    del workspace, expected
    try:
        state = receipt["state"]
        if (native.canonical(answer) != native.canonical(ANSWER)
                or receipt["scenario"] != TASK["fixture"]["scenario"]
                or native.canonical(state["initial"]) != native.canonical(INITIAL)
                or native.canonical(state["final"]) != native.canonical(INITIAL)
                or state["mutations"] != []):
            return False
        observations = state["observations"]
        successful = [r for r in receipt["requests"] if r.get("status") == 200]
        return (any(o.get("fact_and_block_returned") is True for o in observations)
                and len(observations) == len(successful)
                and all(r.get("route") in {"identity", "docs_fetch", "docs_metadata", "docs_raw_content", "docs_blocks", "docs_block_get"}
                        and native.canonical(r.get("proof")) == native.canonical(o)
                        for r, o in zip(successful, observations)))
    except (KeyError, TypeError, AttributeError):
        return False
