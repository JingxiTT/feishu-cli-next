// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/extension/fileio"
)

const ArtifactSchema = "feishu-cli-next/result/v1"

// ArtifactBinding records only resolved, non-secret identity facts. App/account
// binding is not proof of tenant identity; no tenant ID is inferred.
type ArtifactBinding struct {
	Environment string `json:"environment"`
	AppID       string `json:"app_id"`
	UserID      string `json:"user_id,omitempty"`
	Identity    string `json:"identity"`
	Profile     string `json:"profile,omitempty"`
}

// ResultArtifact preserves the complete original JSON result inside a versioned
// container. Domain imports must revalidate Binding before reusing its content.
type ResultArtifact struct {
	Schema  string          `json:"schema"`
	Binding ArtifactBinding `json:"binding"`
	Result  json.RawMessage `json:"result"`
}

type ArtifactReference struct {
	Schema   string          `json:"schema"`
	Ref      string          `json:"ref"`
	Bytes    int64           `json:"bytes"`
	Count    *int            `json:"count,omitempty"`
	Complete *bool           `json:"complete,omitempty"`
	Binding  ArtifactBinding `json:"binding"`
}

func (o *AgentOptions) validateArtifact() error {
	if o.SaveResult == "" {
		return nil
	}
	if _, ok := o.FileIO.(fileio.ExclusiveFileIO); !ok {
		return errs.NewValidationError(errs.SubtypeFailedPrecondition,
			"--save-result requires a FileIO provider with atomic exclusive saves").WithParam("--save-result")
	}
	if o.ResolveBinding == nil {
		return errs.NewValidationError(errs.SubtypeFailedPrecondition,
			"--save-result requires resolved account binding").WithParam("--save-result")
	}
	path, err := o.FileIO.ResolvePath(o.SaveResult)
	if err != nil {
		return errs.NewValidationError(errs.SubtypeInvalidArgument,
			"invalid result artifact path").WithParam("--save-result").WithCause(err)
	}
	if _, err := o.FileIO.Stat(path); err == nil {
		return artifactExists()
	} else if !errors.Is(err, fs.ErrNotExist) {
		return errs.NewValidationError(errs.SubtypeInvalidArgument,
			"cannot inspect result artifact path").WithParam("--save-result").WithCause(err)
	}
	return nil
}

func artifactExists() error {
	return errs.NewValidationError(errs.SubtypeInvalidArgument,
		"result artifact already exists").WithParam("--save-result").WithHint("choose a new artifact path; existing files are never overwritten")
}

func (o *AgentOptions) saveArtifact(result json.RawMessage) (json.RawMessage, error) {
	if containsCredentialField(result) {
		return nil, errs.NewValidationError(errs.SubtypeInvalidArgument,
			"credential-bearing results cannot be saved as result artifacts").WithParam("--save-result")
	}
	if err := o.validateArtifact(); err != nil {
		return nil, err
	}
	binding, err := o.ResolveBinding()
	if err != nil {
		return nil, err
	}
	if binding.Environment == "" || binding.AppID == "" || binding.Identity == "" || (binding.Identity == "user" && binding.UserID == "") {
		return nil, errs.NewValidationError(errs.SubtypeFailedPrecondition,
			"result artifact account binding is incomplete").WithParam("--save-result")
	}
	path, err := o.FileIO.ResolvePath(o.SaveResult)
	if err != nil {
		return nil, errs.NewValidationError(errs.SubtypeInvalidArgument,
			"invalid result artifact path").WithParam("--save-result").WithCause(err)
	}
	artifact := ResultArtifact{Schema: ArtifactSchema, Binding: binding, Result: result}
	encoded, err := json.Marshal(artifact)
	if err != nil {
		return nil, &outputMarshalError{err: err}
	}
	encoded = append(encoded, '\n')
	saved, err := o.FileIO.(fileio.ExclusiveFileIO).SaveExclusive(path,
		fileio.SaveOptions{ContentType: "application/json", ContentLength: int64(len(encoded))}, bytes.NewReader(encoded))
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return nil, artifactExists()
		}
		return nil, errs.NewInternalError(errs.SubtypeUnknown,
			"failed to save result artifact").WithCause(err)
	}
	ref := ArtifactReference{Schema: ArtifactSchema, Ref: path, Bytes: saved.Size(), Binding: binding}
	ref.Count, ref.Complete = artifactExtent(result)
	var response map[string]json.RawMessage
	if json.Unmarshal(result, &response) != nil || response == nil {
		response = map[string]json.RawMessage{}
	}
	_, envelope := response["ok"]
	// Bare command results keep their native metadata and outcome. Only known
	// collections may be omitted from an otherwise safe result; never manufacture
	// ok:true merely because serializing the result succeeded.
	unsafe := unsafeProjection(result)
	if !envelope && !unsafe {
		for key := range response {
			if collectionField(key) {
				delete(response, key)
			}
		}
	} else if !envelope && unsafe && len(response) == 0 {
		response["data"] = result
	}
	// An incomplete/failed/unknown result keeps its decision-critical payload on
	// stdout as well as in the artifact. Never let saving turn failure into success.
	if envelope && !unsafe {
		var data map[string]json.RawMessage
		_ = json.Unmarshal(response["data"], &data)
		summary := map[string]json.RawMessage{}
		for key, value := range data {
			if protectedField(key) {
				summary[key] = value
			}
		}
		delete(response, "data")
		if len(summary) != 0 {
			response["data"], err = json.Marshal(summary)
			if err != nil {
				return nil, &outputMarshalError{err: err}
			}
		}
	}
	reference, err := json.Marshal(ref)
	if err != nil {
		return nil, &outputMarshalError{err: err}
	}
	response["artifact"] = reference
	return json.Marshal(response)
}

