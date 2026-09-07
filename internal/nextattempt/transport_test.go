// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextattempt

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/vfs"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func testTransport(t *testing.T, fn roundTripFunc) (*Transport, Store, *Journal) {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	s := Store{Dir: filepath.Join(root, "receipts")}
	j := &Journal{}
	tr := &Transport{Base: fn, Store: func() Store { return s }, Journal: j, IsMutation: func(*http.Request) bool { return true }, Resolve: func(*http.Request) (Binding, error) {
		return Binding{Environment: "https://fixture.invalid", AppID: "app-test", Identity: "bot"}, nil
	}}
	return tr, s, j
}
func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}
func TestEvidenceBeforeDispatchAndPreservation(t *testing.T) {
	var s Store
	body := `{"code":0,"data":{"id":"synthetic","text":"秘密"}}`
	tr, store, j := testTransport(t, func(req *http.Request) (*http.Response, error) {
		records, err := s.List()
		if err != nil || len(records) != 1 || records[0].Outcome != Unknown {
			t.Fatalf("before dispatch: %v %v", records, err)
		}
		if err := s.Cleanup(records[0].ID); err == nil {
			t.Fatal("cleanup acquired active attempt lock")
		}
		payload, _ := io.ReadAll(req.Body)
		if string(payload) != "private-content" {
			t.Fatalf("request changed: %q", payload)
		}
		return response(200, body), nil
	})
	s = store
	req, _ := http.NewRequest("POST", "https://fixture.invalid/create?secret=not-recorded", strings.NewReader("private-content"))
	req.Header.Set("Authorization", "private-token")
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	actual, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(actual) != body {
		t.Fatalf("response changed %q", actual)
	}
	records, err := s.List()
	if err != nil || len(records) != 1 || records[0].Outcome != Applied {
		t.Fatalf("records %v %v", records, err)
	}
	if j.Summary().ExitCode() != 0 {
		t.Fatal(j.Summary())
	}
	raw, _ := os.ReadFile(filepath.Join(s.Dir, records[0].ID+".json"))
	for _, secret := range []string{"private-content", "private-token", "not-recorded", "秘密"} {
		if strings.Contains(string(raw), secret) {
			t.Fatalf("journal leaked %s", secret)
		}
	}
	info, _ := os.Stat(filepath.Join(s.Dir, records[0].ID+".json"))
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatal(info.Mode())
	}
}
func TestResponseEvidence(t *testing.T) {
	tests := []struct {
		name, path, body, outcome string
		status                    int
	}{
		{"success", "/create", `{"code":0}`, Applied, 200},
		{"gateway", "/create", `{"code":99991679}`, Unapplied, 200},
		{"domain error", "/create", `{"code":1234}`, Unknown, 200},
		{"server error", "/create", `{"code":0}`, Unknown, 500},
		{"malformed", "/create", `invalid`, Unknown, 200},
		{"missing code", "/create", `{}`, Unknown, 200},
		{"batch", "/batch_create", `{"code":0}`, Unknown, 200},
		{"item failure", "/create", `{"code":0,"data":{"items":[{"code":1}]}}`, Unknown, 200},
		{"accepted", "/create", `{"code":0}`, Unknown, 202},
		{"permission", "/create", `{}`, Unapplied, 403},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr, _, j := testTransport(t, func(*http.Request) (*http.Response, error) { return response(tt.status, tt.body), nil })
			req, _ := http.NewRequest("POST", "https://fixture.invalid"+tt.path, nil)
			resp, err := tr.RoundTrip(req)
			if err != nil {
				t.Fatal(err)
			}
			raw, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if string(raw) != tt.body || j.Summary().Outcome != tt.outcome {
				t.Fatalf("%q %+v", raw, j.Summary())
			}
		})
	}
}
func TestTransportErrorAndCancellation(t *testing.T) {
	sentinel := errors.New("connection lost after acceptance")
	tr, s, j := testTransport(t, func(*http.Request) (*http.Response, error) { return nil, sentinel })
	req, _ := http.NewRequest("POST", "https://fixture.invalid/create", nil)
	if _, err := tr.RoundTrip(req); !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	if j.Summary().Unknown != 1 || j.Summary().ExitCode() != 22 {
		t.Fatal(j.Summary())
	}
	records, _ := s.List()
	if records[0].Outcome != Unknown {
		t.Fatal(records)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req = req.WithContext(ctx)
	if _, err := tr.RoundTrip(req); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if j.Summary().Unapplied != 1 {
		t.Fatal(j.Summary())
	}
}

type failRenameFS struct{ vfs.FS }

func (f failRenameFS) Rename(string, string) error { return errors.New("synthetic disk failure") }
func TestRemoteSuccessThenSaveFailure(t *testing.T) {
	previous := vfs.DefaultFS
	t.Cleanup(func() { vfs.DefaultFS = previous })
	tr, s, j := testTransport(t, func(*http.Request) (*http.Response, error) {
		vfs.DefaultFS = failRenameFS{previous}
		return response(200, `{"code":0}`), nil
	})
	req, _ := http.NewRequest("POST", "https://fixture.invalid/create", nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	summary := j.Summary()
	if summary.Applied != 1 || !summary.PersistenceFailed || summary.ExitCode() != 22 {
		t.Fatal(summary)
	}
	records, err := s.List()
	if err != nil || records[0].Outcome != Unknown {
		t.Fatalf("%v %v", records, err)
	}
}
func TestPersistenceFailurePreventsDispatch(t *testing.T) {
	called := false
	tr, s, _ := testTransport(t, func(*http.Request) (*http.Response, error) { called = true; return response(200, `{"code":0}`), nil })
	if err := os.WriteFile(s.Dir, []byte("block directory"), 0600); err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest("POST", "https://fixture.invalid/create", nil)
	_, err := tr.RoundTrip(req)
	if called || err == nil {
		t.Fatalf("called=%v err=%v", called, err)
	}
	p, ok := errs.ProblemOf(err)
	if !ok || p.Subtype != errs.SubtypeInvalidConfig {
		t.Fatal(err)
	}
}
func TestJournalPartial(t *testing.T) {
	j := &Journal{}
	j.Add(Receipt{Outcome: Applied})
	j.Add(Receipt{Outcome: Unapplied})
	if j.Summary().Outcome != "partial" || j.Summary().ExitCode() != 21 {
		t.Fatal(j.Summary())
	}
}
func TestCorruptionAndExplicitCleanup(t *testing.T) {
	tr, s, _ := testTransport(t, func(*http.Request) (*http.Response, error) { return response(200, `{"code":0}`), nil })
	req, _ := http.NewRequest("POST", "https://fixture.invalid/create", nil)
	resp, _ := tr.RoundTrip(req)
	resp.Body.Close()
	records, _ := s.List()
	id := records[0].ID
	if _, err := s.Inspect("../../secret"); err == nil {
		t.Fatal("unsafe ID accepted")
	}
	if err := s.Cleanup(id); err != nil {
		t.Fatal(err)
	}
	records, _ = s.List()
	if len(records) != 0 {
		t.Fatal(records)
	}
	if err := os.WriteFile(filepath.Join(s.Dir, id+".json"), []byte(`{"version":99}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.List(); err == nil {
		t.Fatal("corrupt state accepted")
	}
	if err := s.Cleanup(id); err == nil {
		t.Fatal("corrupt state silently deleted")
	}
}

type interruptedBody struct {
	sent   bool
	err    error
	closed bool
}

func (b *interruptedBody) Read(p []byte) (int, error) {
	if !b.sent {
		b.sent = true
		return copy(p, `{"code":0`), b.err
	}
	return 0, b.err
}
func (b *interruptedBody) Close() error { b.closed = true; return nil }
func TestInterruptedResponsePreservesBytesAndCause(t *testing.T) {
	failure := errors.New("synthetic body interruption")
	body := &interruptedBody{err: failure}
	tr, _, j := testTransport(t, func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: body}, nil })
	req, _ := http.NewRequest("POST", "https://fixture.invalid/create", nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(raw) != `{"code":0` || !errors.Is(err, failure) || !body.closed || j.Summary().Unknown != 1 {
		t.Fatalf("%q %v %v %+v", raw, err, body.closed, j.Summary())
	}
}
func TestSymlinkStorageRejectedBeforeCreation(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()
	if err := os.Symlink(target, filepath.Join(root, "next")); err != nil {
		t.Skip(err)
	}
	s := Store{Dir: filepath.Join(root, "next", "receipts")}
	if err := s.validateDir(true); err == nil {
		t.Fatal("symlink storage accepted")
	}
	if _, err := os.Stat(filepath.Join(target, "receipts")); !os.IsNotExist(err) {
		t.Fatal("created directory through symlink")
	}
}
