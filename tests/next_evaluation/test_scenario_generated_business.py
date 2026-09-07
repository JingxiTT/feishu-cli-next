"""Synthetic generated-business adapters and independent verifier contracts."""
from copy import deepcopy
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import socket
import subprocess
import sys
import tempfile
import unittest
from urllib.parse import urlencode

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "evaluation"))
SPEC = importlib.util.spec_from_file_location("business_scenario", ROOT / "evaluation/scenarios/generated_business.py")
M = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(M)


def steps(case):
    if case == "approval-flow":
        body = {"approval_code": "definition_eval", "uuid": "business-eval-once",
                "form": json.dumps(M._FORM, ensure_ascii=False)}
        return [("approval approvals search", {}, {"keyword": "Travel", "page_size": 1}),
                ("approval approvals get", {"approval_code": "definition_eval"}, None),
                ("approval instances create", {}, body),
                ("approval instances get", {"instance_code": "instance_eval"}, None)]
    if case.startswith("approval-action-"):
        v = M._ACTIONS[case.removeprefix("approval-action-")]
        return [(v["command"], v["query"], v["request"])]
    if case == "approval-pagination":
        return [(command, {**extra, "page_size": 1, **({"page_token": "cursor_eval"} if page else {})}, None)
                for command, extra in (("approval instances initiated", {}), ("approval tasks query", {"topic": "17"}))
                for page in (False, True)]
    if case == "attendance-partial":
        return [("attendance user_tasks query", M._ATTENDANCE["query"], M._ATTENDANCE["request"])]
    node = {"node_id": "node_eval", "finish": False,
            "texts": [{"element_type": "text", "text": {"content": "旅行"}}]}
    return [("mindnotes nodes list", {"mindnote_id": "mind_eval"}, None),
            ("mindnotes nodes create", {"mindnote_id": "mind_eval"}, {"client_token": "mind-eval-once", "nodes": [node]}),
            ("mindnotes nodes list", {"mindnote_id": "mind_eval"}, None)]


def answer(case, scenario):
    if case == "approval-flow":
        return {"instance_id": "instance_eval", "form": scenario.state["instances"]["instance_eval"]["form"]}
    if case.startswith("approval-action-"):
        return {"instance_id": "instance_eval", "action": case.removeprefix("approval-action-"), "outcome": "acknowledged"}
    if case == "approval-pagination":
        return {"instances": [{"instance_id": "instance_1", "status": "PENDING"}, {"instance_id": "instance_2", "status": "APPROVED"}],
                "tasks": [{"task_id": "task_1", "status": "PENDING"}, {"task_id": "task_2", "status": "APPROVED"}]}
    if case == "attendance-partial":
        return {"returned_ids": ["0001"], "invalid_ids": ["0002"], "unauthorized_ids": ["0003"]}
    return {"node_id": "node_eval", "text": "旅行", "finish": False}


def uri(command, params):
    route = M._METHODS[command]
    query = dict(params)
    # Frozen path parameters have already been bound in the contract's finite
    # fixture paths; the real CLI performs the same substitution itself.
    query.pop("approval_code", None)
    query.pop("mindnote_id", None)
    return route["path"] + ("?" + urlencode(query) if query else "")


