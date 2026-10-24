// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package event

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/client"
	"github.com/larksuite/cli/internal/core"
	eventlib "github.com/larksuite/cli/internal/event"
	"github.com/larksuite/cli/internal/nextattempt"
)

// consumeRuntime routes event.APIClient calls through the shared client.APIClient with a pinned identity.
type consumeRuntime struct {
	client         *client.APIClient
	accessIdentity core.Identity
}

func (r *consumeRuntime) CallAPI(ctx context.Context, method, path string, body interface{}) (json.RawMessage, error) {
	// Event consumption is read-risk, but lifecycle registrations are writes.
	// The exact request marker preserves the shared journal without journaling
	// per-event enrichment reads or creating a parallel streaming log.
	if method == "POST" {
		apiPath, _, _ := strings.Cut(path, "?")
		// net/http compares URL.Path, whose path segments are already decoded.
		if decoded, err := url.PathUnescape(apiPath); err == nil {
			apiPath = decoded
		}
		ctx = nextattempt.WithMutationRequest(ctx, method, apiPath)
	}
	resp, err := r.client.DoAPI(ctx, client.RawApiRequest{
		Method: method,
		URL:    path,
		Data:   body,
		As:     r.accessIdentity,
	})
	if err != nil {
		if _, ok := errs.ProblemOf(err); ok {
			return nil, err
		}
		return nil, errs.NewNetworkError(errs.SubtypeNetworkTransport,
			"api %s %s: %s", method, path, err).WithCause(err)
	}
	// Non-JSON HTTP errors (gateway text/plain 404 etc.) skip OAPI envelope parsing.
	ct := resp.Header.Get("Content-Type")
	if resp.StatusCode >= 400 && !client.IsJSONContentType(ct) && ct != "" {
		const maxBodyEcho = 256
		body := string(resp.RawBody)
		body = eventlib.TruncateDiagnostic(body, maxBodyEcho, "…(truncated)")
		if resp.StatusCode >= 500 {
			return nil, errs.NewNetworkError(errs.SubtypeNetworkServer,
				"api %s %s returned %d: %s", method, path, resp.StatusCode, body).WithRetryable()
		}
		return nil, errs.NewInternalError(errs.SubtypeInvalidResponse,
			"api %s %s returned %d: %s", method, path, resp.StatusCode, body)
	}
	result, err := client.ParseJSONResponse(resp)
	if err != nil {
		if _, ok := errs.ProblemOf(err); ok {
			return nil, err
		}
		return nil, errs.NewInternalError(errs.SubtypeInvalidResponse,
			"api %s %s: %s", method, path, err).WithCause(err)
	}
	if apiErr := r.client.CheckResponse(result, r.accessIdentity); apiErr != nil {
		return json.RawMessage(resp.RawBody), apiErr
	}
	return json.RawMessage(resp.RawBody), nil
}
