#!/usr/bin/env python3
"""Account for every original capability without promoting task samples to parity.

Bind local probes and hash-bound historical workflow traces to the inventory.
Observed command use is not functional verification or per-function savings.
"""
import argparse
from collections import Counter
import gzip
import json
from pathlib import Path

from measure_surface import ROOT, read, sha
from harness import ORIGINAL_ARMS, aggregate
from run_matrix import score


def historical_usage(matrix):
    summary = read(matrix / "summary.json")
    plan = {job["key"]: job for job in read(matrix / "plan.json")["jobs"]}
    observations = {}
    reconstructed_tasks = {}
    for evidence in summary["evidence"]:
        path = (matrix / evidence["path"]).resolve()
        if not path.is_relative_to(matrix.resolve()):
            raise ValueError("matrix evidence escapes its directory")
        if sha(path) != evidence["sha256"]:
            raise ValueError("matrix evidence hash changed")
        seal = path.parent / "completed.json"
        if seal.exists() and read(seal)["batch_sha256"] != sha(path):
            raise ValueError("completed matrix seal changed")
        job = plan[evidence["key"]]
        batch = read(path)
        grouped = aggregate(batch["runs"], ORIGINAL_ARMS)["groups"]
        for task in job["tasks"]:
            arms = {row["arm"]: row for row in grouped if row["scope"] == "task_id" and row["name"] == task}
            key = (job["suite"], task)
            reconstructed_tasks[key] = (score(arms, job["section"] == "docs")["gate"]
                                         if len(arms) == 2 and all(row["attempts"] == 3 for row in arms.values()) else "pending")
        for run in batch["runs"]:
            seen = set()
            for event in run.get("control_trace") or []:
                for choice in event["choices"]:
                    if choice["kind"] != "cli" or "--help" in choice["flags"]:
                        continue
                    path_words = choice["command"]
                    # The single frozen alias applies to its whole subtree.
                    if path_words and path_words[0] == "slide":
                        path_words = ["slides", *path_words[1:]]
                    seen.add(" ".join(path_words))
            for command in seen:
                key = (job["suite"], run["task_id"], run["arm"])
                value = observations.setdefault(command, {}).setdefault(key, {
                    "suite": job["suite"], "task": run["task_id"], "arm": run["arm"],
                    "observed_attempts": 0, "verified_task_attempts": 0,
                    "cli_sha256": job["expected_arms"][run["arm"]]["cli_sha256"]})
                value["observed_attempts"] += 1
                value["verified_task_attempts"] += int(run["success"] is True)
    published_tasks = {(task["suite"], task["task"]): task["gate"] for task in summary["tasks"]}
    if reconstructed_tasks != published_tasks:
        raise ValueError("published task gates differ from retained attempts")
    return {command: list(rows.values()) for command, rows in observations.items()}, summary


def ledger(coverage, registrations, sources, probes, observed, matrix_summary):
    surface = registrations[probes["brand"]]
    rows = {case["id"]: case for case in probes["cases"]}
    expected = {"help:" + path for path in surface["commands"]}
    if not expected <= set(rows):
        raise ValueError("surface report omits original command nodes")
    commands = {}
    for path, contract in surface["commands"].items():
        history = observed.get(path, [])
        commands[path] = {"section": coverage["commands"][path]["section"],
                          "source": contract["source"], "hidden": contract["hidden"],
                          "local_help": rows["help:" + path]["required_probes_pass"],
                          "local_method_schema": rows.get("schema:" + path, {}).get("required_probes_pass", False),
                          "historical_workflow_observations": history,
                          "historical_next_binary_matches": any(
                              row["arm"] == "next" and row["cli_sha256"] == probes["binary_sha256"]["next"] for row in history),
                          "current_function_savings": "unverified"}
    return {"schema": 1, "whole_cli_savings": "not_established",
            "counts": {"original_command_nodes": len(commands), "original_shortcuts": len(surface["shortcuts"]),
                       "original_generated_methods": sum(row["kind"] == "schema" for row in probes["cases"]),
                       "original_event_keys": len(surface["events"]),
                       "nodes_observed_in_historical_workflows": sum(bool(value["historical_workflow_observations"]) for value in commands.values()),
                       "nodes_without_historical_workflow_observations": sum(not value["historical_workflow_observations"] for value in commands.values()),
                       "current_function_savings_verified": 0},
            "historical_task_gates": dict(Counter(task["gate"] for task in matrix_summary["tasks"])),
            "interpretation": "A command appearing in a verified task does not verify every mode or attribute the task saving to that command. Help/schema probes measure introspection only. Historical binary identities are retained; these trials do not evaluate a changed candidate.",
            "commands": commands,
            "events": {key: {"local_schema": rows["event:" + key]["required_probes_pass"],
                             "consume_and_recovery_savings": "unverified"} for key in surface["events"]},
            "extension_public_symbol_index": sources["public_symbol_index"],
            "platform_build_conditions": sources["conditions"],
            "non_command_savings": {"extensions": "unverified", "packaging": "unverified"}}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--probes", type=Path, required=True)
    parser.add_argument("--matrix", type=Path, required=True)
    parser.add_argument("--reference-dir", type=Path, default=ROOT / "docs/reference")
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    probes = read(args.probes)
    registration_path = args.reference_dir / "configured-brand-registrations.json.gz"
    coverage_path = args.reference_dir / "coverage.json"
    if sha(registration_path) != probes["registration_sha256"] or sha(coverage_path) != probes["coverage_sha256"]:
        raise ValueError("probe/reference inventory binding changed")
    observed, summary = historical_usage(args.matrix)
    report = ledger(read(coverage_path), read(registration_path), read(args.reference_dir / "sources.json"),
                    probes, observed, summary)
    report["inputs"] = {"probe_sha256": sha(args.probes), "matrix_summary_sha256": sha(args.matrix / "summary.json"),
                        "matrix_plan_sha256": sha(args.matrix / "plan.json"), "sources_sha256": sha(args.reference_dir / "sources.json"),
                        "runner_sha256": sha(Path(__file__))}
    raw = (json.dumps(report, indent=2) + "\n").encode()
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_bytes(gzip.compress(raw, mtime=0) if args.output.suffix == ".gz" else raw)
    print(json.dumps({"status": report["whole_cli_savings"], **report["counts"]}))


if __name__ == "__main__":
    main()
