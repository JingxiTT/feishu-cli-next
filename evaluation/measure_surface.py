#!/usr/bin/env python3
"""Exhaustive local introspection comparison against the frozen CLI inventory.

Every reference command, method, event key and embedded guidance file is named
individually. This measures help/schema/guidance, never live functionality or
provider tokens. Unsupported/failed cases remain visible and have no score.
"""
import argparse
from collections import Counter
import gzip
import hashlib
from itertools import product
import json
from pathlib import Path, PurePosixPath
import platform
import subprocess
import sys
import tempfile

from measure_cli import isolated_environment, observation, probe_variants, request_contract

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "scripts"))
from next_inventory import metadata_methods


def read(path):
    raw = path.read_bytes()
    return json.loads(gzip.decompress(raw) if path.suffix == ".gz" else raw)


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def inventory_cases(coverage, registrations, methods, brand):
    surface = registrations[brand]
    if set(coverage["commands"]) != set(surface["commands"]):
        raise ValueError("coverage/registration command inventory differs")
    if not set(methods) <= set(surface["commands"]):
        raise ValueError("generated method missing from reference command inventory")
    cases = []
    for path, contract in sorted(surface["commands"].items()):
        section = coverage["commands"][path]["section"]
        # A guarded command can bypass Cobra flag parsing: --help is then not
        # a read-only help request. Account for it without dispatching RunE.
        cases.append({"id": "help:" + path, "kind": "help", "section": section,
                      "capability": path, "args": [*path.split(), "--help"],
                      "skip": "brand_guard_bypasses_help" if contract.get("disable_flag_parsing") else None})
        # Groups also have RunE guards, so Runnable/Source cannot distinguish
        # a method from a service/resource. Use the canonical frozen catalog.
        if path in methods:
            cases.append({"id": "schema:" + path, "kind": "schema", "section": section,
                          "capability": path, "args": ["schema", *path.split()]})
        words = path.split()
        alternatives = [[word, *(surface["commands"][" ".join(words[:i + 1])]["aliases"] or [])]
                        for i, word in enumerate(words)]
        for spelling in product(*alternatives):
            if list(spelling) != words:
                cases.append({"id": "alias-help:" + " ".join(spelling), "kind": "alias-help", "section": section,
                              "capability": path, "args": [*spelling, "--help"],
                              "skip": "brand_guard_bypasses_help" if contract.get("disable_flag_parsing") else None})
    for key, event in sorted(surface["events"].items()):
        cases.append({"id": "event:" + key, "kind": "event", "section": "events",
                      "capability": key, "args": ["event", "schema", key, "--json"]})
    cases.append({"id": "version", "kind": "version", "section": "foundation",
                  "capability": "version", "args": ["--version"]})
    return cases


def guidance_cases(binary, env, directory):
    """Enumerate the actual embedded filesystem, not a maintained skill sample."""
    def listing(path=None):
        args = [str(binary), "skills", "list", *([path] if path else [])]
        result = subprocess.run(args, env=env, cwd=directory, capture_output=True, timeout=30)
        if result.returncode:
            raise ValueError("reference embedded guidance enumeration failed")
        return json.loads(result.stdout)
    cases = []
    skills = listing()["skills"]
    for skill in skills:
        name = skill["name"]
        stack = [name]
        seen = set()
        while stack:
            parent = stack.pop()
            if parent in seen:
                raise ValueError("cyclic guidance listing")
            seen.add(parent)
            for entry in listing(parent)["entries"]:
                path = entry["path"]
                if (not isinstance(path, str) or PurePosixPath(path).is_absolute()
                        or ".." in PurePosixPath(path).parts
                        or str(PurePosixPath(path)) != path
                        or str(PurePosixPath(path).parent) != parent):
                    raise ValueError("invalid embedded guidance path")
                if entry["is_dir"]:
                    stack.append(path)
                else:
                    cases.append({"id": "guidance:" + path, "kind": "guidance",
                                  "section": "guidance", "capability": path,
                                  "args": ["skills", "read", path]})
    return sorted(cases, key=lambda case: case["id"])


def compare_case(case, arms, repetitions, encoding, env, directory):
    variants = [(label, binary, case["args"]) for label, binary in arms.items()]
    if case["kind"] == "schema":
        variants.append(("next-input", arms["next"], case["args"] + ["--input-only"]))
    outputs, samples, failures = probe_variants(variants, repetitions, env, directory, tolerate_failures=True)
    result = {**case, "arms": {}, "contracts": {}}
    for label, _, args in variants:
        if label in failures:
            result["arms"][label] = {"status": "unavailable", "reason": failures[label], "args": args}
        else:
            result["arms"][label] = {"status": "measured", **observation(label, args, outputs[label], samples[label], encoding)}
    required = ("reference", "before", "next")
    result["required_probes_pass"] = all(label not in failures for label in required)
    if case["kind"] == "schema":
        required += ("next-input",)
        result["required_probes_pass"] = all(label not in failures for label in required)
        if result["required_probes_pass"]:
            contracts = {label: request_contract(outputs[label][0]) for label in required}
            result["contracts"]["request_permission_guidance_parity"] = all(
                contract == contracts["reference"] for contract in contracts.values())
            result["contracts"]["default_bytes_unchanged"] = outputs["before"] == outputs["next"]
            if "current" in outputs and "current" not in failures:
                result["contracts"]["current_request_contract_matches_reference"] = (
                    request_contract(outputs["current"][0]) == contracts["reference"])
    elif case["kind"] == "event" and result["required_probes_pass"]:
        result["contracts"]["event_definition_parity"] = (
            json.loads(outputs["reference"][0]) == json.loads(outputs["before"][0]) == json.loads(outputs["next"][0]))
    return result


