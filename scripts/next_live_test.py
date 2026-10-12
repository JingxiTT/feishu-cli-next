#!/usr/bin/env python3
# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT
"""Run one explicitly selected, account-bound Feishu test locally, never in CI."""

import argparse
import json
import os
from pathlib import Path
import sys
import tempfile

from next_test import ROOT, run_command


# Keep exact selectors: inherited live workflows do not verify private fixtures.
SECTIONS = {
    "contact": ("contact", "TestContactNextLiveCurrentAccountReads", "NEXT_CONTACT_LIVE",
                ("NEXT_CONTACT_LIVE_ACCOUNT_ID", "NEXT_CONTACT_LIVE_TENANT_KEY")),
    "drive": ("drive", "TestDriveNextPrivateFileLifecycle", "NEXT_DRIVE_LIVE",
              ("NEXT_DRIVE_LIVE_FOLDER_TOKEN", "NEXT_DRIVE_LIVE_BINDING_SHA256",
               "NEXT_DRIVE_LIVE_CREATION_VERIFIED", "NEXT_DRIVE_LIVE_STATE_DIR")),
    "sheets": ("sheets", "TestSheets_PrivateUserLifecycle", "FEISHU_NEXT_SHEETS_PRIVATE_LIVE",
               ("FEISHU_NEXT_PRIVATE_LIVE_FOLDER_TOKEN", "FEISHU_NEXT_PRIVATE_LIVE_OPEN_ID",
                "FEISHU_NEXT_PRIVATE_LIVE_TENANT_KEY", "FEISHU_NEXT_SHEETS_LIVE_EVIDENCE_DIR")),
    "docs": ("docs", "TestDocsNextPrivateLiveLifecycle", "NEXT_DOCS_LIVE",
             ("NEXT_DOCS_LIVE_FIXTURE", "NEXT_DOCS_LIVE_BINDING", "NEXT_DOCS_LIVE_EVIDENCE_DIR")),
    "markdown": ("markdown", "TestMarkdownNextPrivateLiveLifecycle", "NEXT_MARKDOWN_LIVE",
                 ("NEXT_MARKDOWN_LIVE_FIXTURE", "NEXT_MARKDOWN_LIVE_BINDING", "NEXT_MARKDOWN_LIVE_EVIDENCE_DIR")),
    "base": ("base", "TestBaseNextPrivateLiveLifecycle", "NEXT_BASE_PRIVATE_LIVE",
             ("NEXT_BASE_PRIVATE_FIXTURE", "NEXT_BASE_PRIVATE_BINDING", "NEXT_BASE_PRIVATE_STATE")),
    "base-supported": ("base", "TestBaseNextPrivateSupportedLiveLifecycle", "NEXT_BASE_PRIVATE_SUPPORTED_LIVE",
                       ("NEXT_BASE_PRIVATE_FIXTURE", "NEXT_BASE_PRIVATE_BINDING", "NEXT_BASE_PRIVATE_STATE")),
}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--section", required=True, help="one of: " + ", ".join(SECTIONS))
    parser.add_argument("--allow-live", action="store_true", help="explicitly authorize this local test invocation")
    parser.add_argument("--binary", type=Path, default=ROOT / "feishu-cli-next")
    parser.add_argument("--report", type=Path, default=ROOT / ".tmp/next-tests/live.json")
    args = parser.parse_args(argv)
    args.report.parent.mkdir(parents=True, exist_ok=True)
    report = {"schema_version": 1, "section": args.section, "suite": "live", "evidence": "live",
              "status": "blocked", "commands": [],
              "coverage_note": "One selected live scenario; other live capabilities remain unverified."}
    selected = SECTIONS.get(args.section)
    code = 2
    if os.environ.get("CI") or os.environ.get("GITHUB_ACTIONS") == "true":
        report["reason"] = "Real-account tests are manual and local only; CI execution is disabled."
    elif not args.allow_live:
        report["reason"] = "Local live tests require --allow-live."
    elif selected is None:
        report["reason"] = "No account-bound local harness for this section; select one of: " + ", ".join(SECTIONS)
    elif not args.binary.resolve().is_file():
        report["reason"] = "Build the CLI and pass its path with --binary."
    else:
        package, test, opt_in, required = selected
        missing = [key for key in required if not os.environ.get(key)]
        if os.environ.get(opt_in) != "1":
            missing.insert(0, opt_in + "=1")
        if missing:
            report["reason"] = "Verify the section's account/private-fixture prerequisites and set: " + ", ".join(missing)
        else:
            env = dict(os.environ, LARK_CLI_BIN=str(args.binary.resolve()),
                       LARKSUITE_CLI_NO_UPDATE_NOTIFIER="1", LARKSUITE_CLI_NO_SKILLS_NOTIFIER="1")
            command = ["go", "test", "-json", "-count=1", "-timeout=15m",
                       "./tests/cli_e2e/" + package, "-run", "^" + test + "$"]
            # Reserve private diagnostics before any API call; never overwrite a
            # previous run's recovery information or publish it in CI artifacts.
            with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8", dir=args.report.parent,
                                             prefix=args.report.stem + "-", suffix=".log", delete=False) as log:
                result = run_command(command, env, 1200)
                log.write(result.get("stdout", "") + "\n" + result.get("stderr", ""))
                report["diagnostics"] = log.name
            # Private API output and credentials must not enter the summary.
            result.pop("stdout", None)
            result.pop("stderr", None)
            report["commands"].append(result)
            if result["exit_code"]:
                report.update(status="fail", reason="Inspect private fixture/recovery evidence before any rerun; mutations are not replayed.")
                code = result["exit_code"] if result["exit_code"] > 0 else 128 - result["exit_code"]
            elif result["tests"].get("skip") or not result["tests"].get("pass"):
                report["reason"] = "The selected live test skipped or did not execute; live behavior remains unverified."
            else:
                report["status"] = "pass"
                code = 0
    args.report.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"status": report["status"], "evidence": "live", "section": args.section,
                      "report": str(args.report), "exit_code": code, "reason": report.get("reason"),
                      "diagnostics": report.get("diagnostics")}))
    return code


if __name__ == "__main__":
    sys.exit(main())
