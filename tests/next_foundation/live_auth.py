#!/usr/bin/env python3
# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT
"""Explicit read-only live auth verification; never imported as a live test.

Run only at the foundation acceptance boundary:
  python3 tests/next_foundation/live_auth.py --allow-live-read \
    --binary ./feishu-cli-next --report .tmp/next-tests/live-auth.json

Uses the current account's credential source. Auth may renew an existing token.
The only operations are auth status --json --verify and the authenticated user's
own user_info GET. No resource, recipient, profile, or installation is changed.
Raw command output, account identifiers, names, tokens, and hints are never saved.
General live/content suites remain blocked by scripts/next_test.py.
"""

import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

AUTH_ARGS = ("auth", "status", "--json", "--verify")
USER_ARGS = ("api", "GET", "/open-apis/authen/v1/user_info", "--as", "user")
STATUSES = {"ready", "not_configured", "missing", "needs_refresh", "verify_failed"}
ERROR_TYPES = {"validation", "authentication", "authorization", "config", "network",
               "api", "policy", "internal", "confirmation"}


def digest(value):
    return hashlib.sha256(value.encode("utf-8")).hexdigest() if isinstance(value, str) and value else None


def object_output(stdout):
    try:
        value = json.loads(stdout)
    except (ValueError, TypeError):
        return None
    return value if isinstance(value, dict) else None


def safe_error(stderr):
    payload = object_output(stderr) or {}
    error = payload.get("error", {})
    if not isinstance(error, dict):
        return "unrecognized"
    kind = error.get("type")
    return kind if isinstance(kind, str) and kind in ERROR_TYPES else "unrecognized"


def summarize_identity(identity):
    if not isinstance(identity, dict):
        return {"status": "unrecognized", "verified": None}
    scopes = identity.get("scope")
    scopes = sorted(set(scopes.split())) if isinstance(scopes, str) else []
    status = identity.get("status")
    return {"status": status if isinstance(status, str) and status in STATUSES else "unrecognized",
            "verified": identity.get("verified") if type(identity.get("verified")) is bool else None,
            "available": identity.get("available") if type(identity.get("available")) is bool else None,
            "identity_sha256": digest(identity.get("openId")),
            "scope_count": len(scopes), "scopes_sha256": digest(" ".join(scopes))}


def auth_evidence(payload, exit_code, stderr=""):
    if exit_code != 0:
        return {"name": "auth_status_verify", "status": "fail", "exit_code": exit_code,
                "reason": "command_failed", "error_type": safe_error(stderr)}
    identities = payload.get("identities") if isinstance(payload, dict) else None
    if not isinstance(identities, dict):
        return {"name": "auth_status_verify", "status": "fail", "reason": "invalid_auth_response"}
    summary = {kind: summarize_identity(identities.get(kind)) for kind in ("bot", "user")}
    verified = any(identity["verified"] is True for identity in summary.values()) and bool(digest(payload.get("appId")))
    return {"name": "auth_status_verify", "status": "pass" if verified else "blocked",
            "reason": "verified_identity_observed" if verified else "no_verified_identity",
            "account_sha256": digest(payload.get("appId")), "identities": summary}


