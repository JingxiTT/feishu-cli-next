# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Supplemental synthetic business tasks; no live/model access or forwarding.

The reviewed contract models only these finite fixtures. Approval effects are
fixture ledger events, not evidence of real notification delivery. Mindnote
partial-field merge/idempotency is contract-declared synthetic behavior. Current
product code preserves pagination failures, recognizes native POST reads, and
leaves mixed/existing-node update receipts unknown pending readback. Lost-response
fault injection remains in the product process suite, outside these 12 tasks.
"""
from copy import deepcopy
import json
from pathlib import Path
import threading
from urllib.parse import parse_qs, urlsplit

from scenario_protocol import Reply, ScenarioBase

CONTRACT = json.loads((Path(__file__).parents[1] / "contracts/generated-business.json").read_text())
SCENARIO = "generated-business-v1"
_PROPOSALS = {item["id"]: item for item in CONTRACT["proposals"]}
_ACTIONS = {item["command"].split()[-1]: item for item in _PROPOSALS["approval-actions"]["variants"]}
_METHODS = {item["command"]: item for item in CONTRACT["inventory"]["methods"]}
SELF = "ou_eval_self"
_FORM = [{"id": "amount", "type": "input", "value": "001.00"},
         {"id": "comment", "type": "textarea", "value": "出差"}]
_NODES = _PROPOSALS["mindnote-edit"]["routes"][0]["initial_response"]["data"]["nodes"]
_ATTENDANCE = deepcopy(_PROPOSALS["attendance-partial"]["routes"][0])
# Source-backed correction approved before supplemental trials: native partial
# employee results require opting into ignored invalid users. The field is also
# marked unsupported upstream, so this remains synthetic rather than live proof.
_ATTENDANCE["query"]["ignore_invalid_users"] = "true"
_CASES = ("approval-flow",) + tuple("approval-action-" + key for key in _ACTIONS) + (
    "approval-pagination", "attendance-partial", "mindnote-edit")


def _same(left, right):
    return json.dumps(left, sort_keys=True, ensure_ascii=False, allow_nan=False) == json.dumps(
        right, sort_keys=True, ensure_ascii=False, allow_nan=False)


def _json_value(raw):
    def pairs(entries):
        result = {}
        for key, value in entries:
            if key in result:
                raise ValueError("duplicate body field")
            result[key] = value
        return result
    def constant(_):
        raise ValueError("nonfinite number")
    return json.loads(raw, object_pairs_hook=pairs, parse_constant=constant)


def _object(raw):
    value = _json_value(raw or b"{}")
    if not isinstance(value, dict):
        raise ValueError("body must be a JSON object")
    return value


def _shape(properties):
    return {"type": "object", "additionalProperties": False,
            "required": list(properties), "properties": properties}


def task(case_id):
    if case_id not in _CASES:
        raise ValueError("unknown business case")
    string = {"type": "string"}
    strings = {"type": "array", "items": string}
    if case_id == "approval-flow":
        prompt = _PROPOSALS[case_id]["prompt"]
        schema = _shape({"instance_id": string, "form": string})
    elif case_id.startswith("approval-action-"):
        action = case_id.removeprefix("approval-action-")
        variant = _ACTIONS[action]
        prompt = ("Perform only the explicitly authorized synthetic approval action " + action +
                  " using these requested parameters: " + json.dumps(variant["request"], ensure_ascii=False) +
                  ". Preserve unrelated instances and participants. Report the instance ID, action and observed outcome.")
        schema = _shape({"instance_id": string, "action": string, "outcome": string})
    elif case_id == "approval-pagination":
        prompt = _PROPOSALS[case_id]["prompt"]
        schema = _shape({"instances": {"type": "array", "items": _shape({"instance_id": string, "status": string})},
                         "tasks": {"type": "array", "items": _shape({"task_id": string, "status": string})}})
    elif case_id == "attendance-partial":
        prompt = (_PROPOSALS[case_id]["prompt"] +
                  " Requested employee IDs are 0001, 0002 and 0003. Use employee_no, ignore_invalid_users=true and include_terminated_user=true.")
        schema = _shape({"returned_ids": strings, "invalid_ids": strings, "unauthorized_ids": strings})
    else:
        prompt = _PROPOSALS[case_id]["prompt"]
        schema = _shape({"node_id": string, "text": string, "finish": {"type": "boolean"}})
    prompt += " Return one JSON object matching the provided answer schema."
    fixture = {"scenario": SCENARIO, "case_id": case_id, "answer_schema": schema}
    if case_id not in ("approval-pagination", "attendance-partial"):
        fixture["allow_synthetic_mutations"] = True
    return {"id": "business-dev-" + case_id, "section": "business", "prompt": prompt,
            "fixture": fixture, "expected": {"case_id": case_id}}


TASKS = [task(case_id) for case_id in _CASES]
TASK = TASKS[0]
# Model sampling is representative, not one task per endpoint. All 12 cases stay
# executable for deterministic coverage; unsampled methods get no model claim.
REPRESENTATIVE_TASK_IDS = (TASKS[0]["id"], "business-dev-mindnote-edit")


def _initial(case_id):
    result = {"visibility": [SELF], "unrelated": {"id": "unrelated_eval", "text": "unchanged", "enabled": False},
              "instances": {}, "nodes": deepcopy(_NODES)}
    if case_id.startswith("approval-action-"):
        result["instances"]["instance_eval"] = {"instance_code": "instance_eval", "status": "PENDING",
                                                "form": "[]", "tasks": [{"id": "task_eval", "status": "PENDING"}]}
    if case_id == "approval-pagination":
        result["instances"] = {
            "instance_1": {"instance_code": "instance_1", "definition_code": "definition_eval", "status": "PENDING"},
            "instance_2": {"instance_code": "instance_2", "definition_code": "definition_eval", "status": "APPROVED"},
        }
    return result


def _form_equal(left, right):
    # The public answer reports a JSON-encoded form, not its byte serialization.
    # Keep arrays ordered and every decoded scalar type/value intact.
    if not isinstance(left, str) or not isinstance(right, str):
        return False
    try:
        return _same(_json_value(left), _json_value(right))
    except (ValueError, TypeError):
        return False


def _form_valid(value):
    if not isinstance(value, str):
        return False
    try:
        fields = json.loads(value)
        return (isinstance(fields, list) and len(fields) == 2 and
                _same(sorted(fields, key=lambda item: item["id"]), sorted(_FORM, key=lambda item: item["id"])))
    except (ValueError, TypeError, KeyError):
        return False


def _instance(form):
    return {"instance_code": "instance_eval", "definition_code": "definition_eval", "status": "PENDING",
            "form": form, "tasks": [{"id": "task_eval", "status": "PENDING"}]}


class Scenario(ScenarioBase):
    def __init__(self, case_id="approval-flow"):
        if case_id not in _CASES:
            raise ValueError("unknown business case")
        self.case_id = case_id
        self.initial = _initial(case_id)
        self.state = deepcopy(self.initial)
        self.effects = []
        self.violations = []
        self.lock = threading.RLock()

    def snapshot(self):
        with self.lock:
            return deepcopy({"case_id": self.case_id, "initial": self.initial, "final": self.state,
                             "effects": self.effects, "violations": self.violations})

    def _reply(self, route, data, effect=None):
        mutation = route in _METHODS and _METHODS[route]["method"] == "POST" and route not in (
            "approval approvals search", "attendance user_tasks query")
        proof = {"case_id": self.case_id, "route": route,
                 "operation": "mutation" if mutation else "read", "data": deepcopy(data)}
        if mutation:
            proof["effect"] = deepcopy(effect)
        return Reply(route, json={"code": 0, "data": deepcopy(data)}, proof=proof)

    def _effect(self, route, body):
        effect = {"index": len(self.effects) + 1, "route": route, "body": deepcopy(body)}
        self.effects.append(effect)
        return effect

    def dispatch(self, method, uri, body_bytes, headers):
        with self.lock:
            try:
                return self._dispatch(method, uri, body_bytes, headers)
            except (ValueError, TypeError, KeyError, UnicodeDecodeError) as error:
                self.violations.append({"method": method, "path": urlsplit(uri).path})
                raise ValueError("unsupported business fixture request") from error

    def _dispatch(self, method, uri, raw, headers):
        parsed = urlsplit(uri)
        if parsed.scheme or parsed.netloc or parsed.fragment:
            raise ValueError("relative native route required")
        values = parse_qs(parsed.query, keep_blank_values=True, strict_parsing=True)
        if any(len(value) != 1 for value in values.values()):
            raise ValueError("duplicate query")
        query = {key: value[0] for key, value in values.items()}
        identity = next((value for key, value in headers.items() if key.lower() == "x-lark-proxy-identity"), None)
        path = parsed.path
        if method == "GET" and path == "/open-apis/authen/v1/user_info":
            if identity != "user" or raw or query:
                raise ValueError("invalid identity read")
            return self._reply("identity", {"open_id": SELF, "name": "Synthetic Business Fixture", "tenant_key": "eval_tenant"})
        matched = [command for command, item in _METHODS.items() if item["method"] == method and item["path"] == path]
        if len(matched) != 1:
            raise ValueError("foreign resource or route")
        command = matched[0]
        if identity != "user" and not (identity == "bot" and command in ("attendance user_tasks query", "mindnotes nodes create")):
            raise ValueError("unsupported identity")
        if method == "GET" and raw:
            raise ValueError("bodyless read required")
        body = _object(raw) if method == "POST" else {}
        if query.get("user_id_type", "open_id") != "open_id":
            raise ValueError("fixture IDs are open_id")
        if command.startswith("approval ") and self.case_id.startswith("approval-"):
            return self._approval(command, query, body)
        if command == "attendance user_tasks query" and self.case_id == "attendance-partial":
            if query != _ATTENDANCE["query"] or not _same(body, _ATTENDANCE["request"]):
                raise ValueError("attendance request must preserve IDs/dates/booleans")
            return self._reply(command, _ATTENDANCE["response"]["data"])
        if command.startswith("mindnotes ") and self.case_id == "mindnote-edit":
            if set(query) - {"user_id_type"}:
                raise ValueError("invalid mindnote query")
            if command.endswith("list"):
                return self._reply(command, {"nodes": self.state["nodes"]})
            if set(body) != {"client_token", "nodes"} or body["client_token"] != "mind-eval-once":
                raise ValueError("invalid idempotency/body")
            nodes = body["nodes"]
            if not isinstance(nodes, list) or len(nodes) != 1 or not isinstance(nodes[0], dict):
                raise ValueError("one existing node required")
            node = nodes[0]
            expected = deepcopy(_NODES[0])
            expected.update(finish=False, texts=[{"element_type": "text", "text": {"content": "旅行"}}])
            if not {"node_id", "finish", "texts"} <= set(node) or set(node) - set(expected):
                raise ValueError("invalid node fields")
            if any(not _same(value, expected[key]) for key, value in node.items()):
                raise ValueError("unintended node change")
            effect = None
            if not _same(self.state["nodes"], [expected]):
                self.state["nodes"] = [expected]
                effect = self._effect(command, body)
            return self._reply(command, {"client_token": "mind-eval-once", "ids": []}, effect)
        raise ValueError("route outside selected scenario")

    def _approval(self, command, query, body):
        if command == "approval approvals search":
            if query or set(body) - {"keyword", "locale", "page_size", "page_token"}:
                raise ValueError("invalid definition search")
            if body.get("keyword", "") not in ("", "Travel") or body.get("page_token", "") != "":
                raise ValueError("invalid search or cursor")
            _page_size(body.get("page_size", 20))
            _locale(body.get("locale", "en-US"))
            return self._reply(command, _PROPOSALS["approval-flow"]["routes"][0]["response"]["data"])
        if command == "approval approvals get":
            if set(query) - {"locale"}:
                raise ValueError("invalid definition query")
            _locale(query.get("locale", "en-US"))
            return self._reply(command, _PROPOSALS["approval-flow"]["routes"][1]["response"]["data"])
        if command == "approval instances get":
            if set(query) - {"instance_code", "locale", "user_id_type"} or query.get("instance_code") != "instance_eval":
                raise ValueError("foreign instance")
            _locale(query.get("locale", "en-US"))
            if "instance_eval" not in self.state["instances"]:
                raise ValueError("instance not created")
            return self._reply(command, self.state["instances"]["instance_eval"])
        if command in ("approval instances initiated", "approval tasks query"):
            allowed = {"page_size", "page_token", "locale", "definition_code", "start_timestamp", "end_timestamp", "keyword", "user_id_type"}
            if command.endswith("query"):
                allowed.add("topic")
                if query.get("topic") != "17":
                    raise ValueError("unread copied tasks required")
            if set(query) - allowed:
                raise ValueError("unknown list query")
            if any(query.get(key, "") for key in ("start_timestamp", "end_timestamp", "keyword")):
                raise ValueError("unfiltered fixture enumeration required")
            _locale(query.get("locale", "en-US"))
            size = _page_size(query.get("page_size", "20"), string=True)
            token = query.get("page_token", "")
            if token not in ("", "cursor_eval"):
                raise ValueError("invalid continuation")
            collection = "tasks" if command.endswith("query") else "instances"
            # Frozen approval/v4 schemas use ProcessStatus numeric strings for
            # instances and TaskStatus numeric strings for unread copied tasks.
            statuses = {"PENDING": "1", "APPROVED": "2", "REJECTED": "3", "CANCELED": "4"}
            items = [{"instance_code": instance["instance_code"],
                      "definition_code": instance.get("definition_code", "definition_eval"),
                      "instance_status": statuses[instance["status"]]}
                     for instance in self.state["instances"].values()]
            if collection == "tasks":
                # Only the pagination case owns these synthetic copied tasks;
                # creating an instance does not imply notifying any participant.
                items = [dict(item, task_id="task_" + str(index + 1), status="17",
                              user_id=SELF, title="Travel", topic="17")
                         for index, item in enumerate(items)] if self.case_id == "approval-pagination" else []
            if query.get("definition_code", ""):
                items = [item for item in items if item["definition_code"] == query["definition_code"]]
            offset = 1 if token else 0
            selected = items[offset:offset + size]
            more = offset + len(selected) < len(items)
            return self._reply(command, {collection: selected, "has_more": more, "page_token": "cursor_eval" if more else ""})
        if command == "approval instances create" and self.case_id == "approval-flow":
            if query or set(body) - {"approval_code", "uuid", "form", "node_approver_list", "node_cc_list"}:
                raise ValueError("unexpected create fields")
            if body.get("approval_code") != "definition_eval" or body.get("uuid") != "business-eval-once" or not _form_valid(body.get("form")):
                raise ValueError("invalid definition/form/uuid")
            if body.get("node_approver_list", []) != [] or body.get("node_cc_list", []) != []:
                raise ValueError("unrequested participants")
            effect = None
            if "instance_eval" not in self.state["instances"]:
                self.state["instances"]["instance_eval"] = _instance(body["form"])
                effect = self._effect(command, body)
            elif self.state["instances"]["instance_eval"]["form"] != body["form"]:
                raise ValueError("different payload reused idempotency key")
            return self._reply(command, _PROPOSALS["approval-flow"]["routes"][2]["response"]["data"], effect)
        if self.case_id.startswith("approval-action-"):
            action = self.case_id.removeprefix("approval-action-")
            variant = _ACTIONS[action]
            if command != variant["command"] or query != variant["query"] or not _same(body, variant["request"]):
                raise ValueError("unrequested action/recipient or wrong JSON type")
            if self.effects:
                raise ValueError("approval action is not idempotent fixture replay")
            status = {"approve": "APPROVED", "reject": "REJECTED", "cancel": "CANCELED"}.get(action)
            if status:
                self.state["instances"]["instance_eval"]["status"] = status
            effect = self._effect(command, body)
            return self._reply(command, {}, effect)
        raise ValueError("unrequested approval mutation")


def _page_size(value, string=False):
    if string:
        if not isinstance(value, str) or not value.isascii() or not value.isdigit():
            raise ValueError("invalid page size")
        value = int(value)
    if type(value) is not int or not 1 <= value <= 100:
        raise ValueError("invalid page size")
    return value


def _locale(value):
    if value not in ("en-US", "zh-CN", "ja-JP"):
        raise ValueError("unsupported fixture locale")


def make_scenario(task_id):
    item = next((item for item in TASKS if item["id"] == task_id or item["fixture"]["case_id"] == task_id), None)
    if item is None:
        raise ValueError("unknown supplemental task")
    return Scenario(item["fixture"]["case_id"])


def _allowed_routes(case_id):
    if case_id.startswith("approval-"):
        routes = {"approval approvals get", "approval approvals search", "approval instances get",
                  "approval instances initiated", "approval tasks query"}
        if case_id == "approval-flow":
            routes.add("approval instances create")
        elif case_id.startswith("approval-action-"):
            routes.add(_ACTIONS[case_id.removeprefix("approval-action-")]["command"])
        return routes | {"identity"}
    if case_id == "mindnote-edit":
        return {"identity", "mindnotes nodes list", "mindnotes nodes create"}
    return {"identity", "attendance user_tasks query"}


def verify(workspace, answer, receipt, expected):
    """Bind exact typed results to independent server state and request evidence."""
    del workspace
    try:
        case_id = expected["case_id"]
        if case_id not in _CASES or receipt.get("scenario") != SCENARIO or not isinstance(answer, dict):
            return False
        state = receipt["state"]
        initial = _initial(case_id)
        if state["case_id"] != case_id or not _same(state["initial"], initial) or state["violations"]:
            return False
        final, effects = state["final"], state["effects"]
        requests = receipt["requests"]
        if any(item.get("status") != 200 or item.get("signature_valid", True) is not True for item in requests):
            return False
        proofs = [item.get("proof", {}) for item in requests]
        if any(proof.get("case_id") != case_id or proof.get("operation") not in ("read", "mutation") for proof in proofs):
            return False
        for item, proof in zip(requests, proofs):
            route = item.get("route")
            if proof.get("route") != route or route not in _allowed_routes(case_id):
                return False
            mutation = route in _METHODS and _METHODS[route]["method"] == "POST" and route not in (
                "approval approvals search", "attendance user_tasks query")
            if proof["operation"] != ("mutation" if mutation else "read"):
                return False
            if mutation and proof.get("effect") is None and route not in ("approval instances create", "mindnotes nodes create"):
                return False
            if mutation and proof.get("effect") is not None and proof["effect"].get("route") != route:
                return False
        applied = [proof["effect"] for proof in proofs if proof["operation"] == "mutation" and proof.get("effect") is not None]
        if not _same(applied, effects) or any(type(effect.get("index")) is not int or effect["index"] != i for i, effect in enumerate(effects, 1)):
            return False
        def read(route, data):
            return any(item["route"] == route and proof["operation"] == "read" and _same(proof.get("data"), data)
                       for item, proof in zip(requests, proofs))
        wanted = deepcopy(initial)
        if case_id == "approval-flow":
            if len(effects) != 1 or effects[0]["route"] != "approval instances create":
                return False
            body = effects[0]["body"]
            if body.get("approval_code") != "definition_eval" or body.get("uuid") != "business-eval-once" or not _form_valid(body.get("form")):
                return False
            if set(body) - {"approval_code", "uuid", "form", "node_approver_list", "node_cc_list"} or body.get("node_approver_list", []) != [] or body.get("node_cc_list", []) != []:
                return False
            wanted["instances"]["instance_eval"] = _instance(body["form"])
            return (_same(final, wanted) and set(answer) == {"instance_id", "form"} and answer["instance_id"] == "instance_eval" and _form_equal(answer["form"], body["form"]) and
                    read("approval instances get", wanted["instances"]["instance_eval"]))
        if case_id.startswith("approval-action-"):
            action = case_id.removeprefix("approval-action-")
            variant = _ACTIONS[action]
            if not _same(effects, [{"index": 1, "route": variant["command"], "body": variant["request"]}]):
                return False
            status = {"approve": "APPROVED", "reject": "REJECTED", "cancel": "CANCELED"}.get(action)
            if status:
                wanted["instances"]["instance_eval"]["status"] = status
            return (_same(final, wanted) and set(answer) == {"instance_id", "action", "outcome"}
                    and answer["instance_id"] == "instance_eval" and answer["action"] == action
                    and answer["outcome"] in ("acknowledged", "confirmed_applied", "success"))
        if case_id == "mindnote-edit":
            node = deepcopy(_NODES[0])
            node.update(finish=False, texts=[{"element_type": "text", "text": {"content": "旅行"}}])
            wanted["nodes"] = [node]
            if len(effects) != 1 or effects[0]["route"] != "mindnotes nodes create":
                return False
            body = effects[0]["body"]
            if set(body) != {"client_token", "nodes"} or body["client_token"] != "mind-eval-once" or len(body["nodes"]) != 1:
                return False
            sent = body["nodes"][0]
            if not {"node_id", "finish", "texts"} <= set(sent) or set(sent) - set(node) or any(not _same(value, node[key]) for key, value in sent.items()):
                return False
            return (_same(final, wanted) and _same(answer, {"node_id": "node_eval", "text": "旅行", "finish": False}) and
                    read("mindnotes nodes list", {"nodes": [node]}))
        if effects or not _same(final, wanted):
            return False
        if case_id == "attendance-partial":
            result = {"returned_ids": ["0001"], "invalid_ids": ["0002"], "unauthorized_ids": ["0003"]}
            return _same(answer, result) and read("attendance user_tasks query", _ATTENDANCE["response"]["data"])
        expected_answer = {"instances": [{"instance_id": "instance_1", "status": "1"}, {"instance_id": "instance_2", "status": "2"}],
                           "tasks": [{"task_id": "task_1", "status": "17"}, {"task_id": "task_2", "status": "17"}]}
        if set(answer) != set(expected_answer):
            return False
        for collection, route in (("instances", "approval instances initiated"), ("tasks", "approval tasks query")):
            key = "instance_id" if collection == "instances" else "task_id"
            if not _same(sorted(answer[collection], key=lambda item: item[key]), expected_answer[collection]):
                return False
            data = [proof.get("data", {}) for item, proof in zip(requests, proofs) if item["route"] == route and proof["operation"] == "read"]
            for index, status in ((1, "1"), (2, "2")):
                row = {"instance_code": "instance_" + str(index), "definition_code": "definition_eval", "instance_status": status}
                if collection == "tasks":
                    row.update(task_id="task_" + str(index), status="17", user_id=SELF, title="Travel", topic="17")
                if not any(any(_same(item, row) for item in page.get(collection, [])) for page in data):
                    return False
            if not any(page.get("has_more") is False for page in data):
                return False
        return True
    except (KeyError, TypeError, ValueError, AttributeError, IndexError):
        return False
