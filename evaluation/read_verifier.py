#!/usr/bin/env python3
# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Independent typed-answer and signed-read verification for bounded section fixtures."""
import json
import sys
from read_fixture import verify

if __name__ == "__main__":
    failures = []
    try:
        success = verify(json.load(sys.stdin), failures)
    except (ValueError, KeyError, TypeError, AttributeError, OSError):
        success = False
        failures.append("invalid_verification_input")
    print(json.dumps({"success": success, "verification_failures": failures}))
