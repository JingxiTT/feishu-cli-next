// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/core"
)

func TestPaginationHTTPFailureCannotBecomeSuccessfulPage(t *testing.T) {
	for _, status := range []int{403, 500} {
		for _, body := range []string{`{}`, `{"code":0,"data":{"items":[],"has_more":false}}`, `not JSON`, `{"code":99991679,"log_id":"fixture-http-log"}`} {
			for _, later := range []bool{false, true} {
				t.Run(fmt.Sprintf("%d/%s/later=%t", status, body, later), func(t *testing.T) {
					calls := 0
					ac, _ := newTestAPIClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
						calls++
						if later && calls == 1 {
							return jsonResponse(map[string]interface{}{"code": 0, "data": map[string]interface{}{
								"items": []interface{}{map[string]interface{}{"id": "first"}}, "has_more": true, "page_token": "next",
							}}), nil
						}
						contentType := "application/json"
						if body == "not JSON" {
							contentType = "text/plain"
						}
						return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {contentType}}, Body: io.NopCloser(strings.NewReader(body))}, nil
					}))
					result, err := ac.PaginateAll(context.Background(), RawApiRequest{Method: "GET", URL: "/open-apis/fixture/items", As: core.AsUser}, PaginationOptions{PageDelay: -1})
					wantCalls := 1
					if later {
						wantCalls++
					}
					if err == nil || result != nil || calls != wantCalls {
						t.Fatalf("HTTP failure accepted: result=%v err=%v calls=%d", result, err, calls)
					}
					if strings.Contains(body, "99991679") {
						var permission *errs.PermissionError
						if !errors.As(err, &permission) || permission.Code != 99991679 || permission.LogID != "fixture-http-log" {
							t.Fatalf("lost business error precedence: %#v", err)
						}
					} else if status == 500 {
						var server *errs.NetworkError
						if !errors.As(err, &server) || server.Code != status || server.Subtype != errs.SubtypeNetworkServer {
							t.Fatalf("lost HTTP server failure: %#v", err)
						}
					} else {
						var api *errs.APIError
						if !errors.As(err, &api) || api.Code != status {
							t.Fatalf("lost HTTP failure: %#v", err)
						}
					}
				})
			}
		}
	}
}

func TestPaginationLaterFailureNeverReturnsAggregateSuccess(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, failure := range []string{"permission", "transport", "missing_token", "repeated_token"} {
			t.Run(failure+map[bool]string{false: "/aggregate", true: "/stream"}[stream], func(t *testing.T) {
				calls, streamed := 0, 0
				transportErr := errors.New("synthetic lost response")
				ac, _ := newTestAPIClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					if calls == 1 {
						return jsonResponse(map[string]interface{}{"code": 0, "data": map[string]interface{}{
							"items": []interface{}{map[string]interface{}{"id": "first"}}, "has_more": true, "page_token": "next",
						}}), nil
					}
					if calls > 2 {
						t.Fatal("continued after terminal failure")
					}
					switch failure {
					case "permission":
						return jsonResponse(map[string]interface{}{"code": 99991672, "msg": "permission denied", "log_id": "fixture-log"}), nil
					case "transport":
						return nil, transportErr
					case "missing_token":
						return jsonResponse(map[string]interface{}{"code": 0, "data": map[string]interface{}{"has_more": true}}), nil
					default:
						return jsonResponse(map[string]interface{}{"code": 0, "data": map[string]interface{}{"has_more": true, "page_token": "next"}}), nil
					}
				}))
				request := RawApiRequest{Method: "GET", URL: "/open-apis/fixture/items", As: core.AsBot}
				opts := PaginationOptions{PageDelay: -1}
				var result interface{}
				var err error
				if stream {
					result, _, err = ac.StreamPages(context.Background(), request, func(items []interface{}) error {
						streamed += len(items)
						return nil
					}, opts)
				} else {
					result, err = ac.PaginateAll(context.Background(), request, opts)
				}
				if err == nil || result != nil || calls != 2 {
					t.Fatalf("result=%v err=%v calls=%d", result, err, calls)
				}
				if stream && streamed != 1 {
					t.Fatalf("successful first page lost: %d", streamed)
				}
				switch failure {
				case "permission":
					var permission *errs.PermissionError
					if !errors.As(err, &permission) || permission.Code != 99991672 || permission.LogID != "fixture-log" {
						t.Fatalf("lost typed permission evidence: %#v", err)
					}
				case "transport":
					if !errors.Is(err, transportErr) {
						t.Fatalf("lost transport cause: %v", err)
					}
				default:
					var invalid *errs.InternalError
					if !errors.As(err, &invalid) || invalid.Subtype != errs.SubtypeInvalidResponse {
						t.Fatalf("expected invalid_response: %v", err)
					}
				}
			})
		}
	}
}

// Check live heap while the stream is still running: completed pages must be
// collectible, even when the backend keeps supplying more continuation tokens.
func TestStreamPagesReleasesCompletedPagePayloads(t *testing.T) {
	const pageBytes = 1 << 20
	calls, streamed := 0, 0
	ac, _ := newTestAPIClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return jsonResponse(map[string]interface{}{"code": 0, "data": map[string]interface{}{
			"items": []interface{}{strings.Repeat("x", pageBytes)}, "has_more": calls < 20, "page_token": fmt.Sprint(calls),
		}}), nil
	}))
	var baseline uint64
	result, hasItems, err := ac.StreamPages(context.Background(), RawApiRequest{Method: "GET", URL: "/open-apis/fixture/items", As: core.AsUser}, func(items []interface{}) error {
		streamed++
		if streamed == 2 || streamed == 20 {
			runtime.GC()
			var memory runtime.MemStats
			runtime.ReadMemStats(&memory)
			if streamed == 2 {
				baseline = memory.HeapAlloc
			} else if memory.HeapAlloc > baseline+8*pageBytes {
				t.Fatalf("stream retained earlier page payloads: grew by %d bytes", memory.HeapAlloc-baseline)
			}
		}
		return nil
	}, PaginationOptions{PageDelay: -1})
	if err != nil || !hasItems || streamed != 20 {
		t.Fatalf("streamed=%d hasItems=%t error=%v", streamed, hasItems, err)
	}
	if token := result.(map[string]interface{})["data"].(map[string]interface{})["page_token"]; token != "20" {
		t.Fatalf("lost last page: %v", token)
	}
}
