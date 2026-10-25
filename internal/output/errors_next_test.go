// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package output

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
)

func TestSetCompactErrorsWritesSingleLineWithSameContent(t *testing.T) {
	t.Cleanup(func() { SetCompactErrors(false) })
	err := errs.NewValidationError(errs.SubtypeInvalidArgument, "bad value").WithParam("--doc")

	var indented bytes.Buffer
	SetCompactErrors(false)
	if !WriteTypedErrorEnvelope(&indented, err, "bot") {
		t.Fatal("indented write failed")
	}
	var compact bytes.Buffer
	SetCompactErrors(true)
	if !WriteTypedErrorEnvelope(&compact, err, "bot") {
		t.Fatal("compact write failed")
	}

	if got := strings.Count(compact.String(), "\n"); got != 1 {
		t.Fatalf("compact envelope has %d newlines, want 1: %q", got, compact.String())
	}
	if strings.Contains(indented.String(), "\n  ") == false {
		t.Fatalf("default envelope is not indented: %q", indented.String())
	}
	var want, got map[string]any
	if err := json.Unmarshal(indented.Bytes(), &want); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(compact.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("compact content differs:\nwant %v\n got %v", want, got)
	}
}
