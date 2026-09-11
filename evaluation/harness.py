#!/usr/bin/env python3
"""Balanced CLI agent evaluation. Python standard library only."""

import argparse
from contextlib import contextmanager
import fcntl
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
BEFORE_AFTER_ARMS = ("official", "next_before", "next")
ORIGINAL_ARMS = ("official", "next")


def comparison_arms(config):
    names = set(config.get("arms", {}))
    for arms in (ARMS, BEFORE_AFTER_ARMS, ORIGINAL_ARMS):
        if names == set(arms):
            return arms
    raise ValueError("a recognized comparison arm set is required")


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


def schedule(tasks, repeats, seed, arms=ARMS):
    if repeats < 2:
        raise ValueError("at least two independent trials per task and arm are required")
    runs = [(task, trial, arm) for task in tasks for trial in range(repeats) for arm in arms]
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


def guidance_delivery(config):
    delivery = config.get("guidance_delivery", "inline")
    if delivery not in ("inline", "file") or any("guidance_delivery" in arm for arm in config.get("arms", {}).values()):
        raise ValueError("guidance_delivery must be one shared inline or file mode")
    return delivery


def validate_config(config):
    if not isinstance(config, dict):
        raise ValueError("config must be an object")
    guidance_delivery(config)
    for key in ("backend", "model", "tokenizer", "reasoning", "tools", "permissions", "budget", "accounting_method"):
        if not config.get(key):
            raise ValueError("pin config field: " + key)
    if config.get("mode") not in ("controlled", "real"):
        raise ValueError("mode must be controlled or real")
    comparison_arms(config)
    for key in ("runner", "verifier"):
        command = config.get(key)
        if not isinstance(command, list) or not command or not all(isinstance(arg, str) for arg in command):
            raise ValueError(key + " must be a nonempty argv array")
        if not Path(command[0]).is_absolute():
            raise ValueError(key + " executable must be absolute")
    if config["runner"] == config["verifier"]:
        raise ValueError("verifier must be independent of agent runner")
    official = config["arms"]["official"]["cli"]
    if "official_concise" in config["arms"] and official != config["arms"]["official_concise"]["cli"]:
        raise ValueError("official arms must use the identical CLI")
    if type(config.get("timeout_seconds", 120)) not in (int, float) or config.get("timeout_seconds", 120) <= 0:
        raise ValueError("timeout_seconds must be positive")
    for arm in config["arms"].values():
        for key in ("cli", "skill"):
            if not Path(arm[key]).is_absolute() or not Path(arm[key]).is_file():
                raise ValueError("each arm requires an absolute existing " + key + " file")


def run_one(config, task, trial, arm):
    delivery = guidance_delivery(config)
    with tempfile.TemporaryDirectory(prefix="feishu-eval-") as directory:
        workspace = Path(directory)
        (workspace / "fixture.json").write_text(json.dumps(task["fixture"], ensure_ascii=False), encoding="utf-8")
        request = {"protocol": 1, "task_id": task["id"], "trial": trial, "arm": arm,
                   "prompt": task["prompt"], "fixture": task["fixture"],
                   "workspace": directory, "fresh_context": True,
                   "cli": config["arms"][arm]["cli"], "skill": config["arms"][arm]["skill"],
                   "settings": {key: config[key] for key in
                                ("backend", "model", "tokenizer", "reasoning", "tools", "permissions", "budget", "mode")}}
        request["settings"]["guidance_delivery"] = delivery
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
                "accounting_diagnostics": sanitized_accounting_diagnostics(payload.get("accounting_diagnostics")),
                "verification_failures": sanitized_verification_failures(verification.get("verification_failures")),
                "api_request_trace": sanitized_api_trace(payload.get("api_request_trace"), task),
                "control_trace": sanitized_trace(payload.get("control_trace")),
                "usage_granularity": payload.get("usage_granularity") if payload.get("usage_granularity") in ("model_turn", "codex_user_turn") else None,
                "runner_exit": code, "verifier_exit": verify_code,
                "latency_seconds": elapsed, "verification_seconds": verify_elapsed, **metrics}



def sanitized_accounting_diagnostics(values):
    allowed = {"provider_event_invalid", "provider_turn_failed", "provider_turn_count_mismatch", "unexpected_provider_tool"}
    return sorted({value for value in values if isinstance(value, str) and value in allowed}) if isinstance(values, list) else None


