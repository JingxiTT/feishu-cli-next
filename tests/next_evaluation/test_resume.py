# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Resume safety controls: no provider subprocesses or live credentials."""
import copy
import io
import json
from pathlib import Path
import shutil
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "evaluation"))
import harness


class ResumeTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.output = self.root / "result.json"
        self.marker = self.root / "result.json.inflight.json"
        self.stop = self.root / "STOP"
        guide = self.root / "guide"
        guide.mkdir()
        (guide / "SKILL.md").write_text("Complete synthetic guidance.\n")
        (guide / "reference.md").write_text("Synthetic reference.\n")
        self.runner = self.root / "runner.py"
        self.runner.write_text("# never executed\n")
        self.verifier = self.root / "verifier.py"
        self.verifier.write_text("# independently never executed\n")
        self.cli = self.root / "cli"
        self.cli.write_text("synthetic executable bytes")
        self.config = {"mode": "controlled", "backend": "synthetic", "model": "fake", "tokenizer": "fake",
            "reasoning": "none", "tools": ["shell"], "permissions": "offline", "budget": {"turns": 2},
            "accounting_method": "synthetic", "runner": [sys.executable, str(self.runner)],
            "verifier": [sys.executable, str(self.verifier)],
            "arms": {arm: {"cli": str(self.cli), "skill": str(guide / "SKILL.md")} for arm in harness.ARMS}}
        self.config_path = self.root / "config.json"
        self.write_config()
        self.task = {"id": "sample", "section": "docs", "prompt": "Synthetic task", "fixture": {}, "expected": {}}
        self.tasks_path = self.root / "tasks.json"
        self.tasks_path.write_text(json.dumps({"split": "controlled", "tasks": [self.task]}))
        self.invocations = []

    def write_config(self):
        self.config_path.write_text(json.dumps(self.config))

    def row(self, config, task, trial, arm):
        self.invocations.append((task["id"], trial, arm))
        return {**harness.run_identity((task, trial, arm)), "success": True, "outcome_inconclusive": False,
            "usage": {"total_tokens": 10}, "known_usage": {"total_tokens": 10}, "categories": [],
            "runner_exit": 0, "verifier_exit": 0, "latency_seconds": 1, "verification_seconds": 1,
            "api_calls": 1, "retries": 0, "command_executions": 1}

    def invoke(self, extra=(), runner=None):
        command = ["harness", "--tasks", str(self.tasks_path), "--config", str(self.config_path),
                   "--output", str(self.output), "--repeats", "2", "--stop-file", str(self.stop), *extra]
        stdout, stderr = io.StringIO(), io.StringIO()
        with patch.object(sys, "argv", command), patch.object(harness, "run_one", side_effect=runner or self.row) as run, \
                patch.object(sys, "stdout", stdout), patch.object(sys, "stderr", stderr):
            result = harness.main()
        return result, run.call_count, json.loads(stderr.getvalue()) if stderr.getvalue() else None

    def checkpoint(self):
        return json.loads(self.output.read_text())

    def partial(self):
        def stop_after_two(*args):
            result = self.row(*args)
            if len(self.invocations) == 2:
                result.update(success=False, usage=None, outcome_inconclusive=True,
                              categories=["usage_unavailable", "fixture_unsupported"], runner_exit="timeout")
                self.stop.touch()
            return result
        self.assertEqual(self.invoke(runner=stop_after_two)[:2], (0, 2))
        self.stop.unlink()
        return copy.deepcopy(self.checkpoint()["runs"])

    def test_resume_reuses_failure_inconclusive_and_unknown_usage_prefix(self):
        prefix = self.partial()
        self.assertEqual(self.invoke(("--resume",))[:2], (1, 4))
        saved = self.checkpoint()
        self.assertEqual(saved["runs"][:2], prefix)
        self.assertEqual(self.invocations, [(t["id"], n, a) for t, n, a in harness.schedule([self.task], 2, 20260907)])
        self.assertTrue(saved["completed"])
        self.assertNotIn("stopped", saved)
        self.assertNotIn("cancelled_before_model", saved)
        self.assertFalse(self.marker.exists())

    def test_existing_output_requires_opt_in_and_completed_never_runs_again(self):
        self.assertEqual(self.invoke()[:2], (0, 6))
        content = self.output.read_bytes()
        code, calls, error = self.invoke()
        self.assertEqual((code, calls, error["category"]), (2, 0, "output_exists_use_resume"))
        self.assertEqual(self.invoke(("--resume",))[:2], (0, 0))
        self.assertEqual(self.output.read_bytes(), content)

    def test_new_output_can_start_with_resume(self):
        self.assertEqual(self.invoke(("--resume",))[:2], (0, 6))

    def test_config_and_guidance_paths_can_move_when_all_bytes_match(self):
        prefix = self.partial()
        shutil.copytree(self.root / "guide", self.root / "moved-guide")
        for arm in self.config["arms"].values():
            arm["skill"] = str(self.root / "moved-guide/SKILL.md")
        self.config_path = self.root / "ephemeral-new-config.json"
        self.write_config()
        self.assertEqual(self.invoke(("--resume",))[:2], (1, 4))
        self.assertEqual(self.checkpoint()["runs"][:2], prefix)

    def test_provenance_and_schedule_changes_refuse_before_runner(self):
        self.partial()
        original_config = copy.deepcopy(self.config)
        original_runner = self.runner.read_bytes()
        original_task = self.tasks_path.read_bytes()
        original_verifier = self.verifier.read_bytes()
        original_cli = self.cli.read_bytes()
        reference = self.root / "guide/reference.md"
        for change in ("model", "timeout", "runner", "verifier", "cli", "reference", "manifest", "seed", "repeats", "arm"):
            with self.subTest(change=change):
                self.config = copy.deepcopy(original_config)
                self.runner.write_bytes(original_runner)
                self.verifier.write_bytes(original_verifier)
                self.cli.write_bytes(original_cli)
                self.tasks_path.write_bytes(original_task)
                reference.write_text("Synthetic reference.\n")
                extra = ["--resume"]
                if change == "model": self.config["model"] = "another"
                elif change == "timeout": self.config["timeout_seconds"] = 999
                elif change == "runner": self.runner.write_text("# different implementation\n")
                elif change == "verifier": self.verifier.write_text("# changed verifier\n")
                elif change == "cli": self.cli.write_text("changed executable bytes")
                elif change == "reference": reference.write_text("changed\n")
                elif change == "manifest": self.tasks_path.write_text(original_task.decode() + "\n")
                elif change == "seed": extra += ["--seed", "20"]
                elif change == "repeats": extra += ["--repeats", "3"]
                elif change == "arm": extra += ["--only-arm", "next"]
                self.write_config()
                self.assertEqual(self.invoke(extra)[:2], (2, 0))

    def test_prefix_and_terminal_rows_are_exact_not_just_a_set(self):
        self.partial()
        original = self.checkpoint()
        for change in ("reverse", "duplicate", "skip", "placeholder", "completed", "summary", "claims", "legacy"):
            with self.subTest(change=change):
                saved = copy.deepcopy(original)
                if change == "reverse": saved["runs"].reverse()
                elif change == "duplicate": saved["runs"][1] = saved["runs"][0]
                elif change == "skip": saved["runs"].pop(0)
                elif change == "placeholder": saved["runs"][1].pop("runner_exit")
                elif change == "completed": saved["completed"] = True
                elif change == "summary": saved["groups"] = []
                elif change == "claims": saved["performance_claims_supported"] = True
                elif change == "legacy": saved.pop("resume_contract")
                self.output.write_text(json.dumps(saved))
                before = self.output.read_bytes()
                self.assertEqual(self.invoke(("--resume",))[:2], (2, 0))
                self.assertEqual(before, self.output.read_bytes())

    def test_death_after_possible_billing_leaves_marker_and_blocks_retry(self):
        def interrupted(*args):
            self.row(*args)
            raise RuntimeError("simulated process death")
        with self.assertRaises(RuntimeError):
            self.invoke(runner=interrupted)
        self.assertTrue(self.marker.exists())
        self.assertEqual(self.checkpoint()["runs"], [])
        code, calls, error = self.invoke(("--resume",))
        self.assertEqual((code, calls, error["category"]), (2, 0, "interrupted_run_requires_review"))

    def test_death_before_result_checkpoint_also_blocks_retry(self):
        atomic = harness.atomic_json
        writes = 0
        def interrupted(path, value):
            nonlocal writes
            if path == self.output.resolve():
                writes += 1
                if writes == 2: raise RuntimeError("simulated checkpoint interruption")
            return atomic(path, value)
        with patch.object(harness, "atomic_json", side_effect=interrupted), self.assertRaises(RuntimeError):
            self.invoke()
        self.assertEqual(len(self.invocations), 1)
        self.assertTrue(self.marker.exists())
        self.assertEqual(self.invoke(("--resume",))[:2], (2, 0))

    def test_checkpointed_exact_marker_is_reconciled_without_billing(self):
        self.invoke()
        saved = self.checkpoint()
        plan = harness.schedule([self.task], 2, 20260907)
        marker = {"resume_contract": saved["resume_contract"], "index": 5, "run": harness.run_identity(plan[5])}
        self.marker.write_text(json.dumps(marker))
        self.assertEqual(self.invoke(("--resume",))[:2], (0, 0))
        self.assertFalse(self.marker.exists())
        self.marker.write_text(json.dumps({**marker, "run": harness.run_identity(plan[0])}))
        self.assertEqual(self.invoke(("--resume",))[:2], (2, 0))
        self.assertTrue(self.marker.exists())

    def test_marker_without_output_is_never_automatically_retried(self):
        self.marker.write_text("{}")
        self.assertEqual(self.invoke(("--resume",))[:2], (2, 0))
        self.assertFalse(self.output.exists())

    def test_output_lock_rejects_concurrent_launcher(self):
        with harness.output_lock(self.output):
            code, calls, error = self.invoke(("--resume",))
        self.assertEqual((code, calls, error["category"]), (2, 0, "output_locked"))
        self.assertFalse(self.output.exists())
        self.assertEqual(self.invoke()[:2], (0, 6))

    def test_atomic_write_failure_preserves_previous_json(self):
        self.output.write_text('{"old": true}\n')
        before = self.output.read_bytes()
        with patch.object(Path, "replace", side_effect=OSError("synthetic replace failure")), self.assertRaises(OSError):
            harness.atomic_json(self.output, {"new": True})
        self.assertEqual(self.output.read_bytes(), before)
        self.assertEqual(list(self.root.glob("result.json.*.tmp")), [])


if __name__ == "__main__":
    unittest.main()
