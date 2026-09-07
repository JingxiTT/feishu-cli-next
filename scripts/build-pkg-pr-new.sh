#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
# Local preview only. No official namespace or external PR publication.
exec python3 scripts/next_package.py --output .pkg-pr-new "$@"
