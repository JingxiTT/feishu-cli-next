// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package client

import (
	"encoding/json"
	"mime"
	"net/http"
	"strings"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"

	"github.com/larksuite/cli/internal/core"
)

const downloadErrorProbeLimit = 64 << 10

func probeDownloadError(header http.Header) bool {
	// A named attachment is file content, even when its JSON resembles an API error.
	if larkcore.FileNameByHeader(header) != "" {
		return false
	}
	contentType := header.Get("Content-Type")
	if contentType == "" {
		return true
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && (mediaType == "application/json" || mediaType == "text/json" || strings.HasSuffix(mediaType, "+json"))
}

func (c *APIClient) checkDownloadError(header http.Header, body []byte, as core.Identity) error {
	// Only complete, small gateway envelopes are inspected. Larger JSON remains
	// file content; expand this bound if the gateway starts returning larger errors.
	if len(body) > downloadErrorProbeLimit || !probeDownloadError(header) {
		return nil
	}
	var envelope struct {
		Code *int    `json:"code"`
		Msg  *string `json:"msg"`
	}
	if json.Unmarshal(body, &envelope) == nil && envelope.Code != nil && *envelope.Code != 0 && envelope.Msg != nil {
		result, err := ParseJSONResponse(&larkcore.ApiResp{RawBody: body})
		if err != nil {
			return WrapJSONResponseParseError(err, body)
		}
		if fields, ok := result.(map[string]interface{}); ok {
			if logID := streamLogID(header); logID != "" {
				fields["log_id"] = logID
			}
		}
		return c.CheckResponse(result, as)
	}
	return nil
}
