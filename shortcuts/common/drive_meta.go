// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package common

import (
	"encoding/json"

	"github.com/larksuite/cli/errs"
)

// DriveMeta is the subset of drive metas/batch_query fields used by shortcuts.
type DriveMeta struct {
	Title string
	URL   string
}

// FetchDriveMeta looks up document metadata via the drive metas batch_query API.
func FetchDriveMeta(runtime *RuntimeContext, token, docType string, withURL bool) (DriveMeta, error) {
	body := map[string]interface{}{
		"request_docs": []map[string]interface{}{
			{
				"doc_token": token,
				"doc_type":  docType,
			},
		},
	}
	if withURL {
		body["with_url"] = true
	}

	data, err := runtime.CallReadOnlyAPITyped(
		"POST",
		"/open-apis/drive/v1/metas/batch_query",
		nil,
		body,
	)
	if err != nil {
		return DriveMeta{}, err
	}

	// Decode the map-returning runtime boundary before selecting metadata. A
	// batch response is not ordered evidence of the requested document identity.
	var response struct {
		Metas []struct {
			Token string `json:"doc_token"`
			Type  string `json:"doc_type"`
			Title string `json:"title"`
			URL   string `json:"url"`
		} `json:"metas"`
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return DriveMeta{}, errs.NewInternalError(errs.SubtypeInvalidResponse,
			"encode Drive metadata response: %v", err).WithCause(err)
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return DriveMeta{}, errs.NewInternalError(errs.SubtypeInvalidResponse,
			"decode Drive metadata response: %v", err).WithCause(err)
	}
	if len(response.Metas) == 0 {
		return DriveMeta{}, nil
	}
	var selected DriveMeta
	found := false
	for _, meta := range response.Metas {
		if meta.Token == "" || meta.Type == "" {
			return DriveMeta{}, errs.NewInternalError(errs.SubtypeInvalidResponse,
				"Drive metadata response is missing document identity")
		}
		if meta.Token != token || meta.Type != docType {
			continue
		}
		if found {
			return DriveMeta{}, errs.NewInternalError(errs.SubtypeInvalidResponse,
				"Drive metadata response contains duplicate document identity")
		}
		selected = DriveMeta{Title: meta.Title, URL: meta.URL}
		found = true
	}
	if !found {
		return DriveMeta{}, errs.NewInternalError(errs.SubtypeInvalidResponse,
			"Drive metadata response does not match the requested document")
	}
	return selected, nil
}

// FetchDriveMetaTitle looks up the document title via the drive metas batch_query API.
func FetchDriveMetaTitle(runtime *RuntimeContext, token, docType string) (string, error) {
	meta, err := FetchDriveMeta(runtime, token, docType, false)
	if err != nil {
		return "", err
	}
	return meta.Title, nil
}

// FetchDriveMetaURL looks up the document access URL via the drive metas batch_query API.
func FetchDriveMetaURL(runtime *RuntimeContext, token, docType string) (string, error) {
	meta, err := FetchDriveMeta(runtime, token, docType, true)
	if err != nil {
		return "", err
	}
	return meta.URL, nil
}
