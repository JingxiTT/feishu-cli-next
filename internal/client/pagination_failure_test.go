// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/core"
)

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
