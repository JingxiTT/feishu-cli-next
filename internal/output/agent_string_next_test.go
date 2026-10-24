// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package output_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/output"
)

type agentStringDestination struct {
	strings.Builder
	byteWrites int
}

func (w *agentStringDestination) Write(p []byte) (int, error) {
	w.byteWrites++
	return w.Builder.Write(p)
}

func TestAgentStringWriteReusesDestination(t *testing.T) {
	var destination agentStringDestination
	opts := &output.AgentOptions{}
	w := output.WithAgentOptions(&destination, opts)
	const text = "消息 <calendar>\n"
	n, err := io.WriteString(w, text)
	if err != nil || n != len(text) || destination.String() != text || destination.byteWrites != 0 {
		t.Fatalf("string output copied or changed: n=%d err=%v bytes=%d text=%q", n, err, destination.byteWrites, destination.String())
	}
	if err := opts.Finish(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentStringWritePreservesResultControlRejection(t *testing.T) {
	for _, param := range []string{"--result-fields", "--save-result"} {
		t.Run(param, func(t *testing.T) {
			var destination agentStringDestination
			opts := &output.AgentOptions{}
			if param == "--result-fields" {
				opts.Fields = "id"
			} else {
				opts.SaveResult = "unused.json"
			}
			n, err := io.WriteString(output.WithAgentOptions(&destination, opts), "private payload")
			var validation *errs.ValidationError
			if !errors.As(err, &validation) || validation.Param != param || n != 0 || destination.Len() != 0 {
				t.Fatalf("unstructured output accepted: n=%d err=%v data=%q", n, err, destination.String())
			}
			if !errors.Is(opts.Finish(), err) {
				t.Fatal("ignored string write error lost at finalization")
			}
		})
	}
}
