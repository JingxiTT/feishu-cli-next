#!/usr/bin/env python3
"""Offline process latency and context-size probes; never provider-token usage.

Build the frozen reference with the same catalog as Next. A current upstream
build is a separate, potentially different-catalog comparison. All commands
are local introspection; no authentication or workspace API calls are made.
"""

import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import statistics
import subprocess
import tempfile
import time


METHODS = (
    "calendar.events.get", "im.messages.patch", "drive.files.list",
    "sheets.spreadsheets.get", "task.tasks.get", "mail.user_mailbox.messages.get",
)


def request_contract(data):
    """Only outputSchema is excluded; every other contract field must agree."""
    parsed = json.loads(data)
    envelopes = parsed if isinstance(parsed, list) else [parsed]
    return [{key: value for key, value in envelope.items() if key != "outputSchema"}
            for envelope in envelopes]


def require_success(result, label, args):
    if result.returncode != 0:
        # Do not retain stderr or unexpected content in public evidence.
        raise ValueError(f"{label} {args}: exit {result.returncode}; comparison incomplete")
    return result.stdout, result.stderr


def measure(arms, repetitions, encoding):
    rows = []
    schemas = {}
    with tempfile.TemporaryDirectory(prefix="cli-efficiency-") as directory:
        env = {key: os.environ[key] for key in ("PATH", "SYSTEMROOT", "WINDIR")
               if key in os.environ}
        env.update({"HOME": directory, "USERPROFILE": directory,
                    "LARKSUITE_CLI_CONFIG_DIR": directory,
                    "LARKSUITE_CLI_REMOTE_META": "off",
                    "LARKSUITE_CLI_NO_UPDATE_NOTIFIER": "1",
                    "LARKSUITE_CLI_NO_SKILLS_NOTIFIER": "1", "CI": "1",
                    "HTTP_PROXY": "http://127.0.0.1:9",
                    "HTTPS_PROXY": "http://127.0.0.1:9"})
        cases = [("version", ["--version"])]
        cases += [(method, ["schema", method]) for method in METHODS]
        cases += [("shared-skill", ["skills", "read", "lark-shared"])]
        cases += [("discover-all", ["discover", "*", "--flags", "--limit", "3"]),
                  ("discover-docs", ["discover", "docs", "--limit", "3"]),
                  ("discover-contact-flags", ["discover", "contact open_id", "--flags", "--limit", "3"])]
        for case, args in cases:
            variants = [(label, binary, args) for label, binary in arms.items()
                        if not case.startswith("discover-") or label in ("before", "next")]
            if case in METHODS:
                variants.append(("next-input", arms["next"], args + ["--input-only"]))
            samples = {label: [] for label, _, _ in variants}
            outputs = {}
            # Warm every variant, then rotate order each repetition to avoid
            # always giving one arm the first/cold process in the pair.
            for label, binary, arguments in variants:
                result = subprocess.run([str(binary), *arguments], env=env, cwd=directory,
                                        capture_output=True, timeout=30)
                outputs[label] = require_success(result, label, arguments)
            for trial in range(repetitions):
                offset = trial % len(variants)
                for label, binary, arguments in variants[offset:] + variants[:offset]:
                    start = time.perf_counter_ns()
                    result = subprocess.run([str(binary), *arguments], env=env, cwd=directory,
                                            capture_output=True, timeout=30)
                    duration = (time.perf_counter_ns() - start) / 1_000_000
                    content = require_success(result, label, arguments)
                    if content != outputs[label]:
                        raise ValueError(f"{label} {case}: unstable output; comparison incomplete")
                    samples[label].append(duration)
            if case in METHODS:
                schemas[case] = {label: request_contract(content[0])
                                 for label, content in outputs.items()}
                if not schemas[case]["reference"] == schemas[case]["before"] == schemas[case]["next"] == schemas[case]["next-input"]:
                    raise ValueError(f"{case}: request/permission/guidance contract changed")
                if outputs["before"] != outputs["next"]:
                    raise ValueError(f"{case}: default schema output changed")
            if case.startswith("discover-") and outputs["before"] != outputs["next"]:
                raise ValueError(f"{case}: discovery output changed")
            for label, _, arguments in variants:
                stdout, stderr = outputs[label]
                rows.append({"case": case, "arm": label, "args": arguments,
                             "same_request_contract_as_reference": (
                                 schemas[case][label] == schemas[case]["reference"]
                                 if case in METHODS else None),
                             "median_ms": statistics.median(samples[label]),
                             "samples_ms": samples[label],
                             "stdout_bytes": len(stdout), "stderr_bytes": len(stderr),
                             "stdout_sha256": hashlib.sha256(stdout).hexdigest(),
                             "context_tokens": sum(len(encoding.encode(stream.decode("utf-8"),
                                                                         disallowed_special=()))
                                                   for stream in (stdout, stderr)) if encoding else None})
    return rows


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for label in ("reference", "before", "next", "current"):
        parser.add_argument(f"--{label}-cli", type=Path, required=label != "current")
    parser.add_argument("--repetitions", type=int, default=31)
    parser.add_argument("--encoding", help="optional tiktoken encoding, e.g. o200k_base")
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    if args.repetitions < 3:
        parser.error("at least three repetitions are required")
    arms = {label: path.resolve(strict=True) for label in ("reference", "before", "next", "current")
            if (path := getattr(args, f"{label}_cli")) is not None}
    encoding = None
    tokenizer_version = None
    if args.encoding:
        import tiktoken
        encoding = tiktoken.get_encoding(args.encoding)
        tokenizer_version = tiktoken.__version__
    report = {"measurement": "offline process probes, not agent task or provider usage",
              "platform": platform.platform(), "python": platform.python_version(),
              "encoding": args.encoding, "tiktoken_version": tokenizer_version,
              "repetitions": args.repetitions,
              "binary_sha256": {label: hashlib.sha256(path.read_bytes()).hexdigest()
                                for label, path in arms.items()},
              "rows": measure(arms, args.repetitions, encoding)}
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2) + "\n")
    print(f"Recorded {len(report['rows'])} probes in {args.output}")


if __name__ == "__main__":
    main()
