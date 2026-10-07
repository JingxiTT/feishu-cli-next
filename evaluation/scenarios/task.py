# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Stateful, non-forwarding Task completion development fixture."""
from copy import deepcopy
import json
from pathlib import Path
import re
from urllib.parse import parse_qs, urlsplit

from scenario_protocol import Reply, ScenarioBase

_CONTRACT = json.loads((Path(__file__).parents[1] / "contracts/task.json").read_text())
_INITIAL = _CONTRACT["fixture"]
TARGET = _INITIAL["target_id"]
SELF = _INITIAL["binding"]["open_id"]
SCENARIO = _CONTRACT["scenario"]
_BASE = "/open-apis/task/v2/tasks"
TASK = {
    "id": "task-dev-01",
    "section": "task",
    "prompt": _CONTRACT["prompt"],
    "fixture": {
        "scenario": SCENARIO,
        "allow_synthetic_mutations": True,
        "answer_schema": {
            "type": "object", "additionalProperties": False,
            "required": ["task_id", "completed", "assignee_ids"],
            "properties": {
                "task_id": {"type": "string"},
                "completed": {"type": "boolean"},
                "assignee_ids": {"type": "array", "items": {"type": "string"}},
            },
        },
    },
    "expected": deepcopy(_CONTRACT["expected"]),
}


def _same(left, right):
    """Python's bool/int equality must not erase native JSON types."""
    return json.dumps(left, sort_keys=True, separators=(",", ":")) == json.dumps(
        right, sort_keys=True, separators=(",", ":"))


def _completed(value):
    return isinstance(value, str) and re.fullmatch(r"0*[1-9][0-9]*", value) is not None


def _query(uri, allowed):
    query = parse_qs(urlsplit(uri).query, keep_blank_values=True, strict_parsing=True)
    if set(query) - allowed or any(len(values) != 1 for values in query.values()):
        raise ValueError("unsupported or repeated query parameter")
    values = {key: value[0] for key, value in query.items()}
    if values.get("user_id_type", "open_id") != "open_id":
        raise ValueError("fixture uses open_id")
    return values


class Scenario(ScenarioBase):
    def __init__(self):
        self.tasks = deepcopy(_INITIAL["tasks"])
        self.visibility = deepcopy(_INITIAL["visibility_by_task"])
        self.notifications = []
        self.transitions = []
        self.forbidden_mutations = []
        self.revision = 0

    def snapshot(self):
        return deepcopy({"tasks": self.tasks, "visibility": self.visibility,
                         "notifications": self.notifications, "transitions": self.transitions,
                         "forbidden_mutations": self.forbidden_mutations,
                         "revision": self.revision})

    def _forbidden(self, method, path, reason):
        if method not in ("GET", "HEAD", "OPTIONS"):
            self.forbidden_mutations.append({"method": method, "path": path, "reason": reason})
        raise ValueError(reason)

    def _reply(self, route, tasks, data, changed=False):
        return Reply(route, json={"code": 0, "data": deepcopy(data)}, proof={
            "operation": "mutation" if route == "task_patch" else "read",
            "revision": self.revision, "changed": changed,
            "tasks": deepcopy(tasks),
        })

    def dispatch(self, method, uri, body_bytes, headers):
        path = urlsplit(uri).path
        identity = next((value for key, value in headers.items()
                         if key.lower() == "x-lark-proxy-identity"), None)
        if identity != "user":
            return self._forbidden(method, path, "private fixture requires its bound user")
        if method == "GET" and path == "/open-apis/authen/v1/user_info":
            if body_bytes or _query(uri, set()):
                raise ValueError("identity is a bodyless read")
            return Reply("identity", json={"code": 0, "data": {
                "open_id": SELF, "name": "Synthetic Task Fixture",
                "tenant_key": _INITIAL["binding"]["tenant_key"],
            }})
        if method == "GET" and path == _BASE:
            if body_bytes:
                raise ValueError("task list is a bodyless read")
            query = _query(uri, {"user_id_type", "type", "completed", "page_size", "page_token"})
            if query.get("type", "my_tasks") != "my_tasks":
                raise ValueError("unsupported task list type")
            if query.get("completed") not in (None, "true", "false"):
                raise ValueError("invalid completion filter")
            size_raw = query.get("page_size", "50")
            if not re.fullmatch(r"[0-9]{1,3}", size_raw) or not 1 <= int(size_raw) <= 100:
                raise ValueError("invalid page_size")
            if query.get("page_token", "") not in ("", "task-page-1"):
                raise ValueError("invalid page_token")
            tasks = [deepcopy(self.tasks[key]) for key in sorted(self.tasks)]
            if "completed" in query:
                wanted = query["completed"] == "true"
                tasks = [item for item in tasks if _completed(item["completed_at"]) == wanted]
            offset = 1 if query.get("page_token") == "task-page-1" else 0
            items = tasks[offset:offset + int(size_raw)]
            has_more = offset + len(items) < len(tasks)
            return self._reply("task_list", items, {"items": items, "has_more": has_more,
                               "page_token": "task-page-1" if has_more else ""})
        if path.startswith(_BASE + "/") and "/" not in path[len(_BASE) + 1:]:
            guid = path[len(_BASE) + 1:]
            if method not in ("GET", "PATCH"):
                return self._forbidden(method, path, "unsupported task operation")
            _query(uri, {"user_id_type"})
            if guid not in self.tasks:
                if method == "PATCH":
                    self.forbidden_mutations.append({"method": method, "path": path, "reason": "foreign task"})
                return Reply("task_missing", json={"code": 1470404, "msg": "task not found"})
            if method == "GET":
                if body_bytes:
                    raise ValueError("task get is a bodyless read")
                current = self.tasks[guid]
                return self._reply("task_get", [current], {"task": current})
            return self._patch(guid, path, body_bytes)
        return self._forbidden(method, path, "unlisted fixture route")

    def _patch(self, guid, path, body_bytes):
        try:
            body = json.loads(body_bytes)
        except (ValueError, UnicodeDecodeError):
            body = None
        invalid = Reply("task_invalid", json={"code": 1470400, "msg": "invalid task update"})
        if not isinstance(body, dict) or set(body) - {"task", "update_fields"}:
            return invalid
        task, fields = body.get("task"), body.get("update_fields")
        if not isinstance(task, dict) or not isinstance(fields, list) or not fields:
            return invalid
        if not all(isinstance(field, str) for field in fields) or len(fields) != len(set(fields)):
            return invalid
        if set(fields) - {"completed_at", "summary", "description"} or "members" in task:
            return self._forbidden("PATCH", path, "unsupported task field or assignment mutation")
        if any(field not in task or not isinstance(task[field], str) for field in fields):
            return invalid
        if "completed_at" in fields and task["completed_at"] != "0" and not _completed(task["completed_at"]):
            return invalid
        if "summary" in fields and not task["summary"].strip():
            return invalid
        before = deepcopy(self.tasks[guid])
        after = deepcopy(before)
        for field in fields:
            after[field] = task[field]
        if "completed_at" in fields:
            after["status"] = "done" if _completed(after["completed_at"]) else "todo"
        changed = not _same(before, after)
        if changed:
            self.revision += 1
            self.tasks[guid] = after
            self.transitions.append({"revision": self.revision, "task_id": guid,
                                     "before": before, "after": deepcopy(after), "fields": fields[:]})
            if not _completed(before["completed_at"]) and _completed(after["completed_at"]):
                self.notifications.append({"type": "task_completed", "task_id": guid, "recipient_id": SELF})
        return self._reply("task_patch", [after], {"task": after}, changed)