def summarize(cases, sections):
    by_kind = {}
    for kind in sorted({case["kind"] for case in cases}):
        rows = [case for case in cases if case["kind"] == kind]
        comparison = "next-input" if kind == "schema" else "next"
        measured = [case for case in rows if case.get("required_probes_pass")]
        pairs = [(case["arms"]["reference"], case["arms"][comparison]) for case in measured]
        token_pairs = [(a["context_tokens"], b["context_tokens"]) for a, b in pairs
                       if a["context_tokens"] is not None and b["context_tokens"] is not None]
        by_kind[kind] = {"planned": len(rows), "measured": len(measured),
                         "skipped": sum(bool(case.get("skip")) for case in rows),
                         "context_smaller": sum(b < a for a, b in token_pairs),
                         "context_equal": sum(b == a for a, b in token_pairs),
                         "context_larger": sum(b > a for a, b in token_pairs),
                         "reference_context_tokens": sum(a for a, _ in token_pairs) if token_pairs else None,
                         "next_context_tokens": sum(b for _, b in token_pairs) if token_pairs else None,
                         "lower_observed_median_ms": sum(b["median_ms"] < a["median_ms"] for a, b in pairs)}
    return {"by_kind": by_kind,
            "by_section": {section: dict(Counter(case["kind"] for case in cases if case["section"] == section))
                           for section in sections},
            "required_probe_failures": sum(not case.get("required_probes_pass", False) and not case.get("skip") for case in cases),
            "contract_failures": sum(any(value is False for key, value in case.get("contracts", {}).items()
                                         if not key.startswith("current_")) for case in cases),
            "functional_and_provider_efficiency": "unverified; local introspection is not task execution",
            "non_command_efficiency": {"extensions": "unmeasured", "packaging": "unmeasured"}}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for label in ("reference", "before", "next", "current"):
        parser.add_argument(f"--{label}-cli", type=Path, required=label != "current")
    parser.add_argument("--reference-dir", type=Path, default=ROOT / "docs/reference")
    parser.add_argument("--brand", choices=("feishu", "lark"), default="feishu")
    parser.add_argument("--repetitions", type=int, default=3)
    parser.add_argument("--encoding")
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    if args.repetitions < 3:
        parser.error("at least three repetitions are required")
    if args.output.exists():
        parser.error("retain prior measurements; use a new output path")
    arms = {label: path.resolve(strict=True) for label in ("reference", "before", "next", "current")
            if (path := getattr(args, f"{label}_cli")) is not None}
    coverage = read(args.reference_dir / "coverage.json")
    registration_path = args.reference_dir / "configured-brand-registrations.json.gz"
    registrations = read(registration_path)
    metadata_path = args.reference_dir / "meta_data.json.gz"
    methods = metadata_methods(read(metadata_path))
    encoding, tokenizer_version = None, None
    if args.encoding:
        import tiktoken
        encoding = tiktoken.get_encoding(args.encoding)
        tokenizer_version = tiktoken.__version__
    with tempfile.TemporaryDirectory(prefix="cli-surface-") as directory:
        env = isolated_environment(directory, args.brand)
        # Bind production brand selection to a synthetic account. No method
        # executes: help, schema and embedded guidance need no API requests.
        env.update(LARKSUITE_CLI_APP_ID="inventory-synthetic-app", LARKSUITE_CLI_APP_SECRET="inventory-synthetic-secret")
        cases = inventory_cases(coverage, registrations, methods, args.brand)
        cases += guidance_cases(arms["reference"], env, directory)
        results = []
        for index, case in enumerate(cases):
            if case.get("skip"):
                results.append({**case, "arms": {}, "required_probes_pass": False})
            else:
                results.append(compare_case(case, arms, args.repetitions, encoding, env, directory))
            if (index + 1) % 50 == 0:
                print(f"Compared {index + 1}/{len(cases)} local surfaces", flush=True)
    report = {"schema": 1, "measurement": "exhaustive local introspection, not live operations or provider usage",
              "brand": args.brand, "platform": platform.platform(),
              "encoding": args.encoding, "tiktoken_version": tokenizer_version,
              "repetitions": args.repetitions, "reference_revision": coverage["reference"]["revision"],
              "registration_sha256": sha(registration_path), "coverage_sha256": sha(args.reference_dir / "coverage.json"),
              "metadata_sha256": sha(metadata_path),
              "runner_sha256": sha(Path(__file__)), "probe_runner_sha256": sha(ROOT / "evaluation/measure_cli.py"),
              "binary_sha256": {label: sha(path) for label, path in arms.items()},
              "summary": summarize(results, coverage["sections"] + ["guidance"]), "cases": results}
    args.output.parent.mkdir(parents=True, exist_ok=True)
    raw = (json.dumps(report, indent=2) + "\n").encode()
    args.output.write_bytes(gzip.compress(raw, mtime=0) if args.output.suffix == ".gz" else raw)
    print(json.dumps(report["summary"], indent=2))
    return int(bool(report["summary"]["required_probe_failures"] or report["summary"]["contract_failures"]))


if __name__ == "__main__":
    raise SystemExit(main())
