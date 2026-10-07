# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Offline Task scenario transitions and independent outcome checks."""
from copy import deepcopy
import importlib.util
import json
from pathlib import Path
import sys
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "evaluation"))
SPEC = importlib.util.spec_from_file_location("task_scenario", ROOT / "evaluation/scenarios/task.py")
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class TaskScenarioTests(unittest.TestCase):
    def setUp(self):
        self.scenario = MODULE.Scenario()
        self.requests = []
        self.headers = {"X-Lark-Proxy-Identity": "user"}
        self.answer = deepcopy(MODULE.TASK["expected"]["answer"])

    def call(self, method, path, body=None, headers=None):
        reply = self.scenario.dispatch(method, path, b"" if body is None else json.dumps(body).encode(),
                                       self.headers if headers is None else headers)
        self.requests.append({"route": reply.route, "status": reply.status,
                              "returned_ids": [], "proof": deepcopy(reply.proof)})
        return reply

    def complete(self, guid=MODULE.TARGET, value="1893546000000"):
        return self.call("PATCH", MODULE._BASE + "/" + guid + "?user_id_type=open_id",
                         {"task": {"completed_at": value}, "update_fields": ["completed_at"]})

    def receipt(self):
        return {"scenario": MODULE.SCENARIO, "requests": deepcopy(self.requests),
                "state": self.scenario.snapshot()}

    def verify(self, answer=None, receipt=None):
        return MODULE.verify(None, self.answer if answer is None else answer,
                             self.receipt() if receipt is None else receipt, MODULE.TASK["expected"])

    def test_public_fixture_has_only_types_and_no_hidden_state(self):
        fixture = MODULE.TASK["fixture"]
        self.assertEqual(set(fixture), {"scenario", "answer_schema", "allow_synthetic_mutations"})
        serialized = json.dumps(fixture)
        for forbidden in (MODULE.TARGET, MODULE.SELF, "const", "enum", "tasks", "assignee_ids\": ["):
            self.assertNotIn(forbidden, serialized)
        self.assertEqual(fixture["answer_schema"]["properties"]["completed"], {"type": "boolean"})

    def test_read_patch_authoritative_response_succeeds_without_forced_get(self):
        before = self.scenario.snapshot()
        read = self.call("GET", MODULE._BASE + "/" + MODULE.TARGET)
        self.assertEqual(read.json["data"]["task"]["completed_at"], "0")
        response = self.complete()
        self.assertEqual(response.json["data"]["task"]["status"], "done")
        self.assertTrue(self.verify())
        self.assertEqual(self.scenario.tasks[MODULE.TARGET]["members"], before["tasks"][MODULE.TARGET]["members"])
        for guid in set(self.scenario.tasks) - {MODULE.TARGET}:
            self.assertEqual(self.scenario.tasks[guid], before["tasks"][guid])

    def test_direct_patch_and_followup_read_are_equivalent(self):
        self.complete()
        self.call("GET", MODULE._BASE + "/" + MODULE.TARGET + "?user_id_type=open_id")
        self.assertTrue(self.verify())
        response = self.complete()
        self.assertFalse(response.proof["changed"])
        self.assertEqual(len(self.scenario.notifications), 1)
        self.assertEqual(len(self.scenario.transitions), 1)
        self.assertTrue(self.verify())

    def test_pagination_and_explicit_false_filter_follow_current_state(self):
        page = self.call("GET", MODULE._BASE + "?type=my_tasks&page_size=1&completed=false")
        self.assertTrue(page.json["data"]["has_more"])
        self.assertEqual(page.json["data"]["page_token"], "task-page-1")
        next_page = self.call("GET", MODULE._BASE + "?page_size=1&page_token=task-page-1&completed=false")
        self.assertFalse(next_page.json["data"]["has_more"])
        self.assertNotEqual(page.json["data"]["items"][0]["guid"], next_page.json["data"]["items"][0]["guid"])
        self.complete()
        complete = self.call("GET", MODULE._BASE + "?completed=true")
        self.assertEqual([item["guid"] for item in complete.json["data"]["items"]], [MODULE.TARGET])
        incomplete = self.call("GET", MODULE._BASE + "?completed=false")
        self.assertNotIn(MODULE.TARGET, [item["guid"] for item in incomplete.json["data"]["items"]])
        self.assertEqual(len(self.call("GET", MODULE._BASE).json["data"]["items"]), 2)
        self.assertTrue(self.verify())

    def test_answer_claim_without_state_or_response_proof_fails(self):
        self.assertFalse(self.verify())
        self.complete()
        forged = self.receipt()
        forged["requests"] = []
        self.assertFalse(self.verify(receipt=forged))
        forged = self.receipt()
        forged["state"]["transitions"] = []
        self.assertFalse(self.verify(receipt=forged))
        forged = self.receipt()
        forged["requests"][0]["proof"]["tasks"] = []
        self.assertFalse(self.verify(receipt=forged))

    def test_wrong_answers_and_native_type_tampering_fail(self):
        self.complete()
        for field, value in (("completed", 1), ("task_id", "other"), ("assignee_ids", [])):
            answer = deepcopy(self.answer)
            answer[field] = value
            self.assertFalse(self.verify(answer=answer))
        for field, value in (("completed_at", 1893546000000), ("mode", True), ("description", None)):
            receipt = self.receipt()
            receipt["state"]["tasks"][MODULE.TARGET][field] = value
            self.assertFalse(self.verify(receipt=receipt))
        receipt = self.receipt()
        receipt["state"]["tasks"][MODULE.TARGET]["members"][0]["role"] = "follower"
        self.assertFalse(self.verify(receipt=receipt))

    def test_decoy_changes_and_restore_do_not_hide_unintended_effects(self):
        other = next(guid for guid in self.scenario.tasks if guid != MODULE.TARGET)
        self.complete(other)
        self.complete(other, "0")
        self.complete()
        self.assertFalse(self.verify())

    def test_content_changes_restored_still_fail_preservation(self):
        path = MODULE._BASE + "/" + MODULE.TARGET
        original = self.scenario.tasks[MODULE.TARGET]["description"]
        for value in ("", original):
            self.call("PATCH", path, {"task": {"description": value}, "update_fields": ["description"]})
        self.complete()
        self.assertFalse(self.verify())

    def test_foreign_and_assignment_write_attempts_are_not_ignored(self):
        for path, payload in (
            (MODULE._BASE + "/" + MODULE.TARGET + "/add_members", {"members": [{"id": "ou_other", "role": "assignee"}]}),
            (MODULE._BASE + "/" + MODULE.TARGET, {"task": {"members": []}, "update_fields": ["members"]}),
        ):
            with self.assertRaises(ValueError):
                self.call("PATCH" if path.endswith(MODULE.TARGET) else "POST", path, payload)
        missing = self.complete("unrelated-guid")
        self.assertEqual(missing.json["code"], 1470404)
        self.complete()
        self.assertFalse(self.verify())

    def test_invalid_value_cannot_mutate_and_can_be_corrected(self):
        for value in (True, 1, None, "", "-1", "1e3"):
            result = self.complete(value=value)
            self.assertEqual(result.json["code"], 1470400)
            self.assertEqual(self.scenario.tasks[MODULE.TARGET]["completed_at"], "0")
        self.complete()
        self.assertTrue(self.verify())

    def test_identity_and_query_boundaries(self):
        for uri in (MODULE._BASE + "?completed=bad", MODULE._BASE + "?page_size=0",
                    MODULE._BASE + "?user_id_type=union_id", MODULE._BASE + "?page_token=unknown",
                    MODULE._BASE + "?completed=false&completed=true"):
            with self.assertRaises(ValueError):
                self.call("GET", uri)
        with self.assertRaises(ValueError):
            self.call("GET", MODULE._BASE, headers={"X-Lark-Proxy-Identity": "bot"})
        self.complete()
        self.assertTrue(self.verify())

    def test_notifications_visibility_and_proof_tampering_fail(self):
        self.complete()
        for key, value in (("notifications", []), ("notifications", [{"type": "task_completed", "task_id": MODULE.TARGET, "recipient_id": "ou_other"}]),
                           ("visibility", {MODULE.TARGET: ["organization"]})):
            receipt = self.receipt()
            receipt["state"][key] = value
            self.assertFalse(self.verify(receipt=receipt))
        receipt = self.receipt()
        receipt["requests"][0]["proof"]["revision"] = True
        self.assertFalse(self.verify(receipt=receipt))
        receipt = self.receipt()
        receipt["requests"][0]["status"] = 403
        self.assertFalse(self.verify(receipt=receipt))
        receipt = self.receipt()
        receipt["state"]["transitions"][0]["revision"] = True
        self.assertFalse(self.verify(receipt=receipt))
        receipt = self.receipt()
        receipt["requests"][0]["proof"]["operation"] = "read"
        self.assertFalse(self.verify(receipt=receipt))
        self.assertFalse(self.verify(receipt={}))


if __name__ == "__main__":
    unittest.main()
