import json
from pathlib import Path
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "evaluation"))
import function_coverage as MODULE


class FunctionCoverageTests(unittest.TestCase):
    def matrix(self, root):
        runs = [{"task_id": "read", "section": "docs", "arm": arm, "trial": trial,
                 "success": True, "usage": {"total_tokens": 1000 if arm == "official" else 400},
                 "latency_seconds": 1, "api_calls": 1, "retries": 0, "categories": [],
                 "control_trace": [{"choices": [
                     {"kind": "cli", "command": ["docs", "+fetch"], "flags": ["--help"]},
                     {"kind": "cli", "command": ["docs", "+pull"], "flags": ["--as"]}], "exit_code": 0}]}
                for arm in ("official", "next") for trial in range(3)]
        path = root / "batch.json"
        path.write_text(json.dumps({"runs": runs}))
        (root / "plan.json").write_text(json.dumps({"jobs": [{"key": "job", "suite": "read", "section": "docs", "tasks": ["read"],
                                                            "expected_arms": {arm: {"cli_sha256": "historical"} for arm in ("official", "next")}}]}))
        (root / "summary.json").write_text(json.dumps({"tasks": [{"suite": "read", "task": "read", "gate": "met"}],
                                                     "evidence": [{"path": "batch.json", "key": "job", "sha256": MODULE.sha(path)}]}))

    def test_help_is_not_execution_and_historical_costs_reconstruct_the_gate(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.matrix(root)
            observed, _ = MODULE.historical_usage(root)
            self.assertEqual(set(observed), {"docs +pull"})
            self.assertEqual(sum(row["observed_attempts"] for row in observed["docs +pull"]), 6)
            summary = json.loads((root / "summary.json").read_text())
            summary["tasks"][0]["gate"] = "missed"
            (root / "summary.json").write_text(json.dumps(summary))
            with self.assertRaisesRegex(ValueError, "gates differ"):
                MODULE.historical_usage(root)

    def test_changed_attempt_bytes_are_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.matrix(root)
            (root / "batch.json").write_text('{}')
            with self.assertRaisesRegex(ValueError, "hash changed"):
                MODULE.historical_usage(root)

    def test_even_verified_task_usage_cannot_establish_current_function_savings(self):
        coverage = {"commands": {"docs +pull": {"section": "docs"}}}
        surface = {"commands": {"docs +pull": {"source": "shortcut", "hidden": False}}, "shortcuts": {"docs +pull": {}}, "events": {}}
        probes = {"brand": "feishu", "cases": [{"id": "help:docs +pull", "kind": "help", "required_probes_pass": True}],
                  "binary_sha256": {"next": "new"}}
        sources = {"public_symbol_index": {"sdk.go": ["Build"]}, "conditions": {"host.go": "linux"}}
        observed = {"docs +pull": [{"arm": "next", "cli_sha256": "historical", "verified_task_attempts": 3}]}
        report = MODULE.ledger(coverage, {"feishu": surface}, sources, probes, observed, {"tasks": [{"gate": "met"}]})
        self.assertEqual(report["counts"]["current_function_savings_verified"], 0)
        self.assertEqual(report["whole_cli_savings"], "not_established")
        self.assertFalse(report["commands"]["docs +pull"]["historical_next_binary_matches"])
        probes["cases"] = []
        with self.assertRaisesRegex(ValueError, "omits original"):
            MODULE.ledger(coverage, {"feishu": surface}, sources, probes, observed, {"tasks": []})


if __name__ == "__main__":
    unittest.main()
