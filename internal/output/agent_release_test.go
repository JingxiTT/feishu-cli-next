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

func TestAgentResultControlsPreserveReleaseApprovalContext(t *testing.T) {
	node := `{"current_node":"ADMIN_APPROVAL","current_status":"PENDING","result":{"approval_url":"https://example.invalid/approval/001","future_field":null},"submitted_by":{"open_id":"ou_fixture","username":"审批人"},"created_at":900719925474099312345}`
	raw := json.RawMessage(`{"ok":true,"identity":"bot","data":{"release_id":"release_fixture","status":"pending","current_node_info":` + node + `,"unused":"drop"}}`)
	for _, tc := range []struct {
		name   string
		fields string
		save   bool
	}{
		{"projection", "release_id", false},
		{"nested status projection", "current_node_info.current_status", false},
		{"nested approval projection", "current_node_info.result.approval_url", false},
		{"artifact", "", true},
		{"artifact with projection", "release_id", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := &output.AgentOptions{}
			if tc.save {
				opts = artifactOptions(t)
			}
			opts.Fields = tc.fields
			var stdout bytes.Buffer
			if err := output.WriteJSON(output.WithAgentOptions(&stdout, opts), raw); err != nil {
				t.Fatal(err)
			}
			var response struct {
				OK       bool                       `json:"ok"`
				Identity string                     `json:"identity"`
				Data     map[string]json.RawMessage `json:"data"`
				Artifact output.ArtifactReference   `json:"artifact"`
			}
			if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if !response.OK || response.Identity != "bot" || string(response.Data["release_id"]) != `"release_fixture"` || string(response.Data["status"]) != `"pending"` {
				t.Fatalf("lost release identity/status: %s", stdout.String())
			}
			if string(response.Data["current_node_info"]) != node {
				t.Fatalf("lost release approval context: %s", stdout.String())
			}
			if _, found := response.Data["unused"]; found {
				t.Fatalf("unselected release payload retained: %s", stdout.String())
			}
			if !tc.save {
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
				t.Fatalf("full release artifact changed: %s", artifact.Result)
			}
		})
	}
}
