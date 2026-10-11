// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package whiteboard

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
)

func TestParseWBcliNodesPreservesNumericAndRichValues(t *testing.T) {
	const node = `{"id":"001","x":9007199254740993,"y":1.234567890123456789,"text":"你好 😀","empty":"","null":null,"image":{"token":"image-fixture"},"custom":[true,0]}`
	for _, source := range []string{`{"nodes":[` + node + `]}`, `{"code":0,"data":{"to":"openapi","result":{"nodes":[` + node + `]}}}`} {
		nodes, err, _ := parseWBcliNodes([]byte(source))
		if err != nil {
			t.Fatal(err)
		}
		value := nodes[0].(map[string]interface{})
		if value["x"] != json.Number("9007199254740993") || value["y"] != json.Number("1.234567890123456789") {
			t.Fatalf("numeric values changed: %#v", value)
		}
		encoded, err := json.Marshal(nodes)
		if err != nil {
			t.Fatal(err)
		}
		for _, expected := range []string{"9007199254740993", "1.234567890123456789", "你好 😀", `"id":"001"`, `"null":null`, `"empty":""`, `"image":{"token":"image-fixture"}`} {
			if !strings.Contains(string(encoded), expected) {
				t.Fatalf("missing %q in %s", expected, encoded)
			}
		}
	}
}

func TestParseWBcliNodesRejectsTrailingInput(t *testing.T) {
	for _, input := range []string{`{"nodes":[]} {}`, `{"nodes":[]} garbage`} {
		_, err, _ := parseWBcliNodes([]byte(input))
		var validation *errs.ValidationError
		if !errors.As(err, &validation) || validation.Param != "--source" {
			t.Fatalf("got %v", err)
		}
	}
}

func TestWBUpdateDryRunPreservesIdempotentToken(t *testing.T) {
	for _, format := range []string{FormatRaw, FormatMermaid, FormatPlantUML, FormatSVG} {
		rt := newTestRuntime(map[string]string{"whiteboard-token": "fixture", "input_format": format, "source": `{"nodes":[]}`, "idempotent-token": "synthetic-idempotent-token"}, nil)
		encoded, err := json.Marshal(wbUpdateDryRun(context.Background(), rt))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(encoded), `"client_token":"synthetic-idempotent-token"`) {
			t.Fatalf("missing query: %s", encoded)
		}
	}
}
