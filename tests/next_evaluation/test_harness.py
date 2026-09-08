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
        for names in (("official", "next"), (*harness.ARMS, "next_before")):
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
        inventory = json.loads((ROOT / "docs/reference/coverage.json").read_text())
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


class ProcessE2ETests(unittest.TestCase):
    def invoke(self, fixture=None, split="controlled", extra_args=(), arms=harness.ARMS):
        with tempfile.TemporaryDirectory(prefix="feishu-harness-test-") as directory:
            root = Path(directory)
            task = {"id": "increment", "section": "foundation", "prompt": "Increment the fixture value.",
                    "fixture": {"value": 1, **(fixture or {})}, "expected": {"value": 2}}
            (root / "tasks.json").write_text(json.dumps({"split": split, "tasks": [task]}))
            config = {"mode": "controlled", "backend": "synthetic-state-v1", "model": "synthetic-model",
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
        self.assertEqual(len(result["runs"]), 6)
        self.assertTrue(all(run["success"] for run in result["runs"]))
        self.assertEqual(result["runs"][0]["usage"]["total_tokens"], 165)
        self.assertEqual(result["runs"][0]["command_executions"], 1)
        self.assertEqual(result["runs"][0]["control_trace"], [{"choices": [{"kind": "cli", "command": ["auth", "status"], "flags": ["--json"]}], "exit_code": 3}])
        self.assertEqual(len(result["matched_success"][0]["pairs"]), 2)
        self.assertNotIn("private marker", json.dumps(result))
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
