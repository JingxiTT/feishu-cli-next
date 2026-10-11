# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "evaluation"))
SPEC = importlib.util.spec_from_file_location("harness", ROOT / "evaluation/harness.py")
harness = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(harness)


class AccountingTests(unittest.TestCase):
    def test_guidance_delivery_rejects_invalid_and_per_arm_modes_before_runner(self):
        from unittest.mock import patch
        invalid = [{'guidance_delivery': value} for value in (None, True, [], {}, 'invalid')]
        invalid += [{'guidance_delivery': 'file', 'arms': {'official': {'guidance_delivery': value}}}
                    for value in ('file', 'inline')]
        for config in invalid:
            with self.subTest(config=config), patch.object(harness, 'process') as process:
                with self.assertRaises(ValueError):
                    harness.validate_config(config)
                with self.assertRaises(ValueError):
                    harness.run_one(config, None, 0, 'next')
                process.assert_not_called()
        self.assertEqual(harness.guidance_delivery({}), 'inline')

    def test_accounting_diagnostics_never_keep_unknown_provider_text(self):
        self.assertEqual(harness.sanitized_accounting_diagnostics([
            "provider_turn_failed", "private provider message", {}, "provider_turn_failed"]), ["provider_turn_failed"])
        self.assertIsNone(harness.sanitized_accounting_diagnostics("private provider message"))

    def test_cache_is_not_free_and_all_turns_count(self):
        usage = harness.token_usage({"accounting_complete": True, "turns": [
            {"input_tokens": 100, "cached_input_tokens": 70, "output_tokens": 20},
            {"input_tokens": 30, "cached_input_tokens": 0, "output_tokens": 10}]})
        self.assertEqual(usage["total_tokens"], 160)
        self.assertEqual(usage["cached_input_tokens"], 70)

    def test_incomplete_or_impossible_usage_is_rejected(self):
        for data in ({}, {"accounting_complete": True, "turns": []},
                     {"accounting_complete": True, "turns": [{"input_tokens": 1, "cached_input_tokens": 2, "output_tokens": 0}]}):
            with self.assertRaises(ValueError):
                harness.token_usage(data)

    def test_balanced_reproducible_schedule(self):
        first = harness.schedule([{"id": "a"}, {"id": "b"}], 3, 7)
        self.assertEqual(first, harness.schedule([{"id": "a"}, {"id": "b"}], 3, 7))
        self.assertNotEqual(first, harness.schedule([{"id": "a"}, {"id": "b"}], 3, 8))
        self.assertEqual(len(first), 18)
        for task in ("a", "b"):
            for arm in harness.ARMS:
                self.assertEqual(sum(t["id"] == task and a == arm for t, _, a in first), 3)

    def test_before_after_schedule_is_balanced_and_rejects_mixed_arms(self):
        config = {"arms": {arm: {} for arm in harness.BEFORE_AFTER_ARMS}}
        arms = harness.comparison_arms(config)
        runs = harness.schedule([{"id": "a"}], 3, 7, arms)
        self.assertEqual(len(runs), 9)
        for arm in arms:
            self.assertEqual(sum(a == arm for _, _, a in runs), 3)
        self.assertEqual(harness.comparison_arms({"arms": dict.fromkeys(harness.ORIGINAL_ARMS)}), harness.ORIGINAL_ARMS)
        for names in (("official", "next_before"), (*harness.ARMS, "next_before")):
            with self.assertRaises(ValueError):
                harness.comparison_arms({"arms": dict.fromkeys(names)})

    def test_no_finite_efficiency_for_no_success_or_missing_usage(self):
        base = {"task_id": "a", "section": "foundation", "trial": 0, "arm": "next",
                "success": False, "usage": {"total_tokens": 10}, "categories": ["outcome_failure"],
                "latency_seconds": 1, "api_calls": 0, "retries": 0}
        self.assertIsNone(harness.aggregate([base])["groups"][0]["tokens_per_success"])
        self.assertIsNone(harness.aggregate([{**base, "success": True, "usage": None}])["groups"][0]["tokens_per_success"])

    def test_inconclusive_fixture_retains_cost_without_product_rate(self):
        base = {"task_id": "a", "section": "docs", "trial": 0, "arm": "next",
                "success": False, "outcome_inconclusive": True, "usage": {"total_tokens": 10},
                "categories": ["fixture_unsupported"], "latency_seconds": 1, "api_calls": 0, "retries": 0}
        group = harness.aggregate([base])["groups"][0]
        self.assertEqual(group["known_total_tokens"], 10)
        self.assertEqual(group["inconclusive"], 1)
        self.assertIsNone(group["success_rate"])
        self.assertIsNone(group["tokens_per_success"])

    def test_executable_development_manifests_keep_every_predeclared_section(self):
        expected = {task["section"] for task in harness.tasks_from(ROOT / "evaluation/tasks.development.json")["tasks"]}
        actual = set()
        for name in ('foundation', 'contact', 'read-sections', 'scenarios'):
            actual.update(task['section'] for task in harness.tasks_from(ROOT / ('evaluation/tasks.' + name + '.json'))['tasks'])
        self.assertEqual(actual, expected)

    def test_development_manifest_covers_inventory_sections(self):
        manifest = harness.tasks_from(ROOT / "evaluation/tasks.development.json")
        inventory = json.loads((ROOT / "baseline/coverage.json").read_text())
        self.assertEqual({task["section"] for task in manifest["tasks"]}, set(inventory["sections"]))

    def test_incomplete_runs_keep_known_cost_without_finite_efficiency(self):
        run = {"task_id": "a", "section": "foundation", "trial": 0, "arm": "next",
               "success": False, "usage": None, "known_usage": {"total_tokens": 250},
               "categories": ["usage_unavailable"], "latency_seconds": 1, "api_calls": None, "retries": None}
        result = harness.aggregate([run])["groups"][0]
        self.assertEqual(result["known_total_tokens"], 250)
        self.assertFalse(result["usage_complete"])
        self.assertIsNone(result["tokens_per_success"])

    def test_verifier_diagnostics_never_publish_free_form_content(self):
        self.assertEqual(harness.sanitized_verification_failures(["answer_mismatch"]), ["answer_mismatch"])
        self.assertEqual(harness.sanitized_verification_failures([
            "paired_sheets_formula_read_missing", "paired_markdown_native_content_mismatch"]),
            ["paired_sheets_formula_read_missing", "paired_markdown_native_content_mismatch"])
        self.assertIsNone(harness.sanitized_verification_failures(["private answer text"]))
        task = {"fixture": {"runtime_section": "wiki"}}
        event = {"method": "GET", "route": "rejected", "status": 403, "payload": "private"}
        self.assertEqual(harness.sanitized_api_trace([event], task), [{"method": "GET", "route": "rejected", "status": 403}])
        self.assertIsNone(harness.sanitized_api_trace([{**event, "route": "private path"}], task))
        result = harness.sanitized_api_trace([{**event, "native_read": "docx_raw_content"}], task)
        self.assertEqual(result[0]["native_read"], "docx_raw_content")
        self.assertNotIn("native_read", harness.sanitized_api_trace([{**event, "native_read": "private-id"}], task)[0])

    def test_trace_revalidates_all_persisted_names(self):
        trace = [{"choices": [{"kind": "cli", "command": ["docs", "+fetch"], "flags": ["--doc", "--help"]}], "exit_code": 0, "private_extra": "not retained"}]
        result = harness.sanitized_trace(trace)
        self.assertNotIn("private_extra", json.dumps(result))
        trace[0]["choices"][0]["flags"].append("--secret-user-value")
        self.assertIsNone(harness.sanitized_trace(trace))

    def test_cli_error_trace_revalidates_every_field_and_omits_untrusted_content(self):
        event = {"choices": [{"kind": "cli", "command": ["docs", "+push"], "flags": ["--file"]}],
                 "exit_code": 22, "reported_cli_error": {"type": "api", "subtype": "mutation_outcome",
                    "outcome": "unknown", "message": "private", "receipts": ["private"], "cause": {
                        "type": "validation", "subtype": "failed_precondition", "param": "--file",
                        "reason": "workspace_readback_mismatch", "message": "private", "cause": {"private": True}}}}
        self.assertEqual(harness.sanitized_trace([event])[0]["reported_cli_error"], {
            "type": "api", "subtype": "mutation_outcome", "outcome": "unknown", "cause": {
                "type": "validation", "subtype": "failed_precondition", "param": "--file",
                "reason": "workspace_readback_mismatch"}})
        for diagnostic in ([], {"type": "private", "subtype": "private"},
                           {"type": "validation", "subtype": ["failed_precondition"]}):
            clean = harness.sanitized_trace([{**event, "reported_cli_error": diagnostic}])
            self.assertNotIn("reported_cli_error", clean[0])
            self.assertEqual(clean[0]["exit_code"], 22)
        clean = harness.sanitized_trace([{**event, "reported_cli_error": {
            "type": "validation", "subtype": "failed_precondition", "reason": "private", "param": "private"}}])
        self.assertEqual(clean[0]["reported_cli_error"], {"type": "validation", "subtype": "failed_precondition"})
        for extra in ({"exit_code": 0}, {"exit_code": True}, {"choices": [{"kind": "script"}]},
                      {"choices": event["choices"] + [{"kind": "file_read"}]}):
            self.assertNotIn("reported_cli_error", harness.sanitized_trace([{**event, **extra}])[0])

    def test_reported_cli_errors_cannot_change_verdict_or_cost(self):
        from unittest.mock import patch
        config = {"mode": "controlled", "backend": "synthetic-state-v1", "model": "synthetic-model",
                  "tokenizer": "synthetic-counts-v1", "reasoning": "none", "tools": [], "permissions": "offline",
                  "budget": {}, "runner": ["runner"], "verifier": ["verifier"],
                  "arms": {"next": {"cli": "unused", "skill": "unused"}}}
        task = {"id": "controlled", "section": "docs", "prompt": "controlled", "fixture": {}, "expected": {}}
        event = {"choices": [{"kind": "cli", "command": ["docs", "+push"], "flags": []}], "exit_code": 2}
        payload = {"usage_source": "synthetic", "model": config["model"], "tokenizer": config["tokenizer"],
                   "accounting_complete": True, "claimed_success": True,
                   "turns": [{"input_tokens": 200, "cached_input_tokens": 80, "output_tokens": 30}]}
        for verification in ({"success": True}, {"success": False}, {"success": False, "fixture_unsupported": True}):
            baseline = None
            for diagnostic in (None, {"type": "validation", "subtype": "failed_precondition",
                                      "reason": "workspace_edit_unsupported"}, {"type": "private", "subtype": []}):
                trace = [{**event, **({"reported_cli_error": diagnostic} if diagnostic is not None else {})}]
                with patch.object(harness, "process", side_effect=[({**payload, "control_trace": trace}, 0, 1),
                                                                  (verification, 0, 1)]):
                    result = harness.run_one(config, task, 0, "next")
                result.pop("control_trace")
                if baseline is None:
                    baseline = result
                self.assertEqual(result, baseline)
                self.assertEqual(result["usage"]["total_tokens"], 230)


