#!/usr/bin/env python3
# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Check actual synthetic state, independently of the agent's final claim."""
import json
from pathlib import Path
import sys

request = json.load(sys.stdin)
path = Path(request["workspace"]) / "state.json"
actual = json.loads(path.read_text()) if path.exists() else None
print(json.dumps({"success": actual == request["expected"]}))
