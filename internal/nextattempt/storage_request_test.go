// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextattempt

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDeclaredStoragePutEvidence(t *testing.T) {
	lost := errors.New("synthetic connection lost after storage")
	for _, tt := range []struct {
		name, method, path, outcome string
		status                      int
		err                         error
	}{
		{"ok", "PUT", "/object", Applied, 200, nil},
		{"created", "PUT", "/object", Applied, 201, nil},
		{"no content", "PUT", "/object", Applied, 204, nil},
		{"accepted", "PUT", "/object", Unknown, 202, nil},
		{"server failure", "PUT", "/object", Unknown, 500, nil},
		{"denied", "PUT", "/object", Unapplied, 403, nil},
		{"lost", "PUT", "/object", Unknown, 0, lost},
		{"other path", "PUT", "/other", Unknown, 200, nil},
		{"other method", "POST", "/object", Unknown, 200, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			tr, store, journal := testTransport(t, func(req *http.Request) (*http.Response, error) {
				calls++
				body, err := io.ReadAll(req.Body)
				if err != nil || string(body) != "private object bytes" {
					t.Fatalf("request bytes changed: %q %v", body, err)
				}
				if tt.err != nil {
					return nil, tt.err
				}
				return response(tt.status, "opaque storage response"), nil
			})
			ctx := WithStoragePutRequest(context.Background(), "/object")
			req, err := http.NewRequestWithContext(ctx, tt.method, "https://fixture.invalid"+tt.path+"?signature=private", strings.NewReader("private object bytes"))
			if err != nil {
				t.Fatal(err)
			}
			resp, err := tr.RoundTrip(req)
			if !errors.Is(err, tt.err) || calls != 1 {
				t.Fatalf("dispatch/error: %d %v", calls, err)
			}
			if resp != nil {
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil || string(body) != "opaque storage response" {
					t.Fatalf("response bytes changed: %q %v", body, err)
				}
			}
			receipts, err := store.List()
			if err != nil || len(receipts) != 1 || receipts[0].Outcome != tt.outcome || journal.Summary().Outcome != tt.outcome {
				t.Fatalf("outcome: %v %v", receipts, err)
			}
		})
	}
}

func TestDeclaredStoragePutPreservesCancellation(t *testing.T) {
	tr, _, journal := testTransport(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("canceled request dispatched")
		return nil, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	ctx = WithStoragePutRequest(ctx, "/object")
	cancel()
	req, err := http.NewRequestWithContext(ctx, "PUT", "https://fixture.invalid/object", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tr.RoundTrip(req); !errors.Is(err, context.Canceled) || journal.Summary().Outcome != Unapplied {
		t.Fatalf("cancellation outcome: %v %v", journal.Summary(), err)
	}
}
