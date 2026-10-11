#!/usr/bin/env python3
# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Join immutable source hashes to explicit per-path review verdicts; never approve code."""
import argparse
import collections
import hashlib
import io
import json
import pathlib
import subprocess
import tarfile


def digest(data):
    return hashlib.sha256(data).hexdigest()


def review_records(value, pointer="", context=None):
    """Only a per-path verdict is usable; parent summary verdicts are not inherited."""
    if isinstance(value, dict):
        context = dict(context or {})
        for key in ("reviewer", "assignment_id", "reviewed_scope", "limitations", "closes_findings", "historical_finding_ids", "closure_status"):
            if key in value:
                context[key] = value[key]
        if all(key in value for key in ("path", "review_sha256", "review_revision", "verdict")):
            yield {**context, **value, "pointer": pointer or "/"}
        for key, child in value.items():
            yield from review_records(child, pointer + "/" + key.replace("~", "~0").replace("/", "~1"), context)
    elif isinstance(value, list):
        for index, child in enumerate(value):
            yield from review_records(child, pointer + "/" + str(index), context)


def effective_status(current_hash, records):
    if current_hash is None:
        return "missing_from_checkpoint"
    exact = [r for r in records if r["hash_verified"] and r["review_sha256"] == current_hash]
    verdicts = {r["verdict"] for r in exact}
    for objection in (r for r in exact if r["verdict"] == "changes_requested"):
        ids = {f if isinstance(f, str) else f["id"] for f in objection.get("findings", []) if isinstance(f, str) or "id" in f}
        closed = set()
        for approval in exact:
            if approval["verdict"] != "approved_offline" or approval.get("prior_review_sha256") != objection["review_sha256"]:
                continue
            closed.update(f["id"] for f in approval.get("findings", []) if f.get("status") in ("closed_at_review_revision", "closed_offline") and "id" in f)
        if not ids or not ids.issubset(closed):
            return "changes_requested_at_current_hash"
    if "approved_offline" in verdicts:
        return "recorded_approval_at_current_hash"
    if exact:
        return "incomplete_review_at_current_hash"
    return "changed_after_review" if records else "unreviewed"


def category(path):
    if path.startswith(("evaluation/", "evidence/evaluation/", "tests/next_evaluation/")):
        return "evaluation"
    if path.startswith(("docs/", "baseline/", "evidence/")) or path.endswith(".md"):
        return "documentation_evidence"
    if path.startswith("tests/") or path.endswith("_test.go") or "_test." in path:
        return "tests_harness"
    if path.startswith((".github/", "scripts/")) or path in ("Makefile", "build.sh", "package.json", ".goreleaser.yml"):
        return "build_distribution"
    return "runtime_contract"


