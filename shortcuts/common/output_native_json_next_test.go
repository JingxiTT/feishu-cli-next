// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package common

import (
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/output"
)

func TestRuntimeNativeRichJSONSupportsSharedResultControls(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONTENT_SAFETY_MODE", "off")
	for _, formatted := range []bool{false, true} {
		runtime, stdout, _ := newJqTestContext("", "json")
		opts := &output.AgentOptions{Fields: "content"}
		runtime.Factory.IOStreams.Out = output.WithAgentOptions(stdout, opts)
		data := map[string]interface{}{"document_id": "doc_fixture", "content": "<document>你好 & 😀</document>", "unused": "drop"}
		if formatted {
			runtime.OutFormatRaw(data, nil, nil)
		} else {
			runtime.OutRaw(data, nil)
		}
		if err := runtime.OutputError(); err != nil {
			t.Fatal(err)
		}
		if err := opts.Finish(); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(stdout.String(), "unused") || !strings.Contains(stdout.String(), "<document>你好 & 😀</document>") || !strings.Contains(stdout.String(), "doc_fixture") {
			t.Fatalf("native rich result controls ignored: %s", stdout.String())
		}
	}
}
