#!/usr/bin/env python3
# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT
"""Run section regressions without inheriting developer credentials or state.

This is environment isolation, not an OS network sandbox. HTTP(S) clients that
honor proxy settings cannot reach external services; mock/local transports still
work. Live tests are deliberately unavailable until fixtures verify the plan's
account, tenant, visibility, recipient, and cleanup boundaries.
"""

import argparse
import json
import os
import re
from pathlib import Path
import signal
import socket
import subprocess
import sys
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]
DOMAIN_DIRS = {"docs": "doc", "events": "event"}
E2E_DIRS = {"events": "event"}
DOMAINS = tuple(sorted(p.name for p in (ROOT / "shortcuts").iterdir()
                       if p.is_dir() and p.name != "common" and p.name not in ("doc", "event"))) + ("docs", "events", "business",)


def isolated_env(directory, parent, proxy):
    """Inherit tool locations only. No TEST_*, CLI_*, auth, or loader settings."""
    base = Path(directory)
    env = {key: parent[key] for key in ("PATH", "SYSTEMROOT", "WINDIR", "COMSPEC",
                                       "PATHEXT", "SDKROOT", "DEVELOPER_DIR") if key in parent}
    for key, name in (("HOME", "home"), ("USERPROFILE", "home"),
                      ("XDG_CONFIG_HOME", "config"), ("XDG_STATE_HOME", "state"),
                      ("XDG_CACHE_HOME", "cache"), ("APPDATA", "config"),
                      ("LOCALAPPDATA", "state"), ("TMPDIR", "tmp"),
                      ("TMP", "tmp"), ("TEMP", "tmp"),
                      ("LARKSUITE_CLI_CONFIG_DIR", "cli-config")):
        path = base / name
        path.mkdir(parents=True, exist_ok=True)
        env[key] = str(path)
    # Reuse compiled dependencies, never the original home/config itself. Go's
    # persistent environment file and automatic toolchain downloads stay off.
    original_home = Path(parent.get("HOME", parent.get("USERPROFILE", str(base))))
    default_cache = (original_home / "Library/Caches/go-build" if sys.platform == "darwin"
                     else original_home / ".cache/go-build")
    env.update({"GOCACHE": parent.get("GOCACHE", str(default_cache)),
                "GOMODCACHE": parent.get("GOMODCACHE", str(original_home / "go/pkg/mod")),
                "GOPATH": str(base / "go"), "GOENV": "off", "GOTOOLCHAIN": "local",
                "GOPROXY": "off", "GOSUMDB": "off", "GOWORK": "off",
                "LARKSUITE_CLI_REMOTE_META": "off",
                "LARKSUITE_CLI_NO_UPDATE_NOTIFIER": "1",
                "LARKSUITE_CLI_NO_SKILLS_NOTIFIER": "1",
                "CI": "1",
                "PYTHONNOUSERSITE": "1", "PYTHONDONTWRITEBYTECODE": "1",
                "GIT_CONFIG_NOSYSTEM": "1", "GIT_TERMINAL_PROMPT": "0"})
    for name in ("HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"):
        env[name] = proxy
    env["NO_PROXY"] = env["no_proxy"] = "localhost,127.0.0.1,::1"
    return env


def section_suites(section):
    """List next-interface suites separately from inherited CI regressions."""
    if section == "foundation":
        return []
    root = ROOT / "tests/next_sections"
    sections = {section}
    if section == "all":
        sections = set(DOMAINS) | {"extensions", "packaging"}
        sections.update(path.name for path in root.glob("*") if path.is_dir() and not path.name.startswith("."))
    return [{"section": name, "package": "./tests/next_sections/" + name,
             "status": "present" if any((root / name).glob("*_test.go")) else "missing"}
            for name in sorted(sections)]


def integration_name(value):
    if not re.fullmatch(r"[a-z][a-z0-9_]*", value):
        raise argparse.ArgumentTypeError("integration must be one lowercase directory name")
    return value


def integration_suite(name):
    package = "./tests/next_integration/" + name
    return {"section": name, "package": package,
            "status": "present" if any((ROOT / package).glob("*_test.go")) else "missing"}