class ProcessE2ETests(unittest.TestCase):
    def invoke(self, fixture=None, split="controlled", extra_args=(), arms=harness.ARMS, task_count=1,
               guidance_delivery="inline"):
        with tempfile.TemporaryDirectory(prefix="feishu-harness-test-") as directory:
            root = Path(directory)
            task = {"id": "increment", "section": "foundation", "prompt": "Increment the fixture value.",
                    "fixture": {"value": 1, **(fixture or {})}, "expected": {"value": 2}}
            tasks = [task] if task_count == 1 else [{**task, "id": f"increment-{n}"} for n in range(task_count)]
            (root / "tasks.json").write_text(json.dumps({"split": split, "tasks": tasks}))
            config = {"mode": "controlled", "backend": "synthetic-state-v1", "model": "synthetic-model",
                      "guidance_delivery": guidance_delivery,
                      "tokenizer": "synthetic-counts-v1", "reasoning": "none", "tools": ["python"],
                      "permissions": "isolated-offline-fixture", "budget": {"turns": 2},
                      "accounting_method": "invented test counts; not real model usage", "timeout_seconds": 5,
                      "runner": [sys.executable, str(ROOT / "tests/next_evaluation/fake_runner.py")],
                      "verifier": [sys.executable, str(ROOT / "tests/next_evaluation/fake_verifier.py")],
                      "arms": {arm: {"cli": sys.executable, "skill": str(ROOT / "tests/next_evaluation/fake_runner.py")} for arm in arms}}
            (root / "config.json").write_text(json.dumps(config))
            completed = subprocess.run([sys.executable, str(ROOT / "evaluation/harness.py"),
                                        "--tasks", str(root / "tasks.json"), "--config", str(root / "config.json"),
                                        "--output", str(root / "result.json"), "--repeats", "2", *extra_args],
                                       capture_output=True, text=True, timeout=30,
                                       env={**os.environ, "FEISHU_EVAL_TEST_SECRET": "must-not-be-inherited"})
            result = json.loads((root / "result.json").read_text()) if (root / "result.json").exists() else None
            return completed, result

    def test_complete_controlled_comparison(self):
        completed, result = self.invoke()
        self.assertEqual(completed.returncode, 0, completed.stderr)
        self.assertEqual(json.loads(completed.stdout)["status"], "pass")
        self.assertFalse(result["performance_claims_supported"])
        self.assertEqual(result['environment']['guidance_delivery'], 'inline')
        self.assertEqual(len(result["runs"]), 6)
        self.assertEqual(result["planned_runs"], 6)
        self.assertNotIn("arm_selection", result)
        self.assertTrue(all(run["success"] for run in result["runs"]))
        self.assertEqual(result["runs"][0]["usage"]["total_tokens"], 165)
        self.assertEqual(result["runs"][0]["command_executions"], 1)
        self.assertEqual(result["runs"][0]["control_trace"], [{"choices": [{"kind": "cli", "command": ["auth", "status"], "flags": ["--json"]}], "exit_code": 3}])
        self.assertEqual(len(result["matched_success"][0]["pairs"]), 2)
        self.assertNotIn("private marker", json.dumps(result))

    def test_file_guidance_is_shared_and_persisted_for_original_comparison(self):
        completed, result = self.invoke(fixture={'expected_guidance_delivery': 'file'}, guidance_delivery='file',
                                        arms=harness.ORIGINAL_ARMS, extra_args=('--original-comparison',))
        self.assertEqual(completed.returncode, 0, completed.stderr)
        self.assertEqual(result['environment']['guidance_delivery'], 'file')
        self.assertEqual({run['arm'] for run in result['runs']}, {'official', 'next'})
        self.assertTrue(all(run['success'] and run['usage']['total_tokens'] == 165 for run in result['runs']))
        self.assertEqual(result['environment']['budget'], {'turns': 2})
        self.assertNotIn("must-not-be-inherited", json.dumps(result))

    def test_before_after_comparison_keeps_named_baselines_and_failed_costs(self):
        for fixture in ({}, {"wrong_effect": True}):
            completed, result = self.invoke(fixture, arms=harness.BEFORE_AFTER_ARMS)
            self.assertEqual(completed.returncode, int(bool(fixture)), completed.stderr)
            self.assertEqual(set(result["arms"]), set(harness.BEFORE_AFTER_ARMS))
            self.assertEqual({row["baseline"] for row in result["matched_success"]}, {"official", "next_before"})
            self.assertEqual(len(result["runs"]), 6)
            for row in result["groups"]:
                self.assertEqual(row["known_total_tokens"], 330)
                self.assertEqual(row["successes"], 0 if fixture else 2)
            self.assertTrue(all(len(row["pairs"]) == (0 if fixture else 2) for row in result["matched_success"]))

    def test_candidate_only_invokes_filtered_balanced_schedule_and_keeps_costs(self):
        for arms in (harness.ARMS, harness.BEFORE_AFTER_ARMS):
            with self.subTest(arms=arms), tempfile.TemporaryDirectory() as directory:
                log = Path(directory) / "invocations.jsonl"
                completed, result = self.invoke(
                    {"invocation_log": str(log), "wrong_effect": True},
                    extra_args=("--only-arm", "next"), arms=arms)
                self.assertEqual(completed.returncode, 1, completed.stderr)
                expected = [(trial, arm) for _, trial, arm in
                            harness.schedule([{"id": "increment"}], 2, 20260907, arms) if arm == "next"]
                actual = [json.loads(line) for line in log.read_text().splitlines()]
                self.assertEqual([(row["trial"], row["arm"]) for row in actual], expected)
                self.assertEqual([(row["trial"], row["arm"]) for row in result["runs"]], expected)
                self.assertEqual(result["planned_runs"], 2)
                self.assertTrue(result["completed"])
                self.assertEqual(set(result["arms"]), set(arms))
                self.assertEqual(result["arm_selection"], {
                    "mode": "candidate_only", "selected": ["next"],
                    "schedule": "filtered_balanced", "balanced_comparison": False})
                self.assertFalse(result["performance_claims_supported"])
                self.assertEqual(result["matched_success"], [])
                self.assertTrue(all(row["arm"] == "next" and row["known_total_tokens"] == 330
                                    and row["successes"] == 0 for row in result["groups"]))

    def test_candidate_only_rejects_other_arm_names_before_runner(self):
        completed, result = self.invoke(extra_args=("--only-arm", "official"))
        self.assertEqual(completed.returncode, 2)
        self.assertIsNone(result)

    def test_candidate_only_eight_tasks_three_trials_invokes_24_attempts(self):
        with tempfile.TemporaryDirectory() as directory:
            log = Path(directory) / "invocations.jsonl"
            completed, result = self.invoke({"invocation_log": str(log)}, task_count=8,
                arms=harness.BEFORE_AFTER_ARMS, extra_args=("--only-arm", "next", "--repeats", "3"))
            self.assertEqual(completed.returncode, 0, completed.stderr)
            actual = [json.loads(line) for line in log.read_text().splitlines()]
            tasks = [{"id": f"increment-{n}"} for n in range(8)]
            expected = [(task["id"], trial, arm) for task, trial, arm in
                        harness.schedule(tasks, 3, 20260907, harness.BEFORE_AFTER_ARMS) if arm == "next"]
            self.assertEqual([(r["task_id"], r["trial"], r["arm"]) for r in actual], expected)
            self.assertEqual(result["planned_runs"], 24)
            self.assertEqual(len(result["runs"]), 24)

    def test_original_comparison_eight_tasks_three_trials_invokes_48_balanced_attempts(self):
        with tempfile.TemporaryDirectory() as directory:
            log = Path(directory) / "invocations.jsonl"
            completed, result = self.invoke({"invocation_log": str(log)}, task_count=8,
                arms=harness.ORIGINAL_ARMS, extra_args=("--original-comparison", "--repeats", "3"))
            self.assertEqual(completed.returncode, 0, completed.stderr)
            actual = [json.loads(line) for line in log.read_text().splitlines()]
            tasks = [{"id": f"increment-{n}"} for n in range(8)]
            expected = [(task["id"], trial, arm) for task, trial, arm in
                        harness.schedule(tasks, 3, 20260907, harness.ORIGINAL_ARMS)]
            self.assertEqual([(r["task_id"], r["trial"], r["arm"]) for r in actual], expected)
            self.assertEqual(result["planned_runs"], 48)
            self.assertEqual(len(result["runs"]), 48)
            self.assertEqual(set(result["arms"]), {"official", "next"})
            self.assertEqual(result["arm_selection"], {"mode": "original_comparison",
                "selected": ["official", "next"], "schedule": "balanced", "balanced_comparison": True})
            self.assertEqual(len(result["matched_success"]), 1)
            self.assertEqual(result["matched_success"][0]["baseline"], "official")
            self.assertEqual(len(result["matched_success"][0]["pairs"]), 24)
            self.assertTrue(all(row["attempts"] == 3 for row in result["groups"] if row["scope"] == "task_id"))

    def test_original_comparison_keeps_failed_cost_and_unknown_usage(self):
        for fixture in ({"wrong_effect": True}, {"invalid_usage": True}):
            completed, result = self.invoke(fixture, arms=harness.ORIGINAL_ARMS,
                                            extra_args=("--original-comparison",))
            self.assertEqual(completed.returncode, 1, completed.stderr)
            self.assertEqual(len(result["runs"]), 4)
            self.assertTrue(all(not row["success"] for row in result["runs"]))
            for row in result["groups"]:
                self.assertIsNone(row["tokens_per_success"])
                if fixture.get("wrong_effect"):
                    self.assertEqual(row["known_total_tokens"], 330)
                else:
                    self.assertFalse(row["usage_complete"])
            if fixture.get("invalid_usage"):
                self.assertTrue(all(row["usage"] is None for row in result["runs"]))

    def test_original_comparison_rejects_mismatched_or_candidate_only_options(self):
        for arms, args in ((harness.ORIGINAL_ARMS, ()),
                           (harness.ARMS, ("--original-comparison",)),
                           (harness.BEFORE_AFTER_ARMS, ("--original-comparison",)),
                           (harness.ORIGINAL_ARMS, ("--original-comparison", "--only-arm", "next"))):
            with self.subTest(arms=arms, args=args):
                completed, result = self.invoke(arms=arms, extra_args=args)
                self.assertEqual(completed.returncode, 2)
                self.assertIsNone(result)

    def test_operator_stop_preserves_active_attempt_and_skips_unstarted_models(self):
        with tempfile.TemporaryDirectory() as directory:
            stop = Path(directory) / 'STOP'
            completed, result = self.invoke({'operator_stop_file': str(stop)}, extra_args=['--stop-file', str(stop)])
            self.assertEqual(completed.returncode, 0, completed.stderr)
            self.assertEqual(json.loads(completed.stdout)['status'], 'stopped')
            self.assertTrue(result['stopped'])
            self.assertEqual(result['cancelled_before_model'], 5)
            self.assertEqual(len(result['runs']), 1)
            self.assertEqual(result['runs'][0]['usage']['total_tokens'], 165)
            self.assertTrue(result['runs'][0]['success'])
            self.assertFalse(result['completed'])
            self.assertFalse(result['performance_claims_supported'])

    def test_agent_claim_cannot_override_actual_effect(self):
        completed, result = self.invoke({"wrong_effect": True})
        self.assertEqual(completed.returncode, 1)
        self.assertTrue(all(not run["success"] for run in result["runs"]))
        self.assertIn("misleading_success_claim", result["runs"][0]["categories"])
        self.assertEqual(result["groups"][0]["known_total_tokens"], 330)
        self.assertIsNone(result["groups"][0]["tokens_per_success"])

    def test_failed_process_token_costs_are_kept(self):
        completed, result = self.invoke({"nonzero_exit": True})
        self.assertEqual(completed.returncode, 1)
        self.assertEqual(result["runs"][0]["runner_exit"], 7)
        self.assertEqual(result["runs"][0]["usage"]["total_tokens"], 165)
        self.assertFalse(result["runs"][0]["success"])

    def test_missing_accounting_is_not_zero_usage(self):
        completed, result = self.invoke({"invalid_usage": True})
        self.assertEqual(completed.returncode, 1)
        self.assertIsNone(result["runs"][0]["usage"])
        self.assertFalse(result["groups"][0]["usage_complete"])
        self.assertIsNone(result["groups"][0]["tokens_per_success"])

    def test_held_out_is_not_executed_without_release_flag(self):
        completed, result = self.invoke(split="held_out")
        self.assertEqual(completed.returncode, 2)
        self.assertIsNone(result)


if __name__ == "__main__":
    unittest.main()
