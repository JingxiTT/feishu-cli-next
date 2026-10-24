# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

# Targets this fork adds. Makefile includes this file and keeps its upstream
# targets; the few upstream targets the fork changes are edited there in place.

.PHONY: next-script-test evaluation-tooling-test evaluation-test cross-domain-test offline-test live-test update-install fetch_meta

# Restores the historical frozen comparison catalog for inventory tools only;
# the product build embeds the committed catalog instead.
fetch_meta:
	python3 scripts/next_meta.py

next-script-test:
	python3 scripts/next_test.py --harness scripts --report .tmp/next-tests/scripts.json

# CI tests the evaluator software with synthetic usage; this never bills models.
evaluation-tooling-test:
	python3 scripts/next_test.py --harness evaluation --report .tmp/next-tests/evaluation.json

# Compatibility name used by existing contributor instructions and callers.
evaluation-test: evaluation-tooling-test

cross-domain-test: build
	python3 scripts/next_test.py --integration sheets_base --binary ./$(BINARY) --report .tmp/next-tests/cross-domain.json

# Account-free checks; identical to the existing default test target.
offline-test: test

# Explicit local selection; fixture opt-ins remain required by each live test.
live-test:
	python3 scripts/next_live_test.py --section "$(SECTION)" --allow-live --binary ./$(BINARY) --report .tmp/next-tests/live.json

update-install: build
	python3 scripts/next_binary.py update --prefix "$(PREFIX)" --source "$(BINARY)"
