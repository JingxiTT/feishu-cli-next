# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Pilot wiring must match the independently controlled scenario definitions."""
import json
from pathlib import Path
import sys
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "evaluation"))
from codex_runner import validate_output_schema
from harness import schedule, tasks_from
from scenario_fixture import load_scenario


class PairedManifestTests(unittest.TestCase):
    def test_exact_eight_tasks_match_scenarios_and_balanced_schedule(self):
        manifest = tasks_from(ROOT / "evaluation/tasks.paired.json", False)
        self.assertEqual(manifest["backend"], "synthetic-scenario-v1")
        self.assertEqual(manifest["split"], "development")
        expected = {"paired-" + name for name in (
            "discovery", "contact", "docs-read", "docs-edit", "markdown", "sheets", "base", "drive")}
        self.assertEqual({task["id"] for task in manifest["tasks"]}, expected)
        self.assertEqual(len(manifest["tasks"]), 8)
        for task in manifest["tasks"]:
            module = load_scenario(task["fixture"]["scenario_section"])
            self.assertEqual(task, json.loads(json.dumps(module.TASK)))
            validate_output_schema(task["fixture"]["answer_schema"])
        self.assertEqual(len(schedule(manifest["tasks"], 3, 20260908)), 72)

    def test_paired_loader_does_not_accept_arbitrary_modules(self):
        for name in ("paired-os", "paired-../contact", "paired-__init__"):
            with self.assertRaises(ValueError):
                load_scenario(name)


if __name__ == "__main__":
    unittest.main()
