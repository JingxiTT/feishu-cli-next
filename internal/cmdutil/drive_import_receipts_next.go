// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/larksuite/cli/internal/core"
)

// ReconcileDriveImport is used only after a successful native import status GET
// under this context's explicit request identity. Unverifiable proof leaves the
// original unknown outcome intact. Persistence failures are surfaced by the
// invocation summary, so completed jobs are never polled again for a local error.
// Local proof-assembly errors retain the pending receipt and exit 22; returning
// them would retry the completed task poll and discard its observed ready status.
// Dependent permission changes use their separate exact-create proof validator.
//
//nolint:nilerr // Incomplete local proof deliberately leaves the existing unknown receipt unchanged instead of triggering another poll.
func (f *Factory) ReconcileDriveImport(ctx context.Context, ticket string, data map[string]interface{}) error {
	if !f.EnableAgentAttempts || !f.agentAttempts.HasDriveImport(ticket) {
		return nil
	}
	cfg, err := f.Config()
	if err != nil {
		return nil
	}
	origin := core.ResolveEndpoints(cfg.Brand).Open
	req, err := http.NewRequestWithContext(ctx, "GET", origin+"/open-apis/drive/v1/import_tasks/"+url.PathEscape(ticket), nil)
	if err != nil {
		return nil
	}
	binding, err := f.resolveAgentAttemptBinding(req)
	if err != nil {
		return nil
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	f.agentAttempts.ReconcileDriveImport(f.AgentReceiptStore(), binding, origin, ticket, raw)
	return nil
}
