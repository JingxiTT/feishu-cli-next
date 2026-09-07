#!/usr/bin/env python3
"""Check actual synthetic state, independently of the agent's final claim."""
import json
from pathlib import Path
import sys

request = json.load(sys.stdin)
path = Path(request["workspace"]) / "state.json"
actual = json.loads(path.read_text()) if path.exists() else None
print(json.dumps({"success": actual == request["expected"]}))
