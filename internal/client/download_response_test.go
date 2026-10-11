// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/core"
)

type trackedDownloadBody struct {
	io.Reader
	closed bool
}

func (b *trackedDownloadBody) Close() error { b.closed = true; return nil }

func TestDownloadsRejectAPIErrorAndPreserveFileContent(t *testing.T) {
	const apiError = `{"code":99991679,"msg":"scope rejected"}`
	for _, sdk := range []bool{false, true} {
		for _, tc := range []struct {
			name, body, contentType, disposition string
			wantError                            bool
		}{
			{name: "API error", body: apiError, contentType: "application/json", wantError: true},
			{name: "structured JSON error", body: apiError, contentType: "application/problem+json", wantError: true},
			{name: "missing content type", body: apiError, wantError: true},
			{name: "JSON file", body: `{"items":[1,2]}`, contentType: "application/json"},
			{name: "numeric content code", body: `{"code":42}`, contentType: "application/json"},
			{name: "JSON attachment", body: apiError, contentType: "application/json", disposition: `attachment; filename="fixture.json"`},
			{name: "large JSON", body: `{"items":"` + strings.Repeat("x", downloadErrorProbeLimit*2) + `"}`, contentType: "application/json"},
			{name: "binary", body: "binary bytes", contentType: "application/octet-stream"},
			{name: "binary type with JSON parameter", body: apiError, contentType: `application/octet-stream; note="application/json"`},
		} {
			name := "stream/" + tc.name
			if sdk {
				name = "SDK/" + tc.name
			}
			t.Run(name, func(t *testing.T) {
				body := &trackedDownloadBody{Reader: strings.NewReader(tc.body)}
				rt := roundTripFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: 200, Header: http.Header{
						"Content-Type": []string{tc.contentType}, "Content-Disposition": []string{tc.disposition},
						"X-Tt-Logid": []string{"fixture_log"},
					}, Body: body, ContentLength: int64(len(tc.body))}, nil
				})
				client, _ := newTestAPIClient(t, rt)
				client.HTTP = &http.Client{Transport: rt}
				req := &larkcore.ApiReq{HttpMethod: http.MethodGet, ApiPath: "/open-apis/drive/v1/files/fixture/download"}
				var data []byte
				var err error
				if sdk {
					resp, callErr := client.DoSDKRequest(context.Background(), req, core.AsBot, larkcore.WithFileDownload())
					err = callErr
					if resp != nil {
						data = resp.RawBody
					}
				} else {
					resp, callErr := client.DoStream(context.Background(), req, core.AsBot)
					err = callErr
					if resp != nil {
						data, err = io.ReadAll(resp.Body)
						resp.Body.Close()
					}
				}
				if !body.closed {
					t.Fatal("response body was not closed")
				}
				if tc.wantError {
					problem, ok := errs.ProblemOf(err)
					if !ok || problem.Category != errs.CategoryAuthorization || problem.Code != 99991679 || problem.LogID != "fixture_log" {
						t.Fatalf("wrong typed API failure: %#v, %v", problem, err)
					}
				} else if err != nil || string(data) != tc.body {
					t.Fatalf("file changed: bytes=%d, want=%d, error=%v", len(data), len(tc.body), err)
				}
			})
		}
	}
}
