// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package output

import (
	"encoding/json"
	"io"
	"strings"
	"unicode"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/extension/fileio"
)

// AgentOptions is invocation-scoped. The command validates options before it
// executes and only enables projections/artifacts for supported JSON operations.
// A zero value opts into compact JSON without changing the result's fields.
type AgentOptions struct {
	Fields           string
	Detail           bool
	SaveResult       string
	FileIO           fileio.FileIO
	ResolveBinding   func() (ArtifactBinding, error)
	ResolveMutation  func() (outcome string, receipts []string, persistenceFailed bool)
	firstError       error
	structuredWrites int
}

type agentWriter struct {
	io.Writer
	options *AgentOptions
}

// WithAgentOptions attaches rendering options without intercepting arbitrary
// writes. Raw, jq, pretty and record renderers retain their existing contracts.
// Options are referenced so flag parsing may happen after writer construction.
func WithAgentOptions(w io.Writer, opts *AgentOptions) io.Writer {
	if opts == nil {
		return w
	}
	return &agentWriter{Writer: w, options: opts}
}

func agentOptions(w io.Writer) *AgentOptions {
	if aw, ok := w.(*agentWriter); ok {
		return aw.options
	}
	return nil
}

// Validate checks projection syntax and the artifact destination before an API
// request. Command registration additionally checks command/mode compatibility.
func (o *AgentOptions) Validate() error {
	if o == nil {
		return nil
	}
	if _, err := parseFields(o.Fields); err != nil {
		return err
	}
	if o.Detail && o.Fields != "" {
		return errs.NewValidationError(errs.SubtypeInvalidArgument, "--result-detail and --result-fields cannot be combined").WithParam("--result-fields")
	}
	return o.validateArtifact()
}

type fieldSelection map[string]fieldSelection

func parseFields(fields string) (fieldSelection, error) {
	if fields == "" {
		return nil, nil
	}
	tree := fieldSelection{}
	for _, field := range strings.Split(fields, ",") {
		field = strings.TrimSpace(field)
		current := tree
		parts := strings.Split(field, ".")
		for _, part := range parts {
			if part == "" || !validField(part) {
				return nil, errs.NewValidationError(errs.SubtypeInvalidArgument,
					"--result-fields expects comma-separated field names or dotted paths").WithParam("--result-fields")
			}
		}
		for i, part := range parts {
			if previous, exists := current[part]; exists && previous == nil {
				break
			}
			if i == len(parts)-1 {
				current[part] = nil
				break
			}
			if current[part] == nil {
				current[part] = fieldSelection{}
			}
			current = current[part]
		}
	}
	return tree, nil
}

