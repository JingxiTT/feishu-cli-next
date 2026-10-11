// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package schema

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestSchemaCmdInputOnlyPreservesRequestAndSafetyMetadata(t *testing.T) {
	for _, path := range []string{"calendar.events.get", "im.messages.delete", "im.images.create", "im.images"} {
		t.Run(path, func(t *testing.T) {
			t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
			read := func(inputOnly bool) []map[string]any {
				f, out, _, _ := schemaTestFactory(t, nil)
				cmd := NewCmdSchema(f, nil)
				args := []string{path}
				if inputOnly {
					args = append(args, "--input-only")
				}
				cmd.SetArgs(args)
				if err := cmd.Execute(); err != nil {
					t.Fatal(err)
				}
				var envs []map[string]any
				data := bytes.TrimSpace(out.Bytes())
				if data[0] != '[' {
					data = append(append([]byte{'['}, data...), ']')
				}
				if err := json.Unmarshal(data, &envs); err != nil {
					t.Fatal(err)
				}
				return envs
			}
			full, inputs := read(false), read(true)
			for _, env := range full {
				if _, exists := env["outputSchema"]; !exists {
					t.Fatal("default schema lost its response contract")
				}
				delete(env, "outputSchema")
			}
			if !reflect.DeepEqual(full, inputs) {
				t.Fatalf("input-only changed fields besides outputSchema: full=%v input=%v", full, inputs)
			}
		})
	}
}