def commands_for(section, suite):
    commands = []
    go_test = ["go", "test", "-json", "-count=1", "-timeout=10m"]
    if suite in ("unit", "all"):
        if section == "all":
            packages = ["./cmd/...", "./internal/...", "./shortcuts/...", "./extension/...", "./events/...", "./errs/..."]
        elif section == "foundation":
            packages = ["./cmd/...", "./internal/...", "./shortcuts/common/...", "./extension/...", "./errs/..."]
        elif section == "extensions":
            packages = ["./extension/..."]
        elif section == "packaging":
            packages = []
        elif section == "business":
            packages = ["./cmd/service/...", "./internal/apicatalog/...", "./internal/registry/...",
                        "./internal/schema/...", "./internal/client/...", "./internal/nextattempt/..."]
        else:
            packages = ["./shortcuts/" + DOMAIN_DIRS.get(section, section) + "/..."]
            if section == "events":
                packages.extend(["./events/...", "./cmd/event/...", "./internal/event/..."])
            elif (ROOT / "events" / section).is_dir():
                packages.append("./events/" + section + "/...")
        if packages:
            commands.append(go_test + packages)
    if suite in ("e2e", "all"):
        if section in ("foundation", "all"):
            commands.append([sys.executable, "-m", "unittest", "discover", "-s", "tests/next_foundation", "-p", "*_test.py", "-v"])
            commands.append(go_test + ["./tests/next_foundation", "./tests/cli_e2e", "./tests/cli_e2e/config"])
        if section in ("extensions", "all"):
            commands.append(go_test + ["./tests/plugin_e2e"])
        # Preserve the official CI's explicit deterministic selector; running
        # every test in these packages would also select live workflow tests.
        domains = sorted(DOMAINS) if section == "all" else [section]
        packages = ["./tests/cli_e2e/" + E2E_DIRS.get(domain, domain) for domain in domains
                    if domain != "foundation" and (ROOT / "tests/cli_e2e" / E2E_DIRS.get(domain, domain)).is_dir()]
        if section == "all":
            packages.append("./tests/cli_e2e/dryrun")
        if packages:
            commands.append(go_test + packages + ["-run", "DryRun|Regression"])
        next_packages = [item["package"] for item in section_suites(section) if item["status"] == "present"]
        if next_packages:
            commands.append(go_test + next_packages)
    return commands


def selection_evidence(command):
    """Expose test omissions; an offline pass is not complete live coverage."""
    if "-run" not in command:
        return {"policy": "all tests in the listed packages"}
    selector = command[command.index("-run") + 1]
    selected, excluded = [], []
    for package in command:
        if not package.startswith("./tests/cli_e2e/"):
            continue
        for path in sorted((ROOT / package).glob("*_test.go")):
            for name in re.findall(r"^func (Test\w+)\(", path.read_text(), re.MULTILINE):
                entry = {"file": str(path.relative_to(ROOT)), "test": name}
                (selected if re.search(selector, name) else excluded).append(entry)
    return {"policy": "official CI deterministic name selector; excluded tests remain unverified",
            "selector": selector, "selected_tests": selected, "excluded_tests": excluded}