def sanitized_verification_failures(values):
    from scenario_verifier import SCENARIO_DIAGNOSTICS
    from section_fixture import UNSUPPORTED_REASON_CODES
    allowed = {"answer_mismatch", "scenario_mismatch", "missing_request_evidence", "request_outside_fixture",
               "required_reads_missing", "config_changed", "invalid_verification_input",
               "packaging_answer_path_mismatch"} | SCENARIO_DIAGNOSTICS | UNSUPPORTED_REASON_CODES
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
    from codex_runner import trace_vocabulary, sanitized_cli_error
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
        record = {"choices": choices, "exit_code": code if type(code) is int else None}
        if type(code) is int and code != 0 and len(choices) == 1 and choices[0]["kind"] == "cli":
            diagnostic = sanitized_cli_error(event.get("reported_cli_error"), choices[0]["command"])
            if diagnostic:
                record["reported_cli_error"] = diagnostic
        result.append(record)
    return result


def aggregate(runs, arms=ARMS):
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
    for baseline in arms[:-1]:
        pairs = []
        for run in runs:
            other = indexed.get((run["task_id"], run["trial"], baseline))
            if run["arm"] == "next" and other and run["success"] and other["success"] and run["usage"] and other["usage"]:
                pairs.append({"task_id": run["task_id"], "trial": run["trial"],
                              "baseline_tokens": other["usage"]["total_tokens"], "next_tokens": run["usage"]["total_tokens"]})
        matched.append({"baseline": baseline, "pairs": pairs})
    return {"groups": rows, "matched_success": matched}


class ResumeError(ValueError):
    """A fixed, payload-free reason why launching another trial is unsafe."""


def json_digest(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def sync_directory(path):
    descriptor = os.open(path, os.O_RDONLY)
    try:
        os.fsync(descriptor)
    finally:
        os.close(descriptor)


def atomic_json(path, value):
    temporary = None
    try:
        with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8", dir=path.parent,
                                         prefix=path.name + ".", suffix=".tmp", delete=False) as stream:
            temporary = Path(stream.name)
            stream.write(json.dumps(value, indent=2) + "\n")
            stream.flush()
            os.fsync(stream.fileno())
        temporary.replace(path)
        sync_directory(path.parent)
    finally:
        if temporary is not None:
            temporary.unlink(missing_ok=True)


