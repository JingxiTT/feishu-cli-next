// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/larksuite/cli/errs"
)

func TestMutationOutcomeExitAndEnvelope(t *testing.T) {
	for _, tt := range []struct {
		outcome string
		persist bool
		exit    int
	}{
		{"confirmed_unapplied", false, 20}, {"partial", false, 21}, {"unknown", false, 22}, {"confirmed_applied", true, 22},
	} {
		t.Run(tt.outcome, func(t *testing.T) {
			cause := errs.NewPermissionError(errs.SubtypeMissingScope, "synthetic denial").WithMissingScopes("synthetic:write")
			outcome := errs.NewMutationOutcomeError(tt.outcome, []string{"id"}, tt.persist, cause)
			wrapped := fmt.Errorf("host wrapper: %w", outcome)
			if code := ExitCodeOf(wrapped); code != tt.exit {
				t.Fatalf("exit %d", code)
			}
			var buffer bytes.Buffer
			if !WriteTypedErrorEnvelope(&buffer, wrapped, "bot") {
				t.Fatal("not typed")
			}
			var wire struct {
				OK    bool
				Error struct {
					Outcome  string
					Receipts []string
					Cause    struct {
						MissingScopes []string `json:"missing_scopes"`
					}
				}
			}
			if err := json.Unmarshal(buffer.Bytes(), &wire); err != nil {
				t.Fatal(err)
			}
			if wire.OK || wire.Error.Outcome != tt.outcome || len(wire.Error.Receipts) != 1 || len(wire.Error.Cause.MissingScopes) != 1 {
				t.Fatal(buffer.String())
			}
		})
	}
}
