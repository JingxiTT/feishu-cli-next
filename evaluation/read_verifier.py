#!/usr/bin/env python3
"""Independent typed-answer and signed-read verification for bounded section fixtures."""
import json
import sys
from read_fixture import verify

if __name__ == "__main__":
    try:
        success = verify(json.load(sys.stdin))
    except (ValueError, KeyError, TypeError, AttributeError, OSError):
        success = False
    print(json.dumps({"success": success}))
