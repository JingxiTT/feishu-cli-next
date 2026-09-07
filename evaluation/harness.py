#!/usr/bin/env python3
"""Balanced CLI agent evaluation. Python standard library only."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import random
import statistics
import subprocess
import sys
import tempfile
import time


ARMS = ("official", "official_concise", "next")


def read_json(path):
    return json.loads(Path(path).read_text(encoding="utf-8"))


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def tasks_from(path, allow_held_out=False):
    manifest = read_json(path)
    if manifest.get("split") not in ("development", "held_out", "controlled"):
        raise ValueError("invalid task split")
    if manifest["split"] == "held_out" and not allow_held_out:
        raise ValueError("held-out tasks require --release-held-out; do not use during tuning")
    tasks = manifest["tasks"]
    if not tasks or len({task["id"] for task in tasks}) != len(tasks):
        raise ValueError("tasks must be nonempty with unique IDs")
    for task in tasks:
        for field in ("id", "section", "prompt", "fixture", "expected"):
            if field not in task:
                raise ValueError("task missing " + field)
        if not isinstance(task["id"], str) or not task["id"].replace("-", "").replace("_", "").isalnum():
            raise ValueError("task ID must contain only letters, digits, underscores or hyphens")
    return manifest


def schedule(tasks, repeats, seed):
    if repeats < 2:
        raise ValueError("at least two independent trials per task and arm are required")
    runs = [(task, trial, arm) for task in tasks for trial in range(repeats) for arm in ARMS]
    random.Random(seed).shuffle(runs)
    return runs


def token_usage(payload):
    """Input includes cached tokens, which are also reported separately."""
    if payload.get("accounting_complete") is not True:
        raise ValueError("complete provider usage is required, including errors/recovery")
    turns = payload.get("turns")
    if not isinstance(turns, list) or not turns:
        raise ValueError("per-turn model usage required")
    total = {"input_tokens": 0, "cached_input_tokens": 0, "output_tokens": 0}
    for turn in turns:
        for key in total:
            value = turn.get(key)
            if type(value) is not int or value < 0:
                raise ValueError("token counts must be nonnegative integers")
            total[key] += value
        if turn["cached_input_tokens"] > turn["input_tokens"]:
            raise ValueError("cached input must be included in input_tokens")
    total["total_tokens"] = total["input_tokens"] + total["output_tokens"]
    return total


def process(command, request, workspace, timeout):
    # No inherited profiles, tokens, tenant variables, proxy settings, or Python paths.
    env = {"PATH": os.defpath, "HOME": str(workspace), "TMPDIR": str(workspace),
           "LANG": "C.UTF-8", "LC_ALL": "C.UTF-8", "PYTHONUTF8": "1", "PYTHONIOENCODING": "utf-8"}
    started = time.monotonic()
    try:
        completed = subprocess.run(command, input=json.dumps(request), text=True, encoding="utf-8",
                                   stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                   cwd=workspace, env=env, timeout=timeout, check=False)
        elapsed = time.monotonic() - started
        try:
            payload = json.loads(completed.stdout)
            if not isinstance(payload, dict):
                payload = {}
        except (ValueError, TypeError):
            payload = {}
        # Do not persist untrusted stdout/stderr or exception text: they can contain secrets.
        return payload, completed.returncode, elapsed
    except subprocess.TimeoutExpired:
        return {}, "timeout", time.monotonic() - started
    except OSError:
        return {}, "launch_failed", time.monotonic() - started


def validate_config(config):
    if not isinstance(config, dict):
        raise ValueError("config must be an object")
    for key in ("backend", "model", "tokenizer", "reasoning", "tools", "permissions", "budget", "accounting_method"):
        if not config.get(key):
            raise ValueError("pin config field: " + key)
    if config.get("mode") not in ("controlled", "real"):
        raise ValueError("mode must be controlled or real")
    if set(config.get("arms", {})) != set(ARMS):
        raise ValueError("exactly three comparison arms required")
    for key in ("runner", "verifier"):
        command = config.get(key)
        if not isinstance(command, list) or not command or not all(isinstance(arg, str) for arg in command):
            raise ValueError(key + " must be a nonempty argv array")
        if not Path(command[0]).is_absolute():
            raise ValueError(key + " executable must be absolute")
    if config["runner"] == config["verifier"]:
        raise ValueError("verifier must be independent of agent runner")
    official = config["arms"]["official"]["cli"]
    if official != config["arms"]["official_concise"]["cli"]:
        raise ValueError("official arms must use the identical CLI")
    if type(config.get("timeout_seconds", 120)) not in (int, float) or config.get("timeout_seconds", 120) <= 0:
        raise ValueError("timeout_seconds must be positive")
    for arm in config["arms"].values():
        for key in ("cli", "skill"):
            if not Path(arm[key]).is_absolute() or not Path(arm[key]).is_file():
                raise ValueError("each arm requires an absolute existing " + key + " file")


def run_one(config, task, trial, arm):
    with tempfile.TemporaryDirectory(prefix="feishu-eval-") as directory:
        workspace = Path(directory)
        (workspace / "fixture.json").write_text(json.dumps(task["fixture"], ensure_ascii=False), encoding="utf-8")
        request = {"protocol": 1, "task_id": task["id"], "trial": trial, "arm": arm,
                   "prompt": task["prompt"], "fixture": task["fixture"],
                   "workspace": directory, "fresh_context": True,
                   "cli": config["arms"][arm]["cli"], "skill": config["arms"][arm]["skill"],
                   "settings": {key: config[key] for key in
                                ("backend", "model", "tokenizer", "reasoning", "tools", "permissions", "budget", "mode")}}
        if "selected_skill_source" in config["arms"][arm]:
            request["selected_skill_source"] = config["arms"][arm]["selected_skill_source"]
        payload, code, elapsed = process(config["runner"], request, workspace, config.get("timeout_seconds", 120))
        categories = []
        if code != 0:
            categories.append("runner_failure")
        known_usage = None
        try:
            if payload.get("usage_source") != ("synthetic" if config["mode"] == "controlled" else "provider"):
                raise ValueError("usage source mismatch")
            if payload.get("model") != config["model"] or payload.get("tokenizer") != config["tokenizer"]:
                raise ValueError("usage provenance mismatch")
            # Preserve completed spans from an interrupted run, but never call them its total.
            known_usage = token_usage({**payload, "accounting_complete": True})
            usage = token_usage(payload)
        except (ValueError, TypeError, AttributeError):
            usage = None
            categories.append("usage_unavailable")
        # Expected outcomes are never sent to the runner. Check actual fixture/backend effects.
        verification, verify_code, verify_elapsed = process(config["verifier"], {
            "protocol": 1, "workspace": directory, "task_id": task["id"],
            "fixture": task["fixture"], "expected": task["expected"],
            "backend": config["backend"], "cli": config["arms"][arm]["cli"],
            "claimed_success": payload.get("claimed_success")},
            workspace, config.get("timeout_seconds", 120))
        verified = verify_code == 0 and verification.get("success") is True
        inconclusive = verification.get("fixture_unsupported") is True
        if inconclusive:
            categories.append("fixture_unsupported")
        if not verified and not inconclusive:
            categories.append("outcome_failure" if verify_code == 0 else "verifier_failure")
        misleading = payload.get("claimed_success") is True and not verified and not inconclusive
        if misleading:
            categories.append("misleading_success_claim")
        # Whitelist numeric measurements only; raw content and free-form traces stay out of evidence.
        metrics = {}
        for key in ("api_calls", "retries", "command_executions"):
            value = payload.get(key)
            metrics[key] = value if type(value) is int and value >= 0 else None
        return {"task_id": task["id"], "section": task["section"], "trial": trial, "arm": arm,
                "success": verified and code == 0, "outcome_inconclusive": inconclusive, "usage": usage, "categories": categories,
                "known_usage": known_usage,
                "verification_failures": sanitized_verification_failures(verification.get("verification_failures")),
                "api_request_trace": sanitized_api_trace(payload.get("api_request_trace"), task),
                "control_trace": sanitized_trace(payload.get("control_trace")),
                "usage_granularity": payload.get("usage_granularity") if payload.get("usage_granularity") in ("model_turn", "codex_user_turn") else None,
                "runner_exit": code, "verifier_exit": verify_code,
                "latency_seconds": elapsed, "verification_seconds": verify_elapsed, **metrics}



def sanitized_verification_failures(values):
    allowed = {"answer_mismatch", "scenario_mismatch", "missing_request_evidence", "request_outside_fixture",
               "required_reads_missing", "config_changed", "invalid_verification_input"}
    return values if isinstance(values, list) and all(isinstance(value, str) and value in allowed for value in values) else None


def sanitized_api_trace(values, task):
    section = task.get("fixture", {}).get("runtime_section")
    if section is None or not isinstance(values, list):
        return None
    from read_fixture import load_runtime, NATIVE_READS
    try:
        allowed = {route["id"] for route in load_runtime(section)["routes"]} | {"rejected", "fixture_error"}
    except (OSError, ValueError, KeyError, TypeError):
        return None
    result = []
    for value in values:
        if not isinstance(value, dict) or value.get("route") not in allowed or value.get("method") not in ("GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"):
            return None
        status = value.get("status")
        if type(status) is not int or not 100 <= status <= 599:
            return None
        record = {"method": value["method"], "route": value["route"], "status": status}
        if isinstance(value.get("native_read"), str) and value["native_read"] in NATIVE_READS:
            record["native_read"] = value["native_read"]
        result.append(record)
    return result

def sanitized_trace(trace):
    """Validate even adapter-provided trace strings against the public vocabulary."""
    if not isinstance(trace, list):
        return None
    from codex_runner import trace_vocabulary
    paths, flags = trace_vocabulary()
    result = []
    for event in trace:
        if not isinstance(event, dict) or not isinstance(event.get("choices"), list):
            return None
        choices = []
        for choice in event["choices"]:
            if not isinstance(choice, dict):
                return None
            kind = choice.get("kind")
            if kind == "cli":
                command, names = choice.get("command"), choice.get("flags")
                if not isinstance(command, list) or not all(isinstance(word, str) for word in command) or tuple(command) not in paths:
                    return None
                if not isinstance(names, list) or not all(isinstance(name, str) and name in flags for name in names):
                    return None
                choices.append({"kind": kind, "command": command, "flags": names})
            elif kind in ("file_read", "file_search", "script", "other_shell", "unparsed_shell"):
                choices.append({"kind": kind})
            else:
                return None
        code = event.get("exit_code")
        result.append({"choices": choices, "exit_code": code if type(code) is int else None})
    return result


def aggregate(runs):
    groups = {}
    for scope in ("all", "section", "task_id"):
        for run in runs:
            key = (scope, "all" if scope == "all" else run[scope], run["arm"])
            groups.setdefault(key, []).append(run)
    rows = []
    for (scope, name, arm), values in sorted(groups.items()):
        successes = sum(run["success"] for run in values)
        tokens = [run["usage"]["total_tokens"] for run in values if run["usage"] is not None]
        known_tokens = [observed["total_tokens"] for run in values if (observed := run["usage"] or run.get("known_usage")) is not None]
        complete = len(tokens) == len(values)
        inconclusive = sum(bool(run.get("outcome_inconclusive")) for run in values)
        rows.append({"scope": scope, "name": name, "arm": arm, "attempts": len(values),
                     "successes": successes, "inconclusive": inconclusive,
                     "success_rate": successes / len(values) if not inconclusive else None,
                     "usage_complete": complete, "known_total_tokens": sum(known_tokens),
                     "tokens_per_success": sum(tokens) / successes if complete and successes and not inconclusive else None,
                     "token_distribution": {"min": min(tokens), "median": statistics.median(tokens),
                                            "max": max(tokens), "stdev": statistics.stdev(tokens) if len(tokens) > 1 else 0} if tokens else None,
                     "latency_median_seconds": statistics.median(run["latency_seconds"] for run in values),
                     "api_calls": sum(run["api_calls"] for run in values) if all(run["api_calls"] is not None for run in values) else None,
                     "retries": sum(run["retries"] for run in values) if all(run["retries"] is not None for run in values) else None,
                     "failure_categories": {category: sum(category in run["categories"] for run in values)
                                            for category in sorted({category for run in values for category in run["categories"]})}})
    matched = []
    indexed = {(run["task_id"], run["trial"], run["arm"]): run for run in runs}
    for baseline in ARMS[:2]:
        pairs = []
        for run in runs:
            other = indexed.get((run["task_id"], run["trial"], baseline))
            if run["arm"] == "next" and other and run["success"] and other["success"] and run["usage"] and other["usage"]:
                pairs.append({"task_id": run["task_id"], "trial": run["trial"],
                              "baseline_tokens": other["usage"]["total_tokens"], "next_tokens": run["usage"]["total_tokens"]})
        matched.append({"baseline": baseline, "pairs": pairs})
    return {"groups": rows, "matched_success": matched}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tasks", type=Path, required=True)
    parser.add_argument("--config", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--repeats", type=int, default=3)
    parser.add_argument("--stop-file", type=Path, help="Operator stop request, checked only between complete attempts")
    parser.add_argument("--seed", type=int, default=20260907)
    parser.add_argument("--release-held-out", action="store_true")
    parser.add_argument("--section", action="append", help="Evaluate only this inventory section; may repeat")
    args = parser.parse_args()
    try:
        manifest = tasks_from(args.tasks, args.release_held_out)
        config = read_json(args.config)
        validate_config(config)
        selected = [task for task in manifest["tasks"] if not args.section or task["section"] in args.section]
        if not selected or args.section and set(args.section) - {task["section"] for task in selected}:
            raise ValueError("unknown or empty section selection")
        planned = schedule(selected, args.repeats, args.seed)
        runs = []
        evidence = {"protocol": 1, "mode": config["mode"], "split": manifest["split"],
                    "performance_claims_supported": False, "completed": False, "planned_runs": len(planned),
                    "seed": args.seed, "repeats": args.repeats, "fresh_context": True,
                    "task_manifest_sha256": digest(args.tasks), "config_sha256": digest(args.config),
                    "environment": {key: config[key] for key in ("backend", "model", "tokenizer", "reasoning", "tools", "permissions", "budget", "accounting_method")},
                    "arms": {arm: {key + "_sha256": digest(config["arms"][arm][key]) for key in ("cli", "skill")} for arm in ARMS},
                    "runs": runs, **aggregate(runs)}
        args.output.parent.mkdir(parents=True, exist_ok=True)
        # Checkpoint each finished attempt before launching another billed model run.
        for planned_run in planned:
            if args.stop_file and args.stop_file.exists():
                evidence.update(stopped=True, cancelled_before_model=len(planned) - len(runs))
                checkpoint = args.output.with_suffix(args.output.suffix + ".tmp")
                checkpoint.write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
                checkpoint.replace(args.output)
                print(json.dumps({"status": "stopped", "completed_attempts": len(runs), "cancelled_before_model": len(planned) - len(runs)}))
                return 0
            runs.append(run_one(config, *planned_run))
            evidence.update({"runs": runs, **aggregate(runs), "completed": len(runs) == len(planned),
                             "performance_claims_supported": config["mode"] == "real" and len(runs) == len(planned)
                             and all(run["usage"] and not run.get("outcome_inconclusive") for run in runs)})
            checkpoint = args.output.with_suffix(args.output.suffix + ".tmp")
            checkpoint.write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
            checkpoint.replace(args.output)
        print(json.dumps({"status": "pass" if all(run["success"] and run["usage"] for run in runs) else "fail",
                          "mode": config["mode"], "runs": len(runs), "output": str(args.output)}))
        return 0 if all(run["success"] and run["usage"] for run in runs) else 1
    except (OSError, ValueError, KeyError, TypeError, AttributeError):
        # Avoid echoing untrusted config/credentials in parse errors.
        print(json.dumps({"status": "error", "category": "invalid_configuration_or_manifest"}), file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
