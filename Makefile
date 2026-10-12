# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

BINARY   := feishu-cli-next
MODULE   := github.com/larksuite/cli
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
DATE     := $(shell date +%Y-%m-%d)
NODE     ?= node
QUALITY_GATE_CHANGED_FROM ?= $(shell bash scripts/resolve-changed-from.sh)
QUALITY_GATE_CHANGED_FROM_RESOLVED = $(if $(strip $(QUALITY_GATE_CHANGED_FROM)),$(QUALITY_GATE_CHANGED_FROM),$(shell bash scripts/resolve-changed-from.sh))
QUALITY_GATE_DIR ?= .tmp/quality-gate
QUALITY_GATE_MANIFEST_OUT ?= $(QUALITY_GATE_DIR)/command-manifest.json
QUALITY_GATE_COMMAND_INDEX_OUT ?= $(QUALITY_GATE_DIR)/command-index.json
QUALITY_GATE_FACTS_OUT ?= $(QUALITY_GATE_DIR)/facts.json
PUBLIC_CONTENT_METADATA ?= $(QUALITY_GATE_DIR)/public-content-metadata.json
LDFLAGS  := -s -w -X $(MODULE)/internal/build.Version=$(VERSION) -X $(MODULE)/internal/build.Date=$(DATE)
PREFIX   ?= /usr/local

# Keep the validated riscv64 test mode until race-enabled execution is verified
# on that architecture with the upgraded Go toolchain.
# Prefer GOARCH passed to make (for example, `make GOARCH=riscv64 unit-test`)
# over `go env GOARCH`, because command-line make variables are not visible to
# $(shell ...).
TEST_GOARCH := $(or $(GOARCH),$(shell go env GOARCH))
RACE_FLAG := $(if $(filter riscv64,$(TEST_GOARCH)),,-race)

.PHONY: all build vet fmt-check script-test test unit-test live-skills-test integration-test examples-build quality-gate install update-install uninstall clean fetch_meta gitleaks sidecar-test next-script-test evaluation-test evaluation-tooling-test cross-domain-test

all: test

fetch_meta:
	python3 scripts/next_meta.py

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

vet:
	go vet ./...

# fmt-check fails when any file would be reformatted by gofmt. Keep this
# in sync with the fast-gate "Check formatting" step in CI.
fmt-check:
	@unformatted=$$(gofmt -l . | grep -v '^\.claude/' || true); \
	if [ -n "$$unformatted" ]; then \
		echo "Unformatted Go files:"; \
		echo "$$unformatted"; \
		echo "Run 'gofmt -w .' and commit."; \
		exit 1; \
	fi

next-script-test:
	python3 scripts/next_test.py --harness scripts --report .tmp/next-tests/scripts.json

# CI tests the evaluator software with synthetic usage; this never bills models.
evaluation-tooling-test:
	python3 scripts/next_test.py --harness evaluation --report .tmp/next-tests/evaluation.json

# Compatibility name used by existing contributor instructions and callers.
evaluation-test: evaluation-tooling-test

script-test: next-script-test
	bash scripts/check-doc-tokens.sh
	bash scripts/resolve-changed-from.test.sh
	bash scripts/ci-workflow.test.sh
	bash scripts/release-workflow.test.sh
	bash scripts/semantic-review-workflow.test.sh
	$(NODE) --test scripts/e2e_domains.test.js scripts/fetch_e2e_tat.test.js scripts/install.test.js scripts/release-preflight.test.js scripts/release-publish-policy.test.js scripts/semantic-review-verify-artifact.test.js scripts/pr-quality-summary.test.js scripts/semantic-review-publish.test.js scripts/ci-quality-summary-publish.test.js

# ./extension/... keeps the public plugin SDK in the default test matrix.
unit-test:
	go test $(RACE_FLAG) -gcflags="all=-N -l" -count=1 \
		./cmd/... ./internal/... ./shortcuts/... ./extension/...

live-skills-test:
	LARKSUITE_CLI_RUN_LIVE_SKILLS_TESTS=1 \
	go test -v -count=1 ./cmd/update \
		-run '^TestUpdateCommand_(RealSkillsSyncRewritesState|SkillsSyncColdStart)$$'

