// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package output_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/larksuite/cli/internal/output"
)

func TestAgentResultControlsPreserveUpstreamFailures(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONTENT_SAFETY_MODE", "off")
	for _, fixture := range []struct {
		name   string
		data   string
		fields string
	}{
		{"mail threads", `{"success_thread_ids":["thread_ok"],"failed_thread_ids":[{"thread_id":"thread_failed","reason":"permission denied"}],"context":"keep"}`, "failed_thread_ids.thread_id"},
		{"apps release logs", `{"release_id":"release_fixture","status":"publishing","error_logs":[{"step":"build","error_log":"compile failed"}],"context":"keep"}`, "error_logs.step"},
	} {
		for _, mode := range []string{"writer", "emitter", "artifact"} {
			t.Run(fixture.name+"/"+mode, func(t *testing.T) {
				opts := &output.AgentOptions{}
				if mode == "artifact" {
					opts = artifactOptions(t)
				}
				opts.Fields = fixture.fields
				var stdout bytes.Buffer
				writer := output.WithAgentOptions(&stdout, opts)
				raw := json.RawMessage(`{"ok":true,"data":` + fixture.data + `}`)
				var err error
				if mode == "emitter" {
					err = output.NewEmitter(output.EmitterConfig{Out: writer}).Success(json.RawMessage(fixture.data), output.EmitOptions{})
				} else {
					err = output.WriteJSON(writer, raw)
				}
				if err != nil {
					t.Fatal(err)
				}
				if err := opts.Finish(); err != nil {
					t.Fatal(err)
				}
				var response struct {
					Data     json.RawMessage          `json:"data"`
					Artifact output.ArtifactReference `json:"artifact"`
				}
				if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if string(response.Data) != fixture.data {
					t.Fatalf("lost failure context: %s", stdout.String())
				}
				if mode != "artifact" {
					return
				}
				if response.Artifact.Schema != output.ArtifactSchema || response.Artifact.Ref == "" {
					t.Fatalf("missing artifact reference: %s", stdout.String())
				}
				saved, err := os.ReadFile(opts.SaveResult)
				if err != nil {
					t.Fatal(err)
				}
				var artifact output.ResultArtifact
				if err := json.Unmarshal(saved, &artifact); err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(artifact.Result, raw) {
					t.Fatalf("full failure artifact changed: %s", artifact.Result)
				}
			})
		}
	}
}
