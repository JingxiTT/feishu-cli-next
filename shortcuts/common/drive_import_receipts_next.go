// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package common

import (
	"context"
	"encoding/json"

	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/nextattempt"
)

// WithDriveImportStatusRequest prevents redirects from supplying native import
// completion proof; other requests retain their existing redirect policy.
func WithDriveImportStatusRequest(ctx context.Context, apiPath string) context.Context {
	return nextattempt.WithDriveImportStatusRequest(ctx, apiPath)
}

// DriveImportCompletionMatches validates proof for dependent domain actions,
// including legacy runtimes with no receipt journal. It never changes receipts.
func DriveImportCompletionMatches(ticket string, requestBody, responseData map[string]interface{}) bool {
	request, err := json.Marshal(requestBody)
	if err != nil {
		return false
	}
	response, err := json.Marshal(responseData)
	return err == nil && nextattempt.DriveImportCompletionMatches(ticket, request, response)
}

// ReconcileDriveImport confirms only an import created by this invocation, using
// a strict native completion result just read through this same runtime.
func (ctx *RuntimeContext) ReconcileDriveImport(ticket string, data map[string]interface{}) error {
	if ctx.Factory == nil {
		return nil
	}
	return ctx.Factory.ReconcileDriveImport(core.WithRequestIdentity(ctx.ctx, ctx.As()), ticket, data)
}