@contextmanager
def output_lock(path):
    # Keep the lock file: unlinking it permits two processes to lock different
    # inodes for the same output. flock itself is released when a process dies.
    with path.with_name(path.name + ".lock").open("a") as stream:
        try:
            fcntl.flock(stream.fileno(), fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            raise ResumeError("output_locked") from None
        try:
            yield
        finally:
            fcntl.flock(stream.fileno(), fcntl.LOCK_UN)


def tree_digest(path):
    path = Path(path)
    return json_digest([(str(item.relative_to(path)), digest(item))
                        for item in sorted(path.rglob("*")) if item.is_file() and "__pycache__" not in item.parts])


def run_identity(planned_run):
    task, trial, arm = planned_run
    return {"task_id": task["id"], "section": task["section"], "trial": trial, "arm": arm}


def resume_contract(config, evidence, planned):
    # Hash public executable inputs, never auth-home directory contents. File
    # paths may be ephemeral, but bytes, flags and non-file arguments must match.
    normalized = json.loads(json.dumps(config))
    normalized["timeout_seconds"] = config.get("timeout_seconds", 120)
    normalized["guidance_delivery"] = guidance_delivery(config)
    for name in ("runner", "verifier"):
        normalized[name] = [{"file_sha256": digest(arg)} if Path(arg).is_absolute() and Path(arg).is_file() else arg
                            for arg in config[name]]
    for name, arm in normalized["arms"].items():
        for key in ("cli", "skill"):
            arm[key] = {"sha256": digest(config["arms"][name][key])}
        entry = Path(config["arms"][name]["skill"])
        arm["guidance_tree_sha256"] = tree_digest(entry.parent)
        if (entry.parent.parent / "lark-shared").is_dir():
            arm["linked_skills"] = {child.name: tree_digest(child) for child in sorted(entry.parent.parent.iterdir())
                                   if child.is_dir() and child.name.startswith("lark-")}
        if "selected_skill_source" in arm:
            arm["selected_skill_source"] = {"tree_sha256": tree_digest(arm["selected_skill_source"])}
    fixed = {key: evidence.get(key) for key in ("protocol", "mode", "split", "planned_runs", "seed", "repeats",
        "fresh_context", "task_manifest_sha256", "environment", "arms", "arm_selection")}
    sources = {str(path.relative_to(Path(__file__).parent)): digest(path)
               for path in sorted(Path(__file__).parent.rglob("*.py"))}
    return {"version": 1, "sha256": json_digest({"fixed": fixed, "config": normalized,
        "evaluator_sources": sources, "schedule": [run_identity(run) for run in planned]})}


def resumed_evidence(path, expected, planned, arms):
    previous = read_json(path)
    if previous.get("resume_contract") != expected["resume_contract"]:
        raise ResumeError("resume_contract_mismatch")
    for key in ("protocol", "mode", "split", "planned_runs", "seed", "repeats", "fresh_context",
                "task_manifest_sha256", "environment", "arms", "arm_selection"):
        if previous.get(key) != expected.get(key):
            raise ResumeError("resume_contract_mismatch")
    runs = previous.get("runs")
    if not isinstance(runs, list) or len(runs) > len(planned):
        raise ResumeError("invalid_resume_checkpoint")
    for row, planned_run in zip(runs, planned):
        identity = run_identity(planned_run)
        if (not isinstance(row, dict) or type(row.get("trial")) is not int or
                any(row.get(key) != value for key, value in identity.items()) or
                any(key not in row for key in ("success", "outcome_inconclusive", "usage", "categories", "runner_exit",
                    "verifier_exit", "latency_seconds", "verification_seconds", "api_calls", "retries", "command_executions")) or
                type(row["success"]) is not bool or type(row["outcome_inconclusive"]) is not bool):
            raise ResumeError("invalid_resume_checkpoint")
    if type(previous.get("completed")) is not bool or previous["completed"] != (len(runs) == len(planned)):
        raise ResumeError("invalid_resume_checkpoint")
    claims = (previous.get("arm_selection", {}).get("mode") != "candidate_only" and previous["mode"] == "real"
              and len(runs) == len(planned) and all(row["usage"] and not row["outcome_inconclusive"] for row in runs))
    if type(previous.get("performance_claims_supported")) is not bool or previous["performance_claims_supported"] != claims:
        raise ResumeError("invalid_resume_checkpoint")
    summaries = aggregate(runs, arms)
    if any(previous.get(key) != value for key, value in summaries.items()):
        raise ResumeError("invalid_resume_checkpoint")
    return previous


def reconcile_marker(marker, evidence, planned):
    if not marker.exists():
        return
    value = read_json(marker)
    index = value.get("index")
    if (type(index) is not int or not 0 <= index < len(planned) or
            value != {"resume_contract": evidence["resume_contract"], "index": index,
                      "run": run_identity(planned[index])}):
        raise ResumeError("invalid_inflight_marker")
    if index >= len(evidence["runs"]):
        # It is impossible to know whether billing happened before interruption.
        raise ResumeError("interrupted_run_requires_review")
    marker.unlink()
    sync_directory(marker.parent)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tasks", type=Path, required=True)
    parser.add_argument("--config", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--repeats", type=int, default=3)
    parser.add_argument("--resume", action="store_true", help="Reuse the exact completed run prefix; never retry an unresolved in-flight attempt")
    parser.add_argument("--stop-file", type=Path, help="Operator stop request, checked only between complete attempts")
    parser.add_argument("--seed", type=int, default=20260907)
    parser.add_argument("--release-held-out", action="store_true")
    parser.add_argument("--section", action="append", help="Evaluate only this inventory section; may repeat")
    parser.add_argument("--only-arm", choices=("next",), help="Run only the candidate entries from the frozen balanced schedule; not a new balanced comparison")
    parser.add_argument("--original-comparison", action="store_true", help="Run a fresh balanced comparison of only the original CLI and Next")
    args = parser.parse_args()
    try:
        manifest = tasks_from(args.tasks, args.release_held_out)
        config = read_json(args.config)
        validate_config(config)
        selected = [task for task in manifest["tasks"] if not args.section or task["section"] in args.section]
        if not selected or args.section and set(args.section) - {task["section"] for task in selected}:
            raise ValueError("unknown or empty section selection")
        arms = comparison_arms(config)
        if args.original_comparison != (arms == ORIGINAL_ARMS) or args.original_comparison and args.only_arm:
            raise ValueError("original comparison requires exactly original and Next arms and cannot select only one arm")
        planned = schedule(selected, args.repeats, args.seed, arms)
        selected_arms = arms
        if args.only_arm:
            planned = [run for run in planned if run[2] == args.only_arm]
            selected_arms = (args.only_arm,)
        runs = []
        evidence = {"protocol": 1, "mode": config["mode"], "split": manifest["split"],
                    "performance_claims_supported": False, "completed": False, "planned_runs": len(planned),
                    "seed": args.seed, "repeats": args.repeats, "fresh_context": True,
                    "task_manifest_sha256": digest(args.tasks), "config_sha256": digest(args.config),
                    "environment": {**{key: config[key] for key in ("backend", "model", "tokenizer", "reasoning", "tools", "permissions", "budget", "accounting_method")},
                                    "guidance_delivery": guidance_delivery(config)},
                    "arms": {arm: {key + "_sha256": digest(config["arms"][arm][key]) for key in ("cli", "skill")} for arm in arms},
                    "runs": runs, **aggregate(runs, selected_arms)}
        if args.only_arm:
            evidence["arm_selection"] = {"mode": "candidate_only", "selected": list(selected_arms),
                                         "schedule": "filtered_balanced", "balanced_comparison": False}
        elif args.original_comparison:
            evidence["arm_selection"] = {"mode": "original_comparison", "selected": list(selected_arms),
                                         "schedule": "balanced", "balanced_comparison": True}
        evidence["resume_contract"] = resume_contract(config, evidence, planned)
        args.output = args.output.resolve()
        args.output.parent.mkdir(parents=True, exist_ok=True)
        marker = args.output.with_name(args.output.name + ".inflight.json")
        with output_lock(args.output):
            if args.output.exists():
                if not args.resume:
                    raise ResumeError("output_exists_use_resume")
                evidence = resumed_evidence(args.output, evidence, planned, selected_arms)
                runs = evidence["runs"]
            elif marker.exists():
                raise ResumeError("interrupted_run_requires_review")
            else:
                atomic_json(args.output, evidence)
            reconcile_marker(marker, evidence, planned)
            # Every terminal result is reused, including failures, unsupported
            # outcomes and unknown usage. Resume does not rescore or cherry-pick.
            for index in range(len(runs), len(planned)):
                if args.stop_file and args.stop_file.exists():
                    evidence.update(stopped=True, cancelled_before_model=len(planned) - len(runs))
                    atomic_json(args.output, evidence)
                    print(json.dumps({"status": "stopped", "completed_attempts": len(runs), "cancelled_before_model": len(planned) - len(runs)}))
                    return 0
                evidence.pop("stopped", None)
                evidence.pop("cancelled_before_model", None)
                atomic_json(marker, {"resume_contract": evidence["resume_contract"], "index": index,
                                     "run": run_identity(planned[index])})
                runs.append(run_one(config, *planned[index]))
                evidence.update({"runs": runs, **aggregate(runs, selected_arms), "completed": len(runs) == len(planned),
                                 "performance_claims_supported": not args.only_arm and config["mode"] == "real" and len(runs) == len(planned)
                                 and all(run["usage"] and not run.get("outcome_inconclusive") for run in runs)})
                atomic_json(args.output, evidence)
                marker.unlink()
                sync_directory(marker.parent)
        print(json.dumps({"status": "pass" if all(run["success"] and run["usage"] for run in runs) else "fail",
                          "mode": config["mode"], "runs": len(runs), "output": str(args.output)}))
        return 0 if all(run["success"] and run["usage"] for run in runs) else 1
    except ResumeError as error:
        print(json.dumps({"status": "error", "category": str(error)}), file=sys.stderr)
        return 2
    except (OSError, ValueError, KeyError, TypeError, AttributeError):
        # Avoid echoing untrusted config/credentials in parse errors.
        print(json.dumps({"status": "error", "category": "invalid_configuration_or_manifest"}), file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
