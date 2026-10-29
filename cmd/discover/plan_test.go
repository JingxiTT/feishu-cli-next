// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package discover

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/larksuite/cli/internal/affordance"
	"github.com/larksuite/cli/internal/apicatalog"
	"github.com/larksuite/cli/internal/cmdmeta"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/meta"
	"github.com/larksuite/cli/internal/output"
	"github.com/larksuite/cli/internal/schema"
	"github.com/spf13/cobra"
)

func TestPlanProjectionKeepsLocalInputsGuidanceAndContinuation(t *testing.T) {
	f, out, _, _ := cmdutil.TestFactory(t, nil)
	f.IOStreams.Out = output.WithAgentOptions(out, &output.AgentOptions{Fields: "command,plan"})
	f.Affordance = affordance.NewResolver(fstest.MapFS{
		"docs.md": &fstest.MapFile{Data: []byte("# docs\n\n## +edit\nEdit text.\n\n### Prerequisites\n- Preserve native IDs.\n\n### Tips\n- Inspect unknown outcomes before retrying.\n")},
	}, apicatalog.Catalog{})
	root := &cobra.Command{Use: "cli"}
	root.PersistentFlags().String("result-fields", "", "projection")
	root.PersistentFlags().String("profile", "", "root option")
	docs := &cobra.Command{Use: "docs"}
	docs.PersistentFlags().String("revision", "", "required domain revision")
	edit := &cobra.Command{Use: "+edit [text]", Short: "Edit text", Long: "Edit text without replacing native IDs.", RunE: func(*cobra.Command, []string) error { t.Fatal("discovery executed mutation"); return nil }}
	edit.Flags().String("document-id", "", "native identifier")
	_ = edit.MarkFlagRequired("document-id")
	edit.Flags().String("private", "", "hidden-canary")
	_ = edit.Flags().MarkHidden("private")
	cmdutil.SetRisk(edit, "write")
	cmdutil.SetSupportedIdentities(edit, []string{"user"})
	cmdmeta.SetAffordanceRef(edit, "docs", "+edit")
	docs.AddCommand(edit, &cobra.Command{Use: "+other", Short: "Other operation"})
	root.AddCommand(docs, NewCmd(f, root))
	root.SetArgs([]string{"discover", "docs", "--limit", "1", "--offset", "0", "--result-fields", "command,plan"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Data []entry
		Meta struct {
			Complete bool
			Next     int `json:"next_offset"`
			Total    int
		}
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Data) != 1 || result.Data[0].Command != "docs +edit" || result.Meta.Complete || result.Meta.Next != 1 || result.Meta.Total != 3 {
		t.Fatalf("lost pagination or route: %s", out)
	}
	p := result.Data[0].Plan
	if p == nil || p.Risk != "write" || !reflect.DeepEqual(p.Identities, []string{"user"}) || !strings.Contains(p.Usage, "[text]") || p.Description != edit.Long {
		t.Fatalf("lost command semantics: %s", out)
	}
	if len(p.Flags) != 2 || p.Flags[0].Name != "document-id" || !p.Flags[0].Required || p.Flags[1].Name != "revision" || p.Example != "cli docs +edit --document-id <document-id>" {
		t.Fatalf("lost local/domain inputs: %s", out)
	}
	if p.Guidance == nil || len(p.Guidance.Prerequisites) != 1 || len(p.Guidance.Tips) != 1 || strings.Contains(out.String(), "hidden-canary") || strings.Contains(out.String(), "root option") {
		t.Fatalf("lost guidance or exposed excluded flags: %s", out)
	}
}

func TestPlansAreOptInAndRequestMatchesInputSchema(t *testing.T) {
	f, _, _, _ := cmdutil.TestFactory(t, nil)
	svc := meta.ServiceFromMap(map[string]interface{}{
		"name": "im", "resources": map[string]interface{}{
			"messages": map[string]interface{}{"methods": map[string]interface{}{
				"delete": map[string]interface{}{"id": "im.messages.delete", "risk": "high-risk-write", "accessTokens": []interface{}{"user"}, "requiredScopes": []interface{}{"im:message"},
					"parameters": map[string]interface{}{"message_id": map[string]interface{}{"type": "string", "required": true, "location": "path"}},
					"requestBody": map[string]interface{}{"options": map[string]interface{}{"type": "object", "required": true, "properties": map[string]interface{}{
						"limit": map[string]interface{}{"type": "integer", "required": true, "minimum": 1, "maximum": 10},
					}}}},
			}},
		},
	})
	f.APICatalog = apicatalog.New(apicatalog.SourceEmbedded, []meta.Service{svc})
	root := &cobra.Command{Use: "cli"}
	root.PersistentFlags().String("result-fields", "", "projection")
	method := &cobra.Command{Use: "delete"}
	cmdmeta.SetSource(method, cmdmeta.SourceService, true)
	entries := []entry{{Command: "im messages delete", command: method}}
	if err := describePlans(root, f, root, entries); err != nil {
		t.Fatal(err)
	}
	if entries[0].Plan != nil {
		t.Fatal("default discovery gained a plan")
	}
	if err := root.PersistentFlags().Set("result-fields", "command,plan.request"); err != nil {
		t.Fatal(err)
	}
	if err := describePlans(root, f, root, entries); err != nil {
		t.Fatal(err)
	}
	target, err := f.APICatalog.Resolve([]string{"im", "messages", "delete"})
	if err != nil {
		t.Fatal(err)
	}
	want := schema.InputEnvelopeOf(f.Affordance, *target.Method)
	params := want.InputSchema.Properties.Map["params"]
	if params.Properties == nil || params.Properties.Map["message_id"].Flag != "--message-id" || !reflect.DeepEqual(params.Required, []string{"message_id"}) {
		t.Fatalf("missing required parameter: %+v", params)
	}
	if entries[0].Plan == nil || !reflect.DeepEqual(entries[0].Plan.Request, &want) || want.Meta.Risk != "high-risk-write" || len(want.Meta.RequiredScopes) != 1 || want.OutputSchema != nil {
		t.Fatalf("request/permission parity lost: %+v", entries[0].Plan)
	}
}