def run_command(command, env, timeout, cwd=ROOT):
    started = time.monotonic()
    try:
        process = subprocess.Popen(command, cwd=cwd, env=env, stdout=subprocess.PIPE,
                                   stderr=subprocess.PIPE, text=True, start_new_session=os.name == "posix")
    except OSError as error:
        return {"command": command, "status": "fail", "exit_code": 127,
                "duration_seconds": 0, "stdout": "", "stderr": str(error), "tests": {}}
    timed_out = False
    try:
        stdout, stderr = process.communicate(timeout=timeout)
    except subprocess.TimeoutExpired:
        timed_out = True
        if os.name == "posix":
            os.killpg(process.pid, signal.SIGKILL)
        else:
            process.kill()
        stdout, stderr = process.communicate()
    code = 124 if timed_out else process.returncode
    counts = {"pass": 0, "fail": 0, "skip": 0}
    for line in stdout.splitlines():
        try:
            event = json.loads(line)
        except ValueError:
            continue
        if isinstance(event, dict) and event.get("Test") and event.get("Action") in counts:
            counts[event["Action"]] += 1
    if not any(counts.values()):
        ran = re.search(r"^Ran (\d+) tests?\b", stderr, re.MULTILINE)
        summary = re.search(r"^(?:OK|FAILED)(?: \(([^)]*)\))?$", stderr, re.MULTILINE)
        if ran and summary:
            totals = dict(re.findall(r"(failures|errors|skipped)=(\d+)", summary.group(1) or ""))
            counts["fail"] = int(totals.get("failures", 0)) + int(totals.get("errors", 0))
            counts["skip"] = int(totals.get("skipped", 0))
            counts["pass"] = max(0, int(ran.group(1)) - counts["fail"] - counts["skip"])
    return {"command": command, "status": "timeout" if timed_out else "pass" if code == 0 else "fail",
            "exit_code": code, "duration_seconds": round(time.monotonic() - started, 3),
            "stdout": stdout, "stderr": stderr, "tests": counts}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    selection = parser.add_mutually_exclusive_group()
    selection.add_argument("--section", choices=("foundation", "extensions", "packaging", "all") + DOMAINS, default="foundation")
    selection.add_argument("--integration", type=integration_name, help="run one complete tests/next_integration package")
    selection.add_argument("--harness", choices=("evaluation", "scripts"), help="run controlled Python harness tests without a CLI binary")
    parser.add_argument("--suite", choices=("unit", "e2e", "all", "live"), default="all")
    parser.add_argument("--binary", type=Path, default=ROOT / "feishu-cli-next")
    parser.add_argument("--report", type=Path, default=ROOT / ".tmp/next-tests/report.json")
    parser.add_argument("--timeout", type=int, default=1200, help="maximum seconds per command")
    parser.add_argument("--allow-live", action="store_true", help="explicit intent only; unverified fixtures still block live runs")
    args = parser.parse_args(argv)
    if (args.integration or args.harness) and args.suite not in ("e2e", "all"):
        parser.error("--integration and --harness require --suite e2e or all")
    report = {"schema_version": 1, "section": "harness" if args.harness else "integration" if args.integration else args.section, "suite": args.suite,
              "evidence": "offline", "status": "pass", "commands": []}
    if args.suite in ("e2e", "all"):
        report["section_suites"] = [] if args.harness else [integration_suite(args.integration)] if args.integration else section_suites(args.section)
        if args.harness:
            report["harness"] = args.harness
        if args.integration:
            report["integration"] = args.integration
        report["coverage_note"] = "Offline suite results do not establish live behavior or full capability reconciliation."
    code = 0
    if args.timeout <= 0:
        parser.error("--timeout must be positive")
    if args.suite == "live":
        report.update(evidence="live", status="blocked", reason=(
            "Live tests require --allow-live." if not args.allow_live else
            "No verified fixture harness binds the effective account and tenant, private visibility, "
            "Ying Wu's unambiguous identity when applicable, and run-owned cleanup. Legacy live tests are not authorized fixtures."))
        code = 2
    elif args.section == "packaging" and args.suite == "unit":
        report.update(status="blocked", reason="No packaging unit suite is defined; use the section E2E suite when available.")
        code = 2
    elif args.suite in ("e2e", "all") and not args.harness and not args.binary.resolve().is_file():
        report.update(status="blocked", reason="Build the CLI and pass its path with --binary before E2E tests.")
        code = 2
    else:
        # Reserve a local port without listening: proxy connections fail, and a
        # concurrently launched service cannot claim this port during the run.
        with tempfile.TemporaryDirectory(prefix="next-tests-", dir="/tmp" if os.name == "posix" else None) as directory, socket.socket() as dead_proxy:
            dead_proxy.bind(("127.0.0.1", 0))
            proxy = "http://127.0.0.1:" + str(dead_proxy.getsockname()[1])
            env = isolated_env(directory, os.environ, proxy)
            env["LARK_CLI_BIN"] = str(args.binary.resolve())
            if args.harness:
                folder, pattern = ("tests/next_evaluation", "test_*.py") if args.harness == "evaluation" else ("scripts", "next_*test.py")
                commands = [[sys.executable, "-m", "unittest", "discover", "-s", folder, "-p", pattern, "-v"]]
            elif args.integration:
                selected = report["section_suites"][0]
                commands = [["go", "test", "-json", "-count=1", "-timeout=10m", selected["package"]]] if selected["status"] == "present" else []
            else:
                commands = commands_for(args.section, args.suite)
            for command in commands:
                result = run_command(command, env, args.timeout)
                result["selection"] = selection_evidence(command)
                for stream in ("stdout", "stderr"):
                    result[stream] = result[stream].replace(directory, "<isolated-state>")
                report["commands"].append(result)
                print("{}: {}".format(result["status"], " ".join(command)), file=sys.stderr)
                if result["exit_code"]:
                    report["status"] = "fail"
                    code = result["exit_code"] if result["exit_code"] > 0 else 128 - result["exit_code"]
                    break
        if report["status"] == "pass" and any(item["status"] == "missing" for item in report.get("section_suites", [])):
            report.update(status="blocked", reason="Missing requested E2E suites remain unverified; inherited regressions are only a subset.")
            code = 2
    args.report.parent.mkdir(parents=True, exist_ok=True)
    args.report.write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps({"status": report["status"], "evidence": report["evidence"],
                      "section": report["section"], "suite": args.suite, "report": str(args.report), "exit_code": code}))
    return code


if __name__ == "__main__":
    sys.exit(main())
