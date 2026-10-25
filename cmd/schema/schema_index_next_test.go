// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package schema

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func runSchemaForIndex(t *testing.T, args ...string) []byte {
	t.Helper()
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	f, out, _, _ := schemaTestFactory(t, nil)
	cmd := NewCmdSchema(f, nil)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("schema %v: %v", args, err)
	}
	return bytes.TrimSpace(out.Bytes())
}

type schemaIndexResult struct {
	OK   bool                `json:"ok"`
	Data []map[string]string `json:"data"`
	Meta map[string]any      `json:"meta"`
}

func TestSchemaIndexBareListsServicesWithoutSchemaBodies(t *testing.T) {
	raw := runSchemaForIndex(t)
	if len(raw) >= 64<<10 {
		t.Fatalf("bare schema output = %d bytes, want < 64 KiB", len(raw))
	}
	if bytes.Contains(raw, []byte("inputSchema")) {
		t.Fatal("bare schema index carried schema bodies")
	}
	var res struct {
		OK   bool             `json:"ok"`
		Data []map[string]any `json:"data"`
		Meta map[string]any   `json:"meta"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatal(err)
	}
	if !res.OK || len(res.Data) == 0 {
		t.Fatalf("bare schema index not ok or empty: %s", raw)
	}
	for _, item := range res.Data {
		if _, ok := item["service"].(string); !ok {
			t.Fatalf("service entry lacks service name: %v", item)
		}
		if _, ok := item["methods"].(float64); !ok {
			t.Fatalf("service entry lacks method count: %v", item)
		}
	}
	if res.Meta["hint"] != schemaIndexHint {
		t.Errorf("meta hint = %v, want %q", res.Meta["hint"], schemaIndexHint)
	}
}

func TestSchemaIndexServiceMatchesFullListing(t *testing.T) {
	var index schemaIndexResult
	if err := json.Unmarshal(runSchemaForIndex(t, "im"), &index); err != nil {
		t.Fatal(err)
	}
	if !index.OK || len(index.Data) == 0 {
		t.Fatalf("im index empty or not ok: %+v", index)
	}
	if index.Meta["count"] != float64(len(index.Data)) {
		t.Errorf("meta count = %v, want %d", index.Meta["count"], len(index.Data))
	}
	for _, item := range index.Data {
		if !strings.HasPrefix(item["method"], "im.") || item["description"] == "" {
			t.Fatalf("index entry malformed: %v", item)
		}
		if len([]rune(item["description"])) > schemaIndexDescRunes+3 {
			t.Errorf("description not truncated: %q", item["description"])
		}
	}

	var full []map[string]any
	if err := json.Unmarshal(runSchemaForIndex(t, "im", "--all"), &full); err != nil {
		t.Fatal(err)
	}
	if len(full) != len(index.Data) {
		t.Fatalf("full listing has %d methods, index has %d", len(full), len(index.Data))
	}
	normalize := strings.NewReplacer(".", "_", " ", "_")
	names := make(map[string]bool, len(full))
	for _, env := range full {
		names[normalize.Replace(env["name"].(string))] = true
	}
	for _, item := range index.Data {
		if !names[normalize.Replace(item["method"])] {
			t.Errorf("index method %q has no full envelope", item["method"])
		}
	}
}

func TestSchemaIndexSingleMethodUnchanged(t *testing.T) {
	plain := runSchemaForIndex(t, "im.messages.delete")
	withAll := runSchemaForIndex(t, "im.messages.delete", "--all")
	if !bytes.Equal(plain, withAll) {
		t.Fatal("single method output changed with --all")
	}
	var env map[string]any
	if err := json.Unmarshal(plain, &env); err != nil {
		t.Fatal(err)
	}
	if env["name"] != "im messages delete" {
		t.Errorf("single method name = %v, want \"im messages delete\"", env["name"])
	}
}
