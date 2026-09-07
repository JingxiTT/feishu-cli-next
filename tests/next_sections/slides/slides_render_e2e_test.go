// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT
package slides_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"strings"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/tidwall/gjson"
)

func TestGeneratedRenderNativeImageEvidenceLimit(t *testing.T) {
	for _, width := range []int{1, 960} {
		name := "small"
		if width > 1 {
			name = "large"
		}
		t.Run(name, func(t *testing.T) {
			img := image.NewNRGBA(image.Rect(0, 0, width, width*9/16+1))
			var encoded bytes.Buffer
			// A valid uncompressed image deterministically crosses the shared receipt
			// evidence cap without relying on random pixels or malformed padding.
			encoder := png.Encoder{CompressionLevel: png.NoCompression}
			if err := encoder.Encode(&encoded, img); err != nil {
				t.Fatal(err)
			}
			if _, err := png.Decode(bytes.NewReader(encoded.Bytes())); err != nil {
				t.Fatal(err)
			}
			payload := base64.StdEncoding.EncodeToString(encoded.Bytes())
			calls := 0
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "POST" || r.URL.Path != "/open-apis/slides_ai/v1/slide_image/render" || r.URL.RawQuery != "" {
					t.Error(r.Method, r.URL)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body) != 1 || body["content"] != "<slide><data/></slide>" {
					t.Error(body, err)
				}
				reply(w, map[string]any{"slide_image": map[string]any{"slide_id": "rendered", "slide_number": 1, "format": 1, "data": payload}})
			}))
			r := run(f, "xml_presentation.slide_image", "render", "--data", `{"content":"<slide><data/></slide>"}`)
			wantExit, wantOutcome, wantEvidence := 0, "confirmed_applied", "api_success_response"
			if len(payload) > 1<<20 {
				wantExit, wantOutcome, wantEvidence = 22, "unknown", "response_too_large"
			}
			if r.ExitCode != wantExit || !strings.Contains(r.Stderr, `"outcome":"`+wantOutcome+`"`) {
				t.Fatalf("exit=%d stderr=%s", r.ExitCode, r.Stderr)
			}
			if got := gjson.Get(r.Stdout, "data.slide_image.data").String(); got != payload {
				t.Fatalf("image content changed: got %d bytes want %d", len(got), len(payload))
			}
			if calls != 1 {
				t.Fatal(calls)
			}
			receipts := f.Run("receipts", "list")
			if receipts.ExitCode != 0 {
				t.Fatal(receipts)
			}
			if len(gjson.Get(receipts.Stdout, "data").Array()) != 1 || gjson.Get(receipts.Stdout, "data.0.outcome").String() != wantOutcome || gjson.Get(receipts.Stdout, "data.0.evidence").String() != wantEvidence {
				t.Fatal(receipts)
			}
			if gjson.Get(r.Stdout, "ok").Bool() != (wantExit == 0) || gjson.Get(r.Stdout, "data.slide_image.format").Int() != 1 || gjson.Get(r.Stdout, "data.slide_image.slide_id").String() != "rendered" || gjson.Get(r.Stdout, "data.slide_image.slide_number").Int() != 1 {
				t.Fatal("native metadata or stdout success status changed")
			}
			if calls != 1 {
				t.Fatal("receipt inspection dispatched a request", calls)
			}
			t.Logf("valid PNG=%d bytes base64=%d bytes; dispatch=%d exit=%d evidence=%s", encoded.Len(), len(payload), calls, r.ExitCode, wantEvidence)
		})
	}
}