# examples-build keeps the shipped plugin-SDK examples compilable. If this
# breaks, the plugin author guide's "go build ./..." path is broken.
examples-build:
	go build ./extension/platform/examples/audit-observer
	go build ./extension/platform/examples/readonly-policy

# ./tests/... includes tests/plugin_e2e, which builds ~20 customer-fork
# binaries (~1 min warm; a cold module cache also downloads via GOPROXY).
# Deliberate: local `make test` exercises the L4 plugin contract by default.
cross-domain-test: build
	python3 scripts/next_test.py --integration sheets_base --binary ./$(BINARY) --report .tmp/next-tests/cross-domain.json

integration-test: build cross-domain-test
	python3 scripts/next_test.py --section all --suite e2e --binary ./$(BINARY) --report .tmp/next-tests/integration.json

test: vet fmt-check script-test unit-test examples-build integration-test evaluation-test

quality-gate: build
	mkdir -p $(QUALITY_GATE_DIR) $(dir $(QUALITY_GATE_FACTS_OUT)) $(dir $(PUBLIC_CONTENT_METADATA))
	test -f $(PUBLIC_CONTENT_METADATA) || printf '{}\n' > $(PUBLIC_CONTENT_METADATA)
	LARKSUITE_CLI_NO_UPDATE_NOTIFIER=1 \
	LARKSUITE_CLI_NO_SKILLS_NOTIFIER=1 \
	go run ./internal/qualitygate/cmd/manifest-export \
		--manifest-out $(QUALITY_GATE_MANIFEST_OUT) \
		--command-index-out $(QUALITY_GATE_COMMAND_INDEX_OUT)
	LARKSUITE_CLI_APP_ID=dry-run \
	LARKSUITE_CLI_APP_SECRET=dry-run \
	LARKSUITE_CLI_BRAND=feishu \
	LARKSUITE_CLI_CONFIG_DIR=$${TMPDIR:-/tmp}/quality-gate-cli-config \
	LARKSUITE_CLI_NO_UPDATE_NOTIFIER=1 \
	LARKSUITE_CLI_NO_SKILLS_NOTIFIER=1 \
	go run ./internal/qualitygate/cmd/quality-gate check \
		--repo . \
		--cli-bin ./$(BINARY) \
		--changed-from $(QUALITY_GATE_CHANGED_FROM_RESOLVED) \
		--manifest $(QUALITY_GATE_MANIFEST_OUT) \
		--command-index $(QUALITY_GATE_COMMAND_INDEX_OUT) \
		--public-content-metadata $(PUBLIC_CONTENT_METADATA) \
		--facts-out $(QUALITY_GATE_FACTS_OUT)

install: build
	python3 scripts/next_binary.py install --prefix "$(PREFIX)" --source "$(BINARY)"

update-install: build
	python3 scripts/next_binary.py update --prefix "$(PREFIX)" --source "$(BINARY)"

uninstall:
	python3 scripts/next_binary.py uninstall --prefix "$(PREFIX)"

clean:
	rm -f $(BINARY)

# sidecar-test compiles and runs the authsidecar* build-tagged code that the
# default CI matrix never sees (they carry //go:build tags).
sidecar-test:
	go build -tags authsidecar -o /dev/null .
	go test $(RACE_FLAG) -count=1 -tags authsidecar ./extension/credential/sidecar/ ./extension/transport/sidecar/ ./internal/cmdutil/
	go test $(RACE_FLAG) -count=1 -tags authsidecar_demo ./sidecar/server-demo/
	go test $(RACE_FLAG) -count=1 -tags authsidecar ./tests/sidecar_e2e/

# Run secret-leak checks locally before pushing.
# Step 1: check-doc-tokens catches realistic-looking example tokens in reference
#         docs and asks you to use _EXAMPLE_TOKEN placeholders instead.
# Step 2: gitleaks scans the full repo for real leaked secrets.
# Install gitleaks: https://github.com/gitleaks/gitleaks#installing
gitleaks:
	@bash scripts/check-doc-tokens.sh
	@command -v gitleaks >/dev/null 2>&1 || { echo "gitleaks not found. Install: brew install gitleaks"; exit 1; }
	gitleaks detect --redact -v --exit-code=2