// Only explicit backend pagination or observed array length supplies extent.
// Missing completeness stays omitted; locally finishing serialization proves
// nothing about whether the backend has more results.
func artifactExtent(result json.RawMessage) (*int, *bool) {
	var envelope struct {
		Data json.RawMessage `json:"data"`
		Meta struct {
			Count      *int `json:"count"`
			Pagination *struct {
				Items    *int  `json:"items"`
				Complete *bool `json:"complete"`
			} `json:"pagination"`
		} `json:"meta"`
	}
	_ = json.Unmarshal(result, &envelope)
	if envelope.Meta.Pagination != nil {
		return envelope.Meta.Pagination.Items, envelope.Meta.Pagination.Complete
	}
	count := envelope.Meta.Count
	data := envelope.Data
	if len(data) == 0 {
		data = result
	}
	var array []json.RawMessage
	if json.Unmarshal(data, &array) == nil && array != nil {
		n := len(array)
		return &n, nil
	}
	var page struct {
		Items    []json.RawMessage `json:"items"`
		Records  []json.RawMessage `json:"records"`
		Rows     []json.RawMessage `json:"rows"`
		Files    []json.RawMessage `json:"files"`
		Messages []json.RawMessage `json:"messages"`
		Count    *int              `json:"count"`
		HasMore  *bool             `json:"has_more"`
		Complete *bool             `json:"complete"`
	}
	_ = json.Unmarshal(data, &page)
	if page.Count != nil {
		count = page.Count
	}
	for _, records := range [][]json.RawMessage{page.Items, page.Records, page.Rows, page.Files, page.Messages} {
		if records != nil {
			n := len(records)
			count = &n
			break
		}
	}
	complete := page.Complete
	if page.HasMore != nil {
		exhausted := !*page.HasMore
		complete = &exhausted
	}
	return count, complete
}

// These are authentication protocol fields, not resource tokens such as a
// document token or pagination cursor. Never persist them in a reusable result.
func containsCredentialField(raw json.RawMessage) bool {
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) == nil && object != nil {
		gitCredential := false
		for key := range object {
			switch strings.ToLower(key) {
			case "giturl", "git_url", "git_http_url", "repository_url":
				gitCredential = true
			}
		}
		for key, value := range object {
			if gitCredential && strings.EqualFold(key, "token") && string(value) != "null" && string(value) != `""` {
				return true
			}
			switch strings.ToLower(key) {
			case "access_token", "refresh_token", "tenant_access_token", "app_access_token", "user_access_token", "user_refresh_token", "app_secret", "client_secret", "authorization", "private_key", "api_key", "apikey", "api_secret", "secret_key", "secret", "pat", "password", "passwd", "session_key", "session_token", "credential", "credentials", "git_credential", "git_info", "gitinfo":
				if string(value) != "null" && string(value) != `""` {
					return true
				}
			}
			if containsCredentialField(value) {
				return true
			}
		}
	}
	var array []json.RawMessage
	if json.Unmarshal(raw, &array) == nil {
		for _, value := range array {
			if containsCredentialField(value) {
				return true
			}
		}
	}
	return false
}
