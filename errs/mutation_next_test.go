// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package errs_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
)

func TestMutationOutcomePreservesCauseAndScopeWireFields(t *testing.T) {
	cause := errs.NewPermissionError(errs.SubtypeMissingScope, "synthetic missing scope").WithMissingScopes("synthetic:write").WithCode(99991679).WithLogID("synthetic-log")
	ids := []string{"receipt-id"}
	outcome := errs.NewMutationOutcomeError("confirmed_unapplied", ids, false, fmt.Errorf("wrapper: %w", cause))
	ids[0] = "changed"
	if !errors.Is(outcome, cause) || !errs.IsMutationOutcome(outcome) || outcome.Receipts[0] != "receipt-id" {
		t.Fatal(outcome)
	}
	var permission *errs.PermissionError
	if !errors.As(outcome, &permission) || permission != cause {
		t.Fatal("lost cause")
	}
	body, err := json.Marshal(outcome)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Type, Subtype, Outcome string
		Receipts               []string
		Cause                  struct {
			Type, Subtype, LogID string
			Code                 int
			MissingScopes        []string `json:"missing_scopes"`
		}
	}
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	if wire.Type != "api" || wire.Subtype != "mutation_outcome" || wire.Outcome != "confirmed_unapplied" || wire.Cause.Code != 99991679 || len(wire.Cause.MissingScopes) != 1 {
		t.Fatalf("%s", body)
	}
	if p, ok := errs.ProblemOf(outcome); !ok || p.Subtype != errs.SubtypeMutationOutcome || p.Retryable {
		t.Fatal(outcome)
	}
}
func TestMutationOutcomeDoesNotAdvertiseUnsafeRetry(t *testing.T) {
	cause := errs.NewNetworkError(errs.SubtypeNetworkTimeout, "synthetic timeout").WithHint("retry immediately").WithRetryable().WithRetryAfterSeconds(4)
	outcome := errs.NewMutationOutcomeError("unknown", []string{"id"}, false, cause)
	body, err := json.Marshal(outcome)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "retry immediately") || strings.Contains(string(body), "retryable") || strings.Contains(string(body), "retry_after_seconds") {
		t.Fatalf("unsafe retry diagnostic: %s", body)
	}
	if !cause.Retryable || cause.Hint != "retry immediately" {
		t.Fatal("original cause mutated")
	}
	if errs.NewMutationOutcomeError("unknown", nil, false, outcome) != outcome {
		t.Fatal("nested duplicate outcome")
	}
}
func TestMutationOutcomeNilAndUnknown(t *testing.T) {
	var missing *errs.MutationOutcomeError
	if missing.Error() != "" || missing.Unwrap() != nil {
		t.Fatal("nil error unsafe")
	}
	outcome := errs.NewMutationOutcomeError("unexpected", nil, false, errors.New("private raw cause"))
	body, _ := json.Marshal(outcome)
	if outcome.Outcome != "unknown" || strings.Contains(string(body), "private raw cause") {
		t.Fatalf("%s", body)
	}
}