class BusinessScenarioTests(unittest.TestCase):
    def run_case(self, case, custom_steps=None):
        scenario = M.Scenario(case)
        requests = []
        for command, params, body in steps(case) if custom_steps is None else custom_steps:
            reply = scenario.dispatch(M._METHODS[command]["method"], uri(command, params),
                                      b"" if body is None else json.dumps(body, ensure_ascii=False).encode(),
                                      {"X-Lark-Proxy-Identity": "user"})
            requests.append({"route": reply.route, "status": reply.status,
                             "returned_ids": [], "proof": deepcopy(reply.proof), "signature_valid": True})
        return scenario, {"scenario": M.SCENARIO, "requests": requests, "state": scenario.snapshot()}

    def test_twelve_cases_cover_all_seventeen_methods_without_model_sampling_all(self):
        self.assertEqual(len(M.TASKS), 12)
        self.assertEqual(len(set(t["id"] for t in M.TASKS)), 12)
        covered = set()
        for task in M.TASKS:
            case = task["fixture"]["case_id"]
            with self.subTest(case=case):
                scenario, receipt = self.run_case(case)
                self.assertTrue(M.verify(None, answer(case, scenario), receipt, task["expected"]))
                covered.update(item["route"] for item in receipt["requests"])
                self.assertIsInstance(M.make_scenario(task["id"]), M.Scenario)
        self.assertEqual(covered, set(M._METHODS))
        self.assertEqual(len(covered), 17)
        self.assertEqual(len(M.REPRESENTATIVE_TASK_IDS), 2)
        self.assertTrue(set(M.REPRESENTATIVE_TASK_IDS) <= {t["id"] for t in M.TASKS})

    def test_public_fixture_is_type_only_and_binds_case(self):
        for task in M.TASKS:
            fixture = task["fixture"]
            self.assertLessEqual(set(fixture), {"scenario", "case_id", "answer_schema", "allow_synthetic_mutations"})
            schema = json.dumps(fixture["answer_schema"])
            for value in ("const", "enum", "default", "example", "instance_eval", "node_eval", "0001", "旅行", "PENDING"):
                self.assertNotIn(value, schema)
            self.assertEqual(fixture["scenario"], "generated-business-v1")
            self.assertEqual(fixture["case_id"], task["expected"]["case_id"])
            self.assertEqual(task["section"], "business")
        with self.assertRaises(ValueError):
            M.make_scenario("business-dev-foreign")

    def test_claims_without_observation_or_effect_proof_fail_all_cases(self):
        for task in M.TASKS:
            case = task["expected"]["case_id"]
            with self.subTest(case=case):
                scenario, receipt = self.run_case(case)
                good = answer(case, scenario)
                for mutation in ("no requests", "wrong case", "changed unrelated", "false signature", "wrong route", "changed type"):
                    forged = deepcopy(receipt)
                    if mutation == "no requests": forged["requests"] = []
                    elif mutation == "wrong case": forged["state"]["case_id"] = "foreign"
                    elif mutation == "changed unrelated": forged["state"]["final"]["unrelated"]["text"] = "changed"
                    elif mutation == "false signature": forged["requests"][0]["signature_valid"] = False
                    elif mutation == "wrong route": forged["requests"][0]["route"] = "invented"
                    else: forged["state"]["final"]["unrelated"]["enabled"] = 0
                    self.assertFalse(M.verify(None, good, forged, task["expected"]), mutation)
                forged = deepcopy(good)
                forged[next(iter(forged))] = None
                self.assertFalse(M.verify(None, forged, receipt, task["expected"]))
                self.assertFalse(M.verify(None, good, receipt, {"case_id": "other"}))

    def test_flow_requires_actual_readback_and_preserves_form_string_types(self):
        case = "approval-flow"
        scenario, receipt = self.run_case(case)
        self.assertTrue(M._form_valid(answer(case, scenario)["form"]))
        missing = deepcopy(receipt)
        missing["requests"] = missing["requests"][:-1]
        self.assertFalse(M.verify(None, answer(case, scenario), missing, {"case_id": case}))
        forged = deepcopy(receipt)
        forged["state"]["effects"][0]["body"]["form"] = json.dumps([
            {"id": "amount", "type": "input", "value": 1.0}, M._FORM[1]])
        self.assertFalse(M.verify(None, answer(case, scenario), forged, {"case_id": case}))
        reordered = steps(case)
        reordered[2][2]["form"] = json.dumps(list(reversed(M._FORM)))
        reordered.insert(3, deepcopy(reordered[2]))
        other, other_receipt = self.run_case(case, reordered)
        self.assertEqual(len(other.effects), 1)
        self.assertTrue(M.verify(None, answer(case, other), other_receipt, {"case_id": case}))

    def test_pagination_supports_single_page_and_rejects_hidden_truncation(self):
        case = "approval-pagination"
        scenario, receipt = self.run_case(case)
        for index in (1, 3):
            truncated = deepcopy(receipt)
            del truncated["requests"][index]
            self.assertFalse(M.verify(None, answer(case, scenario), truncated, {"case_id": case}))
        single = [("approval instances initiated", {}, None), ("approval tasks query", {"topic": "17"}, None)]
        scenario, receipt = self.run_case(case, single)
        reversed_answer = answer(case, scenario)
        reversed_answer["instances"].reverse()
        self.assertTrue(M.verify(None, reversed_answer, receipt, {"case_id": case}))

    def test_actions_record_exact_intended_effect_and_reject_other_people_or_replay(self):
        for action, variant in M._ACTIONS.items():
            case = "approval-action-" + action
            with self.subTest(action=action):
                scenario, receipt = self.run_case(case)
                with self.assertRaises(ValueError):
                    scenario.dispatch("POST", uri(variant["command"], variant["query"]),
                                      json.dumps(variant["request"]).encode(), {"X-Lark-Proxy-Identity": "user"})
                receipt["state"] = scenario.snapshot()
                self.assertFalse(M.verify(None, answer(case, scenario), receipt, {"case_id": case}))
                self.assertEqual(len(scenario.effects), 1)
                other = deepcopy(variant["request"])
                other["instance_code"] = "instance_unrelated"
                fresh = M.Scenario(case)
                with self.assertRaises(ValueError):
                    fresh.dispatch("POST", uri(variant["command"], variant["query"]), json.dumps(other).encode(),
                                   {"X-Lark-Proxy-Identity": "user"})
                self.assertEqual(fresh.effects, [])

    def test_mindnote_merge_and_readback_preserve_unrequested_fields_and_types(self):
        case = "mindnote-edit"
        scenario, receipt = self.run_case(case)
        self.assertTrue(M.verify(None, answer(case, scenario), receipt, {"case_id": case}))
        for key in ("notes", "images", "highlight", "parent_id"):
            self.assertEqual(scenario.state["nodes"][0][key], M._NODES[0][key])
        self.assertFalse(M.verify(None, {"node_id": "node_eval", "text": "旅行", "finish": 0}, receipt, {"case_id": case}))
        missing = deepcopy(receipt)
        missing["requests"] = missing["requests"][:-1]
        self.assertFalse(M.verify(None, answer(case, scenario), missing, {"case_id": case}))
        full = steps(case)
        full[1][2]["nodes"][0] = deepcopy(scenario.state["nodes"][0])
        fresh, valid = self.run_case(case, full)
        self.assertTrue(M.verify(None, answer(case, fresh), valid, {"case_id": case}))

    def test_attendance_is_native_post_read_with_full_partial_coverage_evidence(self):
        case = "attendance-partial"
        scenario, receipt = self.run_case(case)
        self.assertEqual(scenario.effects, [])
        self.assertEqual(receipt["requests"][0]["proof"]["operation"], "read")
        for key in ("invalid_ids", "unauthorized_ids"):
            wrong = answer(case, scenario)
            wrong[key] = []
            self.assertFalse(M.verify(None, wrong, receipt, {"case_id": case}))
        for key, value in (("check_date_from", True), ("need_overtime_result", 0), ("user_ids", [1, 2, 3])):
            body = deepcopy(M._ATTENDANCE["request"])
            body[key] = value
            fresh = M.Scenario(case)
            with self.assertRaises(ValueError):
                fresh.dispatch("POST", uri("attendance user_tasks query", M._ATTENDANCE["query"]),
                               json.dumps(body).encode(), {"X-Lark-Proxy-Identity": "user"})
            self.assertEqual(fresh.effects, [])

    def test_cross_case_mutation_proof_cannot_hide_as_a_noop(self):
        case = "mindnote-edit"
        scenario, receipt = self.run_case(case)
        forged = deepcopy(receipt)
        forged["requests"].append({"route": "approval instances create", "status": 200,
            "proof": {"case_id": case, "route": "approval instances create", "operation": "mutation",
                      "effect": None, "data": {}}, "signature_valid": True})
        self.assertFalse(M.verify(None, answer(case, scenario), forged, {"case_id": case}))

    def test_invalid_json_foreign_paths_queries_and_identity_are_rejected_and_retained(self):
        cases = [("POST", "/open-apis/approval/v4/instances/initiate", b'{"uuid":1,"uuid":2}', "user"),
                 ("POST", "/open-apis/approval/v4/instances/initiate", b'{"form":NaN}', "user"),
                 ("POST", "/open-apis/approval/v4/instances/initiate", b'{}', "bot"),
                 ("GET", "/open-apis/approval/v4/approvals/foreign/detail", b'', "user"),
                 ("GET", "/open-apis/approval/v4/approvals/definition_eval/detail?locale=en-US&locale=ja-JP", b'', "user"),
                 ("GET", "/open-apis/approval/v4/approvals/definition_eval/detail?unexpected=1", b'', "user")]
        for method, path, body, identity in cases:
            scenario = M.Scenario()
            with self.subTest(path=path), self.assertRaises(ValueError):
                scenario.dispatch(method, path, body, {"X-Lark-Proxy-Identity": identity})
            self.assertEqual(len(scenario.violations), 1)
            self.assertEqual(scenario.effects, [])
            self.assertEqual(scenario.state, scenario.initial)