func validField(s string) bool {
	for _, r := range s {
		if r != '_' && r != '-' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// RawMessage avoids a float64 round trip: arbitrary JSON numbers, explicit
// nulls, empty values, and Unicode remain semantically unchanged.
func renderAgentJSON(w io.Writer, data interface{}, opts *AgentOptions) error {
	encoded, err := json.Marshal(data)
	if err != nil {
		return &outputMarshalError{err: err}
	}
	encoded, err = opts.annotateMutation(encoded)
	if err != nil {
		return err
	}
	fields, err := parseFields(opts.Fields)
	if err != nil {
		return err
	}
	if opts.SaveResult != "" {
		encoded, err = opts.saveArtifact(encoded)
	} else if len(fields) != 0 {
		encoded, err = projectResult(encoded, fields)
	}
	if err != nil {
		return err
	}
	if opts.Detail {
		var value json.RawMessage = encoded
		encoded, err = json.MarshalIndent(value, "", "  ")
		if err != nil {
			return &outputMarshalError{err: err}
		}
	}
	_, err = agentDestination(w).Write(append(encoded, '\n'))
	return err
}

func projectResult(raw json.RawMessage, fields fieldSelection) (json.RawMessage, error) {
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) == nil && object != nil {
		if _, envelope := object["ok"]; envelope {
			// A failed or unknown result must never be reduced by a success projection.
			if unsafeProjection(raw) {
				return raw, nil
			}
			if data, found := object["data"]; found {
				projected, err := projectValue(data, fields)
				if err != nil {
					return nil, err
				}
				object["data"] = projected
			}
			return json.Marshal(object)
		}
	}
	return projectValue(raw, fields)
}

func projectValue(raw json.RawMessage, fields fieldSelection) (json.RawMessage, error) {
	if unsafeProjection(raw) {
		return raw, nil
	}
	var array []json.RawMessage
	if json.Unmarshal(raw, &array) == nil && array != nil {
		for i, item := range array {
			projected, err := projectValue(item, fields)
			if err != nil {
				return nil, err
			}
			array[i] = projected
		}
		return json.Marshal(array)
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil || object == nil {
		return raw, nil
	}
	projected := map[string]json.RawMessage{}
	for key, value := range object {
		selected, exists := fields[key]
		switch {
		case exists && selected == nil:
			projected[key] = value
		case exists:
			child, err := projectValue(value, selected)
			if err != nil {
				return nil, err
			}
			projected[key] = child
		case collectionField(key):
			child, err := projectValue(value, fields)
			if err != nil {
				return nil, err
			}
			projected[key] = child
		case protectedField(key):
			projected[key] = value
		}
	}
	return json.Marshal(projected)
}

func collectionField(key string) bool {
	switch key {
	case "items", "records", "rows", "files":
		return true
	}
	return false
}

func protectedField(key string) bool {
	if strings.HasSuffix(key, "_id") || strings.HasSuffix(key, "_ids") || strings.HasSuffix(key, "_token") {
		return true
	}
	switch key {
	case "id", "guid", "token", "url", "identity", "resource", "binding", "ok", "status", "state", "outcome", "type", "unsupported", "unsupported_types", "error", "errors", "warning", "warnings", "missing_scopes", "summary", "meta", "pagination", "has_more", "complete", "completeness", "partial", "truncated", "unknown", "count", "total", "receipt", "recovery", "artifact", "_notice", "_content_safety_alert":
		return true
	}
	return false
}

// Retain the whole result when any nested record reports failure, uncertainty,
// or unsupported content. Future domain-specific projections may be narrower
// only after defining and testing that domain's outcome schema.
func unsafeProjection(raw json.RawMessage) bool {
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) == nil && object != nil {
		for key, value := range object {
			switch key {
			case "ok":
				if string(value) == "false" {
					return true
				}
			case "error", "errors", "unsupported", "unsupported_types", "warning", "warnings", "missing_scopes":
				if string(value) != "null" && string(value) != "[]" && string(value) != "{}" && string(value) != "false" {
					return true
				}
			case "partial", "unknown", "truncated":
				if string(value) == "true" {
					return true
				}
			case "outcome", "status", "state":
				var state string
				_ = json.Unmarshal(value, &state)
				switch state {
				case "unknown", "partial", "failed", "unsupported":
					return true
				}
			}
			if unsafeProjection(value) {
				return true
			}
		}
		return false
	}
	var array []json.RawMessage
	if json.Unmarshal(raw, &array) == nil {
		for _, value := range array {
			if unsafeProjection(value) {
				return true
			}
		}
	}
	return false
}

func agentDestination(w io.Writer) io.Writer {
	if aw, ok := w.(*agentWriter); ok {
		return aw.Writer
	}
	return w
}

func (w *agentWriter) Write(p []byte) (int, error) {
	if w.options.Fields != "" || w.options.SaveResult != "" {
		err := errs.NewValidationError(errs.SubtypeInvalidArgument,
			"this command does not support structured result output controls").WithParam("--result-fields")
		if w.options.SaveResult != "" {
			err = err.WithParam("--save-result")
		}
		w.options.recordError(err)
		return 0, err
	}
	return w.Writer.Write(p)
}

func (o *AgentOptions) recordError(err error) {
	if err != nil && o.firstError == nil {
		o.firstError = err
	}
}

// Finish surfaces errors ignored by legacy PrintJson callers and rejects result
// controls when a command produced no structured result. Run it after RunE only
// when the command itself succeeded; command failures keep their original type.
func (o *AgentOptions) Finish() error {
	if o == nil {
		return nil
	}
	if o.firstError != nil {
		return wrapOutputError("write", o.firstError)
	}
	if (o.Fields != "" || o.SaveResult != "") && o.structuredWrites == 0 {
		param := "--result-fields"
		if o.SaveResult != "" {
			param = "--save-result"
		}
		return errs.NewValidationError(errs.SubtypeInvalidArgument,
			"this command did not produce a structured result").WithParam(param)
	}
	return nil
}

// The command preflight rejects these combinations before execution; checking
// again here protects embedded Emitter callers from silently ignored options.
func (e *Emitter) validateAgentMode(format, jq string, raw, stream bool) error {
	if e == nil {
		return nil
	}
	opts := agentOptions(e.out)
	if opts == nil || (opts.Fields == "" && opts.SaveResult == "") {
		return nil
	}
	if raw || jq != "" || stream || (format != "" && format != "json") {
		param := "--result-fields"
		if opts.SaveResult != "" {
			param = "--save-result"
		}
		return errs.NewValidationError(errs.SubtypeInvalidArgument,
			"result projection and artifact output require non-streaming JSON without --raw or --jq").WithParam(param)
	}
	return nil
}

type mutationResult struct {
	Outcome           string   `json:"outcome"`
	Receipts          []string `json:"receipts"`
	PersistenceFailed bool     `json:"persistence_failed,omitempty"`
}

func (m *mutationResult) unsuccessful() bool {
	return m != nil && (m.PersistenceFailed || m.Outcome == "unknown" || m.Outcome == "partial" || m.Outcome == "confirmed_unapplied")
}

func (o *AgentOptions) mutation() *mutationResult {
	if o == nil || o.ResolveMutation == nil {
		return nil
	}
	outcome, receipts, failed := o.ResolveMutation()
	if len(receipts) == 0 {
		return nil
	}
	return &mutationResult{outcome, receipts, failed}
}

func (o *AgentOptions) annotateMutation(raw json.RawMessage) (json.RawMessage, error) {
	var envelope map[string]json.RawMessage
	if json.Unmarshal(raw, &envelope) != nil || envelope == nil {
		return raw, nil
	}
	if _, exists := envelope["ok"]; !exists {
		return raw, nil
	}
	mutation := o.mutation()
	if mutation == nil {
		return raw, nil
	}
	encoded, err := json.Marshal(mutation)
	if err != nil {
		return nil, &outputMarshalError{err: err}
	}
	envelope["mutation"] = encoded
	if mutation.unsuccessful() {
		envelope["ok"] = json.RawMessage("false")
	}
	return json.Marshal(envelope)
}
