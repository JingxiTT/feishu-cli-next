// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package errs

import (
	"encoding/json"
	"errors"
	"slices"
)

// Mutation outcomes recorded by receipts and reported on the wire.
const (
	MutationApplied   = "confirmed_applied"
	MutationUnapplied = "confirmed_unapplied"
	MutationPartial   = "partial"
	MutationUnknown   = "unknown"
)

// MutationOutcomeError exposes observed effects to both process and embedded
// callers. The original error remains in the Go chain; its typed wire fields
// are retained in CauseDetail without granting permission to replay a write.
type MutationOutcomeError struct {
	Problem
	Outcome           string          `json:"outcome"`
	Receipts          []string        `json:"receipts"`
	PersistenceFailed bool            `json:"persistence_failed,omitempty"`
	CauseDetail       json.RawMessage `json:"cause,omitempty"`
	Cause             error           `json:"-"`
}

func (e *MutationOutcomeError) Error() string {
	if e == nil {
		return ""
	}
	return e.Problem.Error()
}
func (e *MutationOutcomeError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// NewMutationOutcomeError constructs a non-retryable effect report. Outcome is
// confirmed_unapplied, partial, unknown, or confirmed_applied with a persistence
// failure. Invalid outcomes conservatively become unknown.
func NewMutationOutcomeError(outcome string, receipts []string, persistenceFailed bool, cause error) *MutationOutcomeError {
	var existing *MutationOutcomeError
	if errors.As(cause, &existing) {
		return existing
	}
	switch outcome {
	case MutationUnapplied, MutationPartial, MutationUnknown:
	case MutationApplied:
		if !persistenceFailed {
			outcome = MutationUnknown
		}
	default:
		outcome = MutationUnknown
	}
	message := "mutation outcome is " + outcome
	if persistenceFailed {
		message = "mutation evidence could not be durably finalized"
	}
	return &MutationOutcomeError{
		Problem: Problem{Category: CategoryAPI, Subtype: SubtypeMutationOutcome, Message: message,
			Hint: "Inspect the referenced receipts. Do not replay an unknown or partially applied mutation without backend evidence that replay is safe."},
		Outcome: outcome, Receipts: slices.Clone(receipts), PersistenceFailed: persistenceFailed,
		Cause: cause, CauseDetail: mutationCauseDetail(cause),
	}
}

func mutationCauseDetail(cause error) json.RawMessage {
	typed, ok := UnwrapTypedError(cause)
	if !ok {
		return nil
	}
	body, err := json.Marshal(typed)
	if err != nil {
		if problem, ok := ProblemOf(cause); ok {
			body, _ = json.Marshal(problem)
		}
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil {
		return nil
	}
	// A transport/rate-limit error may recommend retry in isolation, which is
	// unsafe when nested beneath an outcome report. Preserve all other typed
	// diagnostics, including scopes, category/subtype, API code and request IDs.
	delete(fields, "hint")
	delete(fields, "retryable")
	delete(fields, "retry_after_seconds")
	body, _ = json.Marshal(fields)
	return body
}
