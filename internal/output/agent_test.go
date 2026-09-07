// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package output_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/extension/fileio"
	"github.com/larksuite/cli/internal/output"
	"github.com/larksuite/cli/internal/vfs/localfileio"
)

func TestAgentJSONCompactOptInAndDetail(t *testing.T) {
	data := json.RawMessage(`{"ok":true,"data":{"title":"你好 😀","number":900719925474099312345,"null":null,"empty":"","zero":"001"}}`)
	for _, tc := range []struct {
		name   string
		opts   *output.AgentOptions
		pretty bool
	}{
		{"legacy", nil, true}, {"compact", &output.AgentOptions{}, false}, {"detail", &output.AgentOptions{Detail: true}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := output.WriteJSON(output.WithAgentOptions(&buf, tc.opts), data); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(buf.String(), "\n ") != tc.pretty {
				t.Fatalf("indent mismatch: %s", buf.String())
			}
			var compact bytes.Buffer
			if err := json.Compact(&compact, buf.Bytes()); err != nil {
				t.Fatal(err)
			}
			if compact.String() != string(data) {
				t.Fatalf("changed payload: %s", buf.String())
			}
		})
	}
}

func TestAgentProjectionPreservesEnvelopeIdentityAndPagination(t *testing.T) {
	var buf bytes.Buffer
	raw := json.RawMessage(`{"ok":true,"identity":"bot","data":{"items":[{"record_id":"001","title":"你好","unused":"large","owner":{"open_id":"ou_1","name":"n","unused":false},"n":900719925474099312345,"empty":null}],"has_more":true,"page_token":"next"},"meta":{"pagination":{"complete":false,"items":1,"next_token":"next"}},"new_envelope_field":{"a":1}}`)
	opts := &output.AgentOptions{Fields: "title,owner.name,n,empty"}
	if err := output.WriteJSON(output.WithAgentOptions(&buf, opts), raw); err != nil {
		t.Fatal(err)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if string(got["identity"]) != `"bot"` || len(got["new_envelope_field"]) == 0 || !strings.Contains(string(got["meta"]), `"complete":false`) {
		t.Fatalf("lost envelope: %s", buf.String())
	}
	for _, want := range []string{`"record_id":"001"`, `"has_more":true`, `"page_token":"next"`, `"open_id":"ou_1"`, `"empty":null`, `900719925474099312345`, `你好`} {
		if !strings.Contains(buf.String(), want) {
			t.Fatalf("lost %s: %s", want, buf.String())
		}
	}
	if strings.Contains(buf.String(), "unused") {
		t.Fatalf("unselected field survived: %s", buf.String())
	}
}

func TestAgentProjectionRetainsPartialUnknownUnsupported(t *testing.T) {
	for _, status := range []string{`"ok":false`, `"outcome":"unknown"`, `"partial":true`, `"unsupported_types":["new"]`, `"error":{"message":"lost"}`} {
		raw := json.RawMessage(`{"ok":true,"data":{"items":[{"id":"1","content":"keep",` + status + `}]}}`)
		var buf bytes.Buffer
		if err := output.WriteJSON(output.WithAgentOptions(&buf, &output.AgentOptions{Fields: "id"}), raw); err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(buf.String()) != string(raw) {
			t.Fatalf("unsafe projection reduced result: %s", buf.String())
		}
	}
}

func TestAgentFieldsValidation(t *testing.T) {
	for _, fields := range []string{" ", "a,", ".a", "a..b", "items[]", "a/b", "*", "a b", "a,a..", "a,a.*"} {
		err := (&output.AgentOptions{Fields: fields}).Validate()
		var typed *errs.ValidationError
		if !errors.As(err, &typed) || typed.Param != "--result-fields" {
			t.Fatalf("fields %q: %#v", fields, err)
		}
	}
	if err := (&output.AgentOptions{Fields: "title,owner.name,名字,owner"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (&output.AgentOptions{Fields: "title", Detail: true}).Validate(); !errs.IsValidation(err) {
		t.Fatalf("detail conflict: %v", err)
	}
}

func TestAgentEmitterUsesOptionsThroughBuffer(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONTENT_SAFETY_MODE", "off")
	var buf bytes.Buffer
	opts := &output.AgentOptions{Fields: "title"}
	emitter := output.NewEmitter(output.EmitterConfig{Out: output.WithAgentOptions(&buf, opts), Identity: "bot"})
	if err := emitter.Success(map[string]string{"id": "1", "title": "keep", "extra": "drop"}, output.EmitOptions{}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "extra") || strings.Count(buf.String(), "\n") != 1 {
		t.Fatalf("options lost in buffer: %s", buf.String())
	}
	if err := opts.Finish(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentEmitterPreservesRawJQRecordsAndStreams(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONTENT_SAFETY_MODE", "off")
	data := []interface{}{map[string]interface{}{"id": "1", "title": "<你好>"}}
	for _, opts := range []output.EmitOptions{{Raw: true}, {JQ: ".data"}, {Format: "ndjson"}, {Format: "csv"}, {Format: "table"}} {
		var legacy, agent bytes.Buffer
		if err := output.NewEmitter(output.EmitterConfig{Out: &legacy}).Success(data, opts); err != nil {
			t.Fatal(err)
		}
		if err := output.NewEmitter(output.EmitterConfig{Out: output.WithAgentOptions(&agent, &output.AgentOptions{})}).Success(data, opts); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(legacy.Bytes(), agent.Bytes()) {
			t.Fatalf("changed mode %+v: %q vs %q", opts, legacy.String(), agent.String())
		}
	}
	for _, format := range []string{"json", "ndjson", "csv", "table"} {
		var legacy, agent bytes.Buffer
		for _, w := range []io.Writer{&legacy, output.WithAgentOptions(&agent, &output.AgentOptions{})} {
			emitter := output.NewEmitter(output.EmitterConfig{Out: w})
			if err := emitter.StreamPage(data, output.StreamOptions{Format: format}); err != nil {
				t.Fatal(err)
			}
		}
		if !bytes.Equal(legacy.Bytes(), agent.Bytes()) {
			t.Fatalf("changed stream %s", format)
		}
	}
}

func TestAgentEmitterRejectsProjectionInUnsupportedModes(t *testing.T) {
	for _, opts := range []output.EmitOptions{{Raw: true}, {JQ: ".data"}, {Format: "csv"}, {Format: "ndjson"}, {Format: "table"}, {Format: "pretty"}} {
		var buf bytes.Buffer
		emitter := output.NewEmitter(output.EmitterConfig{Out: output.WithAgentOptions(&buf, &output.AgentOptions{Fields: "id"})})
		err := emitter.Success(map[string]string{"id": "1"}, opts)
		if !errs.IsValidation(err) || buf.Len() != 0 {
			t.Fatalf("mode %+v: %v, %s", opts, err, buf.String())
		}
	}
	emitter := output.NewEmitter(output.EmitterConfig{Out: output.WithAgentOptions(io.Discard, &output.AgentOptions{Fields: "id"})})
	if err := emitter.StreamPage(nil, output.StreamOptions{Format: "json"}); !errs.IsValidation(err) {
		t.Fatalf("stream error = %v", err)
	}
}

func artifactOptions(t *testing.T) *output.AgentOptions {
	t.Helper()
	dir := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	return &output.AgentOptions{
		SaveResult: filepath.Join(dir, "result.json"), FileIO: &localfileio.LocalFileIO{},
		ResolveBinding: func() (output.ArtifactBinding, error) {
			return output.ArtifactBinding{Environment: "feishu", AppID: "cli_fixture", Identity: "bot", Profile: "fixture"}, nil
		},
	}
}

func TestAgentArtifactRoundTripNoClobber(t *testing.T) {
	opts := artifactOptions(t)
	opts.Fields = "id"
	if err := opts.Validate(); err != nil {
		t.Fatal(err)
	}
	raw := json.RawMessage(`{"ok":true,"identity":"bot","data":{"items":[{"id":"001","text":"你好","number":900719925474099312345,"empty":null}],"has_more":true,"page_token":"next"}}`)
	var buf bytes.Buffer
	if err := output.WriteJSON(output.WithAgentOptions(&buf, opts), raw); err != nil {
		t.Fatal(err)
	}
	var ref struct {
		OK       bool                     `json:"ok"`
		Artifact output.ArtifactReference `json:"artifact"`
		Data     json.RawMessage          `json:"data"`
	}
	if err := json.Unmarshal(buf.Bytes(), &ref); err != nil {
		t.Fatal(err)
	}
	resolved, err := opts.FileIO.ResolvePath(opts.SaveResult)
	if err != nil {
		t.Fatal(err)
	}
	if !ref.OK || strings.Contains(string(ref.Data), "items") || !strings.Contains(string(ref.Data), `"page_token":"next"`) || ref.Artifact.Count == nil || *ref.Artifact.Count != 1 || ref.Artifact.Complete == nil || *ref.Artifact.Complete || ref.Artifact.Ref != resolved {
		t.Fatalf("bad ref: %s", buf.String())
	}
	saved, err := os.ReadFile(opts.SaveResult)
	if err != nil {
		t.Fatal(err)
	}
	var artifact output.ResultArtifact
	if err := json.Unmarshal(saved, &artifact); err != nil {
		t.Fatal(err)
	}
	if string(artifact.Result) != string(raw) || artifact.Schema != output.ArtifactSchema || artifact.Binding.AppID != "cli_fixture" || ref.Artifact.Bytes != int64(len(saved)) {
		t.Fatalf("artifact fidelity failure: %s", saved)
	}
	info, err := os.Stat(opts.SaveResult)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("artifact permissions = %v", info.Mode())
	}
	if err := opts.Validate(); !errs.IsValidation(err) {
		t.Fatalf("existing path validation = %v", err)
	}
	buf.Reset()
	if err := output.WriteJSON(output.WithAgentOptions(&buf, opts), json.RawMessage(`{"ok":true}`)); !errs.IsValidation(err) || buf.Len() != 0 {
		t.Fatalf("clobber = %v %s", err, buf.String())
	}
	after, err := os.ReadFile(opts.SaveResult)
	if err != nil || !bytes.Equal(saved, after) {
		t.Fatal("existing artifact changed")
	}
}

func TestAgentArtifactUnknownCompletenessAndFailedOutcome(t *testing.T) {
	for _, raw := range []json.RawMessage{json.RawMessage(`{"ok":true,"data":{"id":"1"}}`), json.RawMessage(`{"ok":false,"data":{"items":[{"id":"1","outcome":"unknown"}],"summary":{"failed":1}}}`)} {
		opts := artifactOptions(t)
		var buf bytes.Buffer
		if err := output.WriteJSON(output.WithAgentOptions(&buf, opts), raw); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(buf.String(), `"complete"`) {
			t.Fatalf("invented completeness: %s", buf.String())
		}
		if strings.Contains(string(raw), `"ok":false`) && (!strings.Contains(buf.String(), `"ok":false`) || !strings.Contains(buf.String(), `"outcome":"unknown"`)) {
			t.Fatalf("hid unknown outcome: %s", buf.String())
		}
	}
}

type noExclusive struct{ fileio.FileIO }
type failedExclusive struct {
	fileio.FileIO
	err error
}

func (f failedExclusive) SaveExclusive(string, fileio.SaveOptions, io.Reader) (fileio.SaveResult, error) {
	return nil, f.err
}

func TestAgentArtifactProviderAndSaveFailure(t *testing.T) {
	opts := artifactOptions(t)
	opts.FileIO = noExclusive{opts.FileIO}
	if err := opts.Validate(); !errs.IsValidation(err) {
		t.Fatalf("unsupported provider: %v", err)
	}
	for _, cause := range []error{fs.ErrExist, io.ErrShortWrite} {
		opts = artifactOptions(t)
		opts.FileIO = failedExclusive{FileIO: opts.FileIO, err: cause}
		var buf bytes.Buffer
		emitter := output.NewEmitter(output.EmitterConfig{Out: output.WithAgentOptions(&buf, opts)})
		err := emitter.Success(map[string]string{"id": "1"}, output.EmitOptions{})
		if buf.Len() != 0 {
			t.Fatalf("false success output: %s", buf.String())
		}
		if errors.Is(cause, fs.ErrExist) {
			if !errs.IsValidation(err) {
				t.Fatalf("race error: %v", err)
			}
		} else if !errors.Is(err, cause) || !errs.IsInternal(err) {
			t.Fatalf("save failure lost type/cause: %v", err)
		}
	}
}

func TestAgentProjectionDoesNotMutateInput(t *testing.T) {
	data := map[string]interface{}{"ok": true, "data": map[string]interface{}{"id": "1", "extra": "untouched"}}
	before := map[string]interface{}{"ok": true, "data": map[string]interface{}{"id": "1", "extra": "untouched"}}
	if err := output.WriteJSON(output.WithAgentOptions(io.Discard, &output.AgentOptions{Fields: "id"}), data); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(data, before) {
		t.Fatal("projection mutated source data")
	}
}

func TestAgentFinishCatchesIgnoredErrorsAndUnsupportedWrites(t *testing.T) {
	opts := &output.AgentOptions{Fields: "id"}
	var buf bytes.Buffer
	w := output.WithAgentOptions(&buf, opts)
	_, _ = io.WriteString(w, "unsupported plain output")
	if err := opts.Finish(); !errs.IsValidation(err) || buf.Len() != 0 {
		t.Fatalf("unsupported output silently passed: %v %s", err, buf.String())
	}
	opts = artifactOptions(t)
	cause := io.ErrShortWrite
	opts.FileIO = failedExclusive{FileIO: opts.FileIO, err: cause}
	output.PrintJson(output.WithAgentOptions(&buf, opts), map[string]bool{"ok": true})
	if err := opts.Finish(); !errs.IsInternal(err) || !errors.Is(err, cause) {
		t.Fatalf("PrintJson ignored save failure: %v", err)
	}
	if err := (&output.AgentOptions{Fields: "id"}).Finish(); !errs.IsValidation(err) {
		t.Fatalf("missing structured output: %v", err)
	}
}

func TestAgentArtifactRejectsCredentialsAndUnresolvedBinding(t *testing.T) {
	for _, data := range []json.RawMessage{json.RawMessage(`{"ok":true,"data":{"access_token":"synthetic"}}`), json.RawMessage(`{"ok":true,"data":{"items":[{"client_secret":"synthetic"}]}}`)} {
		opts := artifactOptions(t)
		err := output.WriteJSON(output.WithAgentOptions(io.Discard, opts), data)
		if !errs.IsValidation(err) {
			t.Fatalf("credential-bearing artifact: %v", err)
		}
		if _, err := os.Stat(opts.SaveResult); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("artifact created: %v", err)
		}
	}
	opts := artifactOptions(t)
	opts.ResolveBinding = func() (output.ArtifactBinding, error) {
		return output.ArtifactBinding{Environment: "feishu", AppID: "fixture", Identity: "user"}, nil
	}
	err := output.WriteJSON(output.WithAgentOptions(io.Discard, opts), map[string]bool{"ok": true})
	if !errs.IsValidation(err) {
		t.Fatalf("unbound user artifact: %v", err)
	}
}

func TestAgentMutationOutcomeOverridesLegacySuccess(t *testing.T) {
	for _, tc := range []struct {
		outcome string
		persist bool
		ok      bool
	}{
		{"confirmed_applied", false, true}, {"unknown", false, false}, {"partial", false, false}, {"confirmed_unapplied", false, false}, {"confirmed_applied", true, false},
	} {
		opts := &output.AgentOptions{Fields: "id", ResolveMutation: func() (string, []string, bool) { return tc.outcome, []string{"attempt_fixture"}, tc.persist }}
		var buf bytes.Buffer
		if err := output.WriteJSON(output.WithAgentOptions(&buf, opts), json.RawMessage(`{"ok":true,"data":{"id":"1","detail":"keep on unknown"}}`)); err != nil {
			t.Fatal(err)
		}
		var env struct {
			OK       bool `json:"ok"`
			Mutation struct {
				Outcome           string   `json:"outcome"`
				Receipts          []string `json:"receipts"`
				PersistenceFailed bool     `json:"persistence_failed"`
			} `json:"mutation"`
		}
		if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
			t.Fatal(err)
		}
		if env.OK != tc.ok || env.Mutation.Outcome != tc.outcome || len(env.Mutation.Receipts) != 1 || env.Mutation.PersistenceFailed != tc.persist {
			t.Fatalf("outcome mismatch: %s", buf.String())
		}
		if !tc.ok && !strings.Contains(buf.String(), "keep on unknown") {
			t.Fatalf("lost uncertain mutation data: %s", buf.String())
		}
	}
}

func TestAgentMutationRawAndJQRemainHonest(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONTENT_SAFETY_MODE", "off")
	for _, mode := range []output.EmitOptions{{Raw: true}, {JQ: ".ok"}} {
		opts := &output.AgentOptions{ResolveMutation: func() (string, []string, bool) { return "unknown", []string{"attempt_fixture"}, false }}
		var buf bytes.Buffer
		emitter := output.NewEmitter(output.EmitterConfig{Out: output.WithAgentOptions(&buf, opts)})
		if err := emitter.Success(map[string]string{"text": "<你好>"}, mode); err != nil {
			t.Fatal(err)
		}
		if mode.Raw {
			if !strings.Contains(buf.String(), `"ok": false`) || !strings.Contains(buf.String(), "<你好>") || !strings.Contains(buf.String(), "attempt_fixture") {
				t.Fatalf("raw outcome/escaping: %s", buf.String())
			}
		} else if buf.String() != "false\n" {
			t.Fatalf("jq status: %s", buf.String())
		}
	}
}

func TestAgentArtifactRejectsAPICredentialsBeforeCreatingFile(t *testing.T) {
	for _, payload := range []string{
		`{"info":{"api_key":"synthetic-api-key"}}`,
		`{"info":{"API_KEY":"synthetic-api-key"}}`,
		`{"items":[{"pat":"synthetic-pat"}]}`,
		`{"password":"synthetic-password"}`,
		`{"GitURL":"https://example.invalid/repo","Token":"synthetic-token"}`,
		`{"git_info":{"token":"synthetic-token"}}`,
		`{"credential":{"token":"synthetic-token"}}`,
		`{"private_key":"synthetic-key"}`,
	} {
		opts := artifactOptions(t)
		var stdout bytes.Buffer
		err := output.WriteJSON(output.WithAgentOptions(&stdout, opts), json.RawMessage(payload))
		var validation *errs.ValidationError
		if !errors.As(err, &validation) || validation.Param != "--save-result" || stdout.Len() != 0 {
			t.Fatalf("credential refusal: %v, stdout=%s", err, stdout.String())
		}
		if _, err := os.Stat(opts.SaveResult); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("credential artifact created: %v", err)
		}
	}
}

func TestAgentArtifactBarePaginationPreservesMetadata(t *testing.T) {
	opts := artifactOptions(t)
	raw := json.RawMessage(`{"messages":[{"message_id":"001","subject":"large content"}],"mailbox_id":"mailbox_fixture","count":1,"has_more":true,"page_token":"next","notice":"results may change","future_metadata":{"version":2}}`)
	var stdout bytes.Buffer
	if err := output.WriteJSON(output.WithAgentOptions(&stdout, opts), raw); err != nil {
		t.Fatal(err)
	}
	var response map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if _, exists := response["ok"]; exists {
		t.Fatalf("invented envelope: %s", stdout.String())
	}
	if _, exists := response["messages"]; exists {
		t.Fatalf("copied collection to stdout: %s", stdout.String())
	}
	for _, key := range []string{"mailbox_id", "count", "has_more", "page_token", "notice", "future_metadata"} {
		var original map[string]json.RawMessage
		_ = json.Unmarshal(raw, &original)
		if !bytes.Equal(response[key], original[key]) {
			t.Fatalf("lost %s: %s", key, stdout.String())
		}
	}
	var ref output.ArtifactReference
	if err := json.Unmarshal(response["artifact"], &ref); err != nil {
		t.Fatal(err)
	}
	if ref.Count == nil || *ref.Count != 1 || ref.Complete == nil || *ref.Complete {
		t.Fatalf("bare extent: %+v", ref)
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
		t.Fatalf("changed original: %s", saved)
	}
}

func TestAgentArtifactBareUnsafeOutcomeNeverInventsSuccess(t *testing.T) {
	for _, raw := range []json.RawMessage{
		json.RawMessage(`{"messages":[{"id":"1","outcome":"unknown"}],"partial":true,"page_token":"next"}`),
		json.RawMessage(`{"error":{"type":"api","message":"synthetic failure"},"mailbox_id":"1"}`),
		json.RawMessage(`{"status":"unknown","messages":[{"id":"1"}]}`),
		json.RawMessage(`[{"id":"1","outcome":"unknown"}]`),
	} {
		opts := artifactOptions(t)
		var stdout bytes.Buffer
		if err := output.WriteJSON(output.WithAgentOptions(&stdout, opts), raw); err != nil {
			t.Fatal(err)
		}
		var response map[string]json.RawMessage
		if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if _, exists := response["ok"]; exists {
			t.Fatalf("invented success: %s", stdout.String())
		}
		var original map[string]json.RawMessage
		if json.Unmarshal(raw, &original) == nil {
			for key, value := range original {
				if !bytes.Equal(response[key], value) {
					t.Fatalf("lost %s: %s", key, stdout.String())
				}
			}
		} else if !bytes.Equal(response["data"], raw) {
			t.Fatalf("lost unsafe records: %s", stdout.String())
		}
	}
}

func TestAgentArtifactResourceTokensRemainSupported(t *testing.T) {
	opts := artifactOptions(t)
	raw := json.RawMessage(`{"files":[{"token":"resource_id","document_token":"document_id"}],"page_token":"next"}`)
	if err := output.WriteJSON(output.WithAgentOptions(io.Discard, opts), raw); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(opts.SaveResult); err != nil {
		t.Fatalf("resource token artifact missing: %v", err)
	}
}
