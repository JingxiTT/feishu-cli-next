// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package im

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/output"
	"github.com/larksuite/cli/shortcuts/common"
	"github.com/spf13/cobra"
)

type conciseWriteFunc func([]byte) (int, error)

func (write conciseWriteFunc) Write(p []byte) (int, error) { return write(p) }

func TestOutputMessagesConcisePreservesWriteFailures(t *testing.T) {
	for _, test := range []struct {
		name  string
		write conciseWriteFunc
		cause error
	}{
		{"writer error", func([]byte) (int, error) { return 0, io.ErrClosedPipe }, io.ErrClosedPipe},
		{"short write", func(p []byte) (int, error) { return len(p) / 2, nil }, io.ErrShortWrite},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
			t.Setenv("LARKSUITE_CLI_CONTENT_SAFETY_MODE", "off")
			f, _, _, _ := cmdutil.TestFactory(t, nil)
			f.IOStreams.Out = test.write
			runtime := &common.RuntimeContext{Cmd: &cobra.Command{Use: "+chat-messages-list"}, Factory: f}
			err := outputMessagesConcise(runtime, conciseMessageView{})
			problem, ok := errs.ProblemOf(err)
			if !ok || problem.Category != errs.CategoryInternal || problem.Subtype != errs.SubtypeUnknown || !errors.Is(err, test.cause) {
				t.Fatalf("write failure lost typed metadata or cause: %v", err)
			}
		})
	}
}

func BenchmarkOutputMessagesConcise(b *testing.B) {
	b.Setenv("LARKSUITE_CLI_CONTENT_SAFETY_MODE", "off")
	for _, count := range []int{50, 200} {
		b.Run(fmt.Sprintf("%dMessages", count), func(b *testing.B) {
			messages := make([]map[string]interface{}, count)
			for i := range messages {
				messages[i] = map[string]interface{}{
					"message_id": fmt.Sprintf("om_%d", i),
					"msg_type":   "text",
					"content":    strings.Repeat("A concise message with text and details. ", 25),
					"sender":     map[string]interface{}{"id": "ou_fixture", "name": "Fixture"},
				}
			}
			view := conciseMessageView{Type: conciseMessageViewChat, ChatSections: []conciseChatSection{{ChatID: "oc_fixture", Messages: messages}}}
			runtime := &common.RuntimeContext{
				Cmd:     &cobra.Command{Use: "+chat-messages-list"},
				Factory: &cmdutil.Factory{IOStreams: &cmdutil.IOStreams{Out: output.WithAgentOptions(io.Discard, &output.AgentOptions{}), ErrOut: io.Discard}},
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := outputMessagesConcise(runtime, view); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
