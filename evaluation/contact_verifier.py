#!/usr/bin/env python3
# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Verify contact answers against the external fixture's post-run request receipt."""
import json
from pathlib import Path
import sys


def verify(request):
    workspace = Path(request["workspace"])
    answer = json.loads((workspace / "answer.json").read_text(encoding="utf-8"))
    receipt = json.loads((workspace / "backend-receipt.json").read_text(encoding="utf-8"))
    if answer != request["expected"]["answer"] or receipt.get("scenario") != "contact-exact-id-v1":
        return False
    records = receipt.get("requests")
    if not isinstance(records, list) or not records:
        return False
    allowed = {"identity", "contact_get", "contact_batch", "contact_search"}
    if any(not isinstance(record, dict) or record.get("status") != 200 or record.get("route") not in allowed for record in records):
        return False
    if any(path.is_file() for path in (workspace / "feishu-config").rglob("*")):
        return False
    return any(answer["user_id"] in record.get("returned_ids", []) and record["route"] != "identity" for record in records)


if __name__ == "__main__":
    try:
        success = verify(json.load(sys.stdin))
    except (ValueError, KeyError, TypeError, OSError):
        success = False
    print(json.dumps({"success": success}))