@unittest.skipUnless(os.environ.get("EVAL_GENERATED_BUSINESS_BINARIES"),
                     "set EVAL_GENERATED_BUSINESS_BINARIES=1 for existing clean-archive manifest binaries")
class BusinessBinaryTests(unittest.TestCase):
    def test_both_clean_archive_binaries_execute_all_seventeen_native_methods(self):
        from section_fixture import SectionFixture
        sys.path.insert(0, str(ROOT / "scripts"))
        from next_test import isolated_env
        paths = json.loads((ROOT / ".tmp/evaluation-paths.json").read_text())
        for arm in ("official", "next"):
            binary = Path(paths[arm + "_cli"])
            self.assertTrue(binary.is_absolute() and binary.is_file())
            self.assertEqual(len(paths[arm + "_revision"]), 40)
            if arm == "next":
                self.assertEqual(hashlib.sha256(binary.read_bytes()).hexdigest(), paths["next_binary_sha256"])
            covered = set()
            for item in M.TASKS:
                case = item["fixture"]["case_id"]
                scenario = M.Scenario(case)
                with self.subTest(arm=arm, case=case), tempfile.TemporaryDirectory(prefix="business-native-") as directory, socket.socket() as dead_proxy:
                    dead_proxy.bind(("127.0.0.1", 0))
                    env = isolated_env(directory, os.environ, "http://127.0.0.1:" + str(dead_proxy.getsockname()[1]))
                    workspace = Path(directory) / "work"
                    workspace.mkdir()
                    with SectionFixture(M.SCENARIO, scenario.dispatch, scenario.snapshot) as server:
                        env.update(server.environment())
                        for command, params, body in steps(case):
                            argv = [str(binary), *command.split(), "--as", "user"]
                            for key, value in params.items():
                                encoded = str(value).lower() if type(value) is bool else str(value)
                                argv.append("--" + key.replace("_", "-") + "=" + encoded)
                            if body is not None:
                                argv += ["--data", json.dumps(body, ensure_ascii=False)]
                            if command.startswith("approval ") and command not in ("approval approvals search", "approval approvals get", "approval instances get", "approval instances initiated", "approval tasks query"):
                                argv.append("--yes")
                            result = subprocess.run(argv, cwd=workspace, env=env, capture_output=True, text=True, timeout=30)
                            expected_exit = 22 if arm == "next" and command == "mindnotes nodes create" else 0
                            self.assertEqual(result.returncode, expected_exit, command + ": " + result.stderr[:1500])
                            covered.add(command)
                    receipt = server.receipt()
                    self.assertTrue(M.verify(workspace, answer(case, scenario), receipt, item["expected"]), case)
            self.assertEqual(covered, set(M._METHODS))


if __name__ == "__main__":
    unittest.main()
