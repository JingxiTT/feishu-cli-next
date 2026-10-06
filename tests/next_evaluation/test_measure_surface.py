import importlib.util
from pathlib import Path
import subprocess
import sys
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "evaluation"))
SPEC = importlib.util.spec_from_file_location("measure_surface", ROOT / "evaluation/measure_surface.py")
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)
import measure_cli


class SurfaceTests(unittest.TestCase):
    def inventory(self):
        paths = ("", "slides", "slides nodes", "slides nodes get", "apps", "apps +write")
        commands = {path: {"source": "service", "runnable": True, "aliases": None} for path in paths}
        commands["slides"]["aliases"] = ["slide"]
        commands["apps +write"]["disable_flag_parsing"] = True
        coverage = {"commands": {path: {"section": path.split()[0] if path else "foundation"} for path in paths}}
        registrations = {"lark": {"commands": commands, "events": {"changed.v1": {}}}}
        return coverage, registrations

    def test_every_node_alias_method_and_event_is_accounted_for(self):
        coverage, registrations = self.inventory()
        cases = MODULE.inventory_cases(coverage, registrations, {"slides nodes get": {}}, "lark")
        self.assertEqual({case["capability"] for case in cases if case["kind"] == "help"}, set(coverage["commands"]))
        self.assertEqual([case["capability"] for case in cases if case["kind"] == "schema"], ["slides nodes get"])
        self.assertEqual({" ".join(case["args"][:-1]) for case in cases if case["kind"] == "alias-help"},
                         {"slide", "slide nodes", "slide nodes get"})
        guarded = next(case for case in cases if case["id"] == "help:apps +write")
        self.assertEqual(guarded["skip"], "brand_guard_bypasses_help")
        self.assertEqual([case["capability"] for case in cases if case["kind"] == "event"], ["changed.v1"])

    def test_inventory_cannot_silently_omit_a_method_or_node(self):
        coverage, registrations = self.inventory()
        with self.assertRaisesRegex(ValueError, "method missing"):
            MODULE.inventory_cases(coverage, registrations, {"slides unknown get": {}}, "lark")
        del coverage["commands"]["apps"]
        with self.assertRaisesRegex(ValueError, "inventory differs"):
            MODULE.inventory_cases(coverage, registrations, {}, "lark")

    def test_failed_arm_is_unavailable_and_other_arms_still_run(self):
        def run(command, **kwargs):
            if command[0] == "broken":
                return subprocess.CompletedProcess(command, 1, b"", b"private failure")
            return subprocess.CompletedProcess(command, 0, b"{}", b"")
        with patch.object(measure_cli.subprocess, "run", side_effect=run) as calls:
            outputs, samples, failures = measure_cli.probe_variants(
                [("reference", "broken", ["--help"]), ("next", "working", ["--help"])],
                3, {}, "/tmp", tolerate_failures=True)
        self.assertEqual(failures, {"reference": "nonzero_exit"})
        self.assertEqual(len(samples["next"]), 3)
        self.assertEqual(samples["reference"], [])
        self.assertEqual(calls.call_count, 5)
        self.assertNotIn("reference", outputs)

    def test_no_area_average_hides_larger_context_or_unmeasured_functions(self):
        def case(name, before, after):
            return {"kind": "help", "section": "docs", "capability": name,
                    "required_probes_pass": True, "arms": {
                        "reference": {"context_tokens": before, "median_ms": 20},
                        "next": {"context_tokens": after, "median_ms": 10}}}
        cases = [case("read", 1000, 100), case("write", 10, 20),
                 {"kind": "help", "section": "docs", "required_probes_pass": False}]
        summary = MODULE.summarize(cases, ["docs", "extensions", "packaging"])
        row = summary["by_kind"]["help"]
        self.assertEqual((row["planned"], row["measured"], row["context_smaller"], row["context_larger"]), (3, 2, 1, 1))
        self.assertEqual(summary["required_probe_failures"], 1)
        self.assertEqual(summary["non_command_efficiency"]["extensions"], "unmeasured")
        self.assertIn("unverified", summary["functional_and_provider_efficiency"])


if __name__ == "__main__":
    unittest.main()