def verify(workspace, answer, receipt, expected):
    """Score immutable backend observations, never agent-written files or claims."""
    del workspace
    try:
        if receipt.get("scenario") != SCENARIO or not isinstance(answer, dict):
            return False
        expected_answer = expected.get("answer", _CONTRACT["expected"]["answer"])
        if set(answer) != {"task_id", "completed", "assignee_ids"} or answer["completed"] is not True:
            return False
        if not _same(answer, expected_answer):
            return False
        state = receipt["state"]
        final = state["tasks"]
        if set(final) != set(_INITIAL["tasks"]) or state["forbidden_mutations"]:
            return False
        if not _same(state["visibility"], _INITIAL["visibility_by_task"]):
            return False
        current = final[TARGET]
        if not _completed(current.get("completed_at")) or current.get("status") != "done":
            return False
        allowed = {"completed_at", "status"}
        for guid, before in _INITIAL["tasks"].items():
            after = final[guid]
            if guid != TARGET and not _same(before, after):
                return False
            if guid == TARGET and (set(before) != set(after) or not _same(
                    {k: v for k, v in before.items() if k not in allowed},
                    {k: v for k, v in after.items() if k not in allowed})):
                return False
        transitions = state["transitions"]
        if not transitions or type(state["revision"]) is not int or state["revision"] != len(transitions):
            return False
        prior = deepcopy(_INITIAL["tasks"][TARGET])
        for revision, change in enumerate(transitions, 1):
            if (change["task_id"] != TARGET or type(change["revision"]) is not int
                    or change["revision"] != revision or not _same(change["before"], prior)):
                return False
            after = change["after"]
            if set(after) != set(prior) or any(not _same(prior[key], after[key]) for key in prior if key not in allowed):
                return False
            if not _completed(after.get("completed_at")) or after.get("status") != "done":
                return False
            prior = after
        if not _same(prior, current):
            return False
        notifications = [{"type": "task_completed", "task_id": TARGET, "recipient_id": SELF}]
        if not _same(state["notifications"], notifications):
            return False
        # A task returned in a successful mutation/read must reflect final completion
        # and membership. PATCH response evidence is sufficient; no forced reread.
        proof_found = False
        applied_revisions = set()
        for request in receipt["requests"]:
            proof = request.get("proof", {})
            if request.get("status") != 200 or request.get("route") not in ("task_patch", "task_get", "task_list"):
                continue
            revision = proof.get("revision")
            if type(revision) is not int or not 0 <= revision <= state["revision"]:
                return False
            operation = "mutation" if request["route"] == "task_patch" else "read"
            if proof.get("operation") != operation or not isinstance(proof.get("tasks"), list):
                return False
            if request["route"] == "task_patch" and proof.get("changed") is True:
                if revision == 0 or not _same(proof["tasks"], [transitions[revision - 1]["after"]]):
                    return False
                applied_revisions.add(revision)
            if revision == state["revision"] and any(_same(task, current) for task in proof.get("tasks", [])):
                proof_found = True
        return proof_found and applied_revisions == set(range(1, state["revision"] + 1))
    except (KeyError, TypeError, ValueError, AttributeError):
        return False
