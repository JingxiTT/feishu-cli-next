// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextattempt

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/larksuite/cli/internal/transport"
)

// Summary is invocation evidence; a local save failure never erases a remotely
// confirmed result, but always makes process success inappropriate.
type Summary struct {
	Receipts          []Receipt `json:"receipts"`
	Outcome           string    `json:"outcome"`
	Applied           int       `json:"applied"`
	Unapplied         int       `json:"unapplied"`
	Unknown           int       `json:"unknown"`
	PersistenceFailed bool      `json:"persistence_failed,omitempty"`
}
type Journal struct {
	mu       sync.Mutex
	receipts []Receipt
}

func (j *Journal) Add(r Receipt) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.receipts = append(j.receipts, r)
}
func (j *Journal) Summary() Summary {
	j.mu.Lock()
	defer j.mu.Unlock()
	s := Summary{Receipts: append([]Receipt(nil), j.receipts...)}
	for _, r := range j.receipts {
		switch r.Outcome {
		case Applied:
			s.Applied++
		case Unapplied:
			s.Unapplied++
		default:
			s.Unknown++
		}
		s.PersistenceFailed = s.PersistenceFailed || r.PersistenceFailed
	}
	switch {
	case s.Unknown > 0:
		s.Outcome = Unknown
	case s.Applied > 0 && s.Unapplied > 0:
		s.Outcome = "partial"
	case s.Unapplied > 0:
		s.Outcome = Unapplied
	case s.Applied > 0:
		s.Outcome = Applied
	}
	return s
}
func (s Summary) ExitCode() int {
	switch {
	case s.Unknown > 0 || s.PersistenceFailed:
		return 22
	case s.Applied > 0 && s.Unapplied > 0:
		return 21
	case s.Unapplied > 0:
		return 20
	default:
		return 0
	}
}

// Transport wraps the existing client transport; it neither sends nor retries
// independently. Resolve runs only for potential mutations, after auth traffic
// and reads have been excluded without touching credential resolution.
type Transport struct {
	Base       http.RoundTripper
	Store      func() Store
	Journal    *Journal
	IsMutation func(*http.Request) bool
	Resolve    func(*http.Request) (Binding, error)
}

func (t *Transport) BaseRoundTripper() http.RoundTripper { return t.Base }
func (t *Transport) WithBaseRoundTripper(base http.RoundTripper) http.RoundTripper {
	cloned := *t
	cloned.Base = base
	return &cloned
}

var _ transport.RoundTripperDecorator = (*Transport)(nil)

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !t.IsMutation(req) {
		return t.Base.RoundTrip(req)
	}
	binding, err := t.Resolve(req)
	if err != nil {
		return nil, err
	}
	store := t.Store()
	receipt, lock, err := store.begin(binding, req.Method)
	if err != nil {
		return nil, err
	}
	defer lock.Unlock()
	// A pre-dispatch canceled request is provably unapplied. Once dispatched,
	// every transport failure remains unknown, even connection errors.
	var response *http.Response
	if err = req.Context().Err(); err != nil {
		receipt.Outcome, receipt.Evidence = Unapplied, "canceled_before_dispatch"
	} else {
		response, err = t.Base.RoundTrip(req)
		classifyResponse(&receipt, response, err, req.URL.Path)
	}
	saveErr := store.finish(receipt)
	if saveErr != nil {
		receipt.PersistenceFailed = true
	}
	t.Journal.Add(receipt)
	// Preserve original response and errors. The command boundary consults the
	// journal even when a command consumes or translates the HTTP response.
	return response, err
}

type replayBody struct {
	io.Reader
	io.Closer
}

const responseEvidenceLimit = 1 << 20

func classifyResponse(r *Receipt, resp *http.Response, requestErr error, path string) {
	r.Outcome, r.Evidence = Unknown, "transport_error"
	if resp != nil {
		r.HTTPStatus = resp.StatusCode
	}
	if requestErr != nil || resp == nil {
		return
	}
	r.Evidence = "unclassified_response"
	// Authentication, authorization and throttling rejections occur before an
	// endpoint executes. Other statuses (including 5xx) are not effect evidence.
	if resp.StatusCode == 401 || resp.StatusCode == 403 || resp.StatusCode == 429 {
		r.Outcome, r.Evidence = Unapplied, "http_rejected"
		return
	}
	if resp.Body == nil {
		return
	}
	original := resp.Body
	data, readErr := io.ReadAll(io.LimitReader(original, responseEvidenceLimit+1))
	// Replay exactly what was consumed, including the original terminal error.
	var tail io.Reader = original
	if readErr != nil {
		tail = &errorReader{err: readErr}
	}
	resp.Body = &replayBody{Reader: io.MultiReader(bytes.NewReader(data), tail), Closer: original}
	if readErr != nil {
		r.Evidence = "response_read_error"
		return
	}
	if len(data) > responseEvidenceLimit {
		r.Evidence = "response_too_large"
		return
	}
	var envelope struct {
		Code *int            `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(data, &envelope) != nil || envelope.Code == nil {
		return
	}
	r.APICode = envelope.Code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return
	}
	if resp.StatusCode == http.StatusAccepted {
		r.Evidence = "accepted_requires_reconciliation"
		return
	}
	if *envelope.Code == 0 {
		// A successful batch envelope may contain per-item errors or async work.
		// Domain owners must interpret these before claiming complete application.
		if strings.Contains(strings.ToLower(path), "batch") || strings.Contains(strings.ToLower(path), "async") || hasItemFailure(envelope.Data) {
			r.Evidence = "batch_or_async_response_requires_reconciliation"
			return
		}
		r.Outcome, r.Evidence = Applied, "api_success_response"
		return
	}
	// Only common gateway rejection codes establish non-application. Arbitrary
	// domain error codes can follow partial effects and remain unknown.
	switch *envelope.Code {
	case 99991661, 99991663, 99991668, 99991671, 99991672, 99991679:
		r.Outcome, r.Evidence = Unapplied, "api_gateway_rejected"
	default:
		r.Evidence = "api_error_requires_reconciliation"
	}
}

type errorReader struct{ err error }

func (r *errorReader) Read([]byte) (int, error) { return 0, r.err }
func hasItemFailure(data []byte) bool {
	var value any
	if json.Unmarshal(data, &value) != nil {
		return false
	}
	var failed func(any) bool
	failed = func(v any) bool {
		switch x := v.(type) {
		case map[string]any:
			for k, v := range x {
				if (k == "code" && v != float64(0)) || ((k == "success" || k == "ok") && v == false) {
					return true
				}
				if (k == "failed_items" || k == "failures" || k == "errors") && v != nil {
					if a, ok := v.([]any); !ok || len(a) > 0 {
						return true
					}
				}
				if failed(v) {
					return true
				}
			}
		case []any:
			for _, v := range x {
				if failed(v) {
					return true
				}
			}
		}
		return false
	}
	return failed(value)
}