def own_identity_evidence(auth_payload, user_payload, exit_code, stderr=""):
    if exit_code != 0:
        kind = safe_error(stderr)
        unavailable = kind in {"authentication", "authorization", "config"}
        return {"name": "own_user_identity", "status": "skip" if unavailable else "fail", "exit_code": exit_code,
                "reason": "own_user_info_unavailable_with_existing_authorization" if unavailable else "command_failed",
                "error_type": kind}
    if not isinstance(user_payload, dict) or user_payload.get("ok") is not True or not isinstance(user_payload.get("data"), dict):
        return {"name": "own_user_identity", "status": "fail", "reason": "invalid_user_response"}
    user = user_payload["data"]
    identities = auth_payload.get("identities", {}) if isinstance(auth_payload, dict) else {}
    expected = identities.get("user", {}) if isinstance(identities, dict) else {}
    if not isinstance(expected, dict):
        expected = {}
    expected_id, observed_id = expected.get("openId"), user.get("open_id")
    if not isinstance(expected_id, str) or not expected_id or not isinstance(observed_id, str) or not observed_id:
        return {"name": "own_user_identity", "status": "blocked", "reason": "identity_binding_unavailable"}
    if observed_id != expected_id:
        return {"name": "own_user_identity", "status": "fail", "reason": "identity_mismatch",
                "expected_identity_sha256": digest(expected_id), "observed_identity_sha256": digest(observed_id)}
    # Current auth status does not normally expose a tenant key. When it is
    # absent, bind the observed tenant but do not claim independent consistency.
    expected_tenant = expected.get("tenant_key") or auth_payload.get("tenant_key")
    observed_tenant = user.get("tenant_key")
    tenant_check = "unavailable"
    if isinstance(expected_tenant, str) and expected_tenant and isinstance(observed_tenant, str) and observed_tenant:
        tenant_check = "match" if expected_tenant == observed_tenant else "mismatch"
    return {"name": "own_user_identity", "status": "fail" if tenant_check == "mismatch" else "pass",
            "identity_sha256": digest(observed_id), "account_sha256": digest(auth_payload.get("appId")),
            "tenant_sha256": digest(observed_tenant), "tenant_consistency": tenant_check}


def invoke(binary, args):
    # Preserve the explicitly selected current account; suppress background
    # discovery/notifier traffic. No test-supplied credential values are added.
    env = dict(os.environ, LARKSUITE_CLI_REMOTE_META="off", LARKSUITE_CLI_NO_UPDATE_NOTIFIER="1",
               LARKSUITE_CLI_NO_SKILLS_NOTIFIER="1")
    try:
        result = subprocess.run([str(binary), *args], env=env, capture_output=True, text=True, timeout=60)
    except subprocess.TimeoutExpired:
        return None, 124, ""
    except OSError:
        return None, 127, ""
    return object_output(result.stdout), result.returncode, result.stderr


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--allow-live-read", action="store_true")
    parser.add_argument("--binary", required=True, type=Path)
    parser.add_argument("--report", required=True, type=Path)
    args = parser.parse_args(argv)
    report = {"schema_version": 1, "evidence": "live", "section": "foundation",
              "scope": "current-account auth and own-user-info read only", "status": "blocked", "tests": []}
    if not args.allow_live_read:
        report["reason"] = "explicit_allow_live_read_required"
    elif not args.binary.resolve().is_file():
        report["reason"] = "built_binary_required"
    else:
        auth_payload, code, stderr = invoke(args.binary.resolve(), AUTH_ARGS)
        auth = auth_evidence(auth_payload, code, stderr)
        report["tests"].append(auth)
        identities = auth_payload.get("identities", {}) if isinstance(auth_payload, dict) else {}
        user = identities.get("user", {}) if isinstance(identities, dict) else {}
        if auth["status"] == "pass" and isinstance(user, dict) and user.get("verified") is True:
            payload, code, stderr = invoke(args.binary.resolve(), USER_ARGS)
            report["tests"].append(own_identity_evidence(auth_payload, payload, code, stderr))
        else:
            report["tests"].append({"name": "own_user_identity", "status": "skip", "reason": "verified_user_identity_unavailable"})
        statuses = {case["status"] for case in report["tests"]}
        report["status"] = "fail" if "fail" in statuses else "blocked" if statuses & {"skip", "blocked"} else "pass"
        if any(case.get("tenant_consistency") == "unavailable" for case in report["tests"]):
            report["limitations"] = ["Independent tenant consistency is unavailable from auth status; observed tenant is hashed only."]
    code = 0 if report["status"] == "pass" else 1 if report["status"] == "fail" else 2
    args.report.parent.mkdir(parents=True, exist_ok=True)
    args.report.write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps(report))
    return code


if __name__ == "__main__":
    sys.exit(main())