def generate(root, revision, review_dir):
    def git(*args):
        return subprocess.check_output(["git", "-C", str(root), *args])

    revision = git("rev-parse", revision + "^{commit}").decode().strip()
    # One archive provides an immutable source snapshot without touching any checkout.
    hashes = {}
    with tarfile.open(fileobj=io.BytesIO(git("archive", revision))) as archive:
        for entry in archive:
            if entry.isfile():
                hashes[entry.name] = digest(archive.extractfile(entry).read())
    assignments = json.loads((root / "evidence/reference/p3-review-assignments.json").read_text(encoding="utf-8"))
    initial = {p["path"]: {"assignment": a["id"], **p} for a in assignments["assignments"] for p in a["paths"]}
    frozen = json.loads((root / "baseline/sources.json").read_text(encoding="utf-8"))
    changed_reference = {p for p, h in frozen["files"].items() if hashes.get(p) != h}
    new_runtime = {p for p in hashes if p not in frozen["files"] and p.endswith(".go") and not p.endswith("_test.go") and p.startswith(("cmd/", "internal/", "shortcuts/", "errs/"))}
    later = set(git("diff", "--name-only", assignments["checkpoint"], revision).decode().splitlines())
    excluded_evidence = sorted(p for p in later if p.startswith(("docs/reference/", "baseline/", "evidence/reference/")))
    scope = set(initial) | changed_reference | new_runtime | (later - set(excluded_evidence))
    reviews_revision = git("rev-parse", "HEAD").decode().strip()
    tracked = set(git("ls-tree", "-r", "--name-only", "-z", reviews_revision).decode().strip("\0").split("\0"))
    reports, invalid, records = [], [], collections.defaultdict(list)
    cache = {}
    for file in sorted(review_dir.glob("*.json")):
        relative = str(file.relative_to(root))
        if relative not in tracked:
            continue  # A reviewer may still be writing an untracked draft.
        raw = git("show", reviews_revision + ":" + relative)
        report = json.loads(raw)
        reports.append({"path": relative, "sha256": digest(raw)})
        for row in review_records(report):
            key = (row["review_revision"], row["path"])
            if key not in cache:
                try:
                    cache[key] = digest(git("show", key[0] + ":" + key[1]))
                except subprocess.CalledProcessError:
                    cache[key] = None
            row["hash_verified"] = cache[key] == row["review_sha256"]
            row["report"] = relative
            if not row["hash_verified"]:
                invalid.append({k: row[k] for k in ("report", "pointer", "path", "review_revision", "review_sha256")})
            records[row["path"]].append(row)
    rows = []
    for path in sorted(scope):
        history = records[path]
        rows.append({"path": path, "category": category(path), "initial_assignment": initial.get(path, {}).get("assignment"), "additional_scope": path not in initial, "candidate_sha256": hashes.get(path), "reference_sha256": frozen["files"].get(path), "effective_status": effective_status(hashes.get(path), history), "review_history": [{k: record[k] for k in ("report", "pointer", "reviewer", "assignment_id", "review_revision", "review_sha256", "hash_verified", "verdict", "review_scope", "reviewed_scope", "findings", "prior_review_revision", "prior_review_sha256", "closes_findings", "historical_finding_ids", "closure_status") if k in record} for record in history]})
    counts = collections.Counter(row["effective_status"] for row in rows)
    initial_counts = collections.Counter(row["effective_status"] for row in rows if not row["additional_scope"])
    return {"schema_version": 1, "status": "review_coverage_report_not_acceptance", "candidate_revision": revision, "initial_assignment_checkpoint": assignments["checkpoint"], "method": "Immutable git archive hashes joined only to explicit per-path verdicts in tracked review reports. Every reported hash is verified against its stated committed blob. Historical findings and narrow closure scopes remain in review_history. Old hashes never approve new bytes. Duplicate report copies do not multiply covered paths.", "counts": {"initial_paths": len(initial), "candidate_changed_reference_paths": len(changed_reference), "candidate_new_runtime_go_paths": len(new_runtime), "total_in_scope": len(rows), "additional_scope": sum(row["additional_scope"] for row in rows), "initial_effective_status": dict(initial_counts), "all_effective_status": dict(counts)}, "review_input_revision": reviews_revision, "review_inputs": reports, "invalid_review_hashes": invalid, "paths": rows, "excluded_inventory_evidence": excluded_evidence, "working_scope_when_written": git("status", "--porcelain=v1").decode().strip(), "limits": ["Recorded approval means reviewer-declared offline semantics only, with that record's scope and dependencies; it is not fresh test execution or a release verdict.", "Later runtime/build/test/documentation/evaluation changes are additional scope, not automatically accepted by initial source reviews.", "Untracked review drafts are excluded. Any subsequent source or review change requires regenerating this report.", "The required live, platform, release and model-evaluation gates remain separate; no checker whitelist is created."]}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--target", type=pathlib.Path, default=pathlib.Path("."))
    parser.add_argument("--revision", required=True)
    parser.add_argument("--out", type=pathlib.Path, required=True)
    args = parser.parse_args()
    root = args.target.resolve()
    report = generate(root, args.revision, root / "evidence/reference/reviews")
    args.out.write_text(json.dumps(report, ensure_ascii=False, sort_keys=True, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(report["counts"], sort_keys=True))
    return 1 if report["invalid_review_hashes"] else 0


if __name__ == "__main__":
    raise SystemExit(main())
