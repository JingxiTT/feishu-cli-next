// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package drive

import (
	"github.com/larksuite/cli/shortcuts/common"
)

// driveImportPermissionGrant prevents a dependent permission write when native
// task proof is contradictory or its receipt could not be durably saved.
func driveImportPermissionGrant(runtime *common.RuntimeContext, status driveImportStatus, verified bool) map[string]interface{} {
	if !runtime.IsBot() {
		return nil
	}
	persistenceFailed := runtime.Factory != nil && runtime.Factory.AgentAttemptSummary().PersistenceFailed
	if !verified || persistenceFailed {
		return map[string]interface{}{
			"status":  common.PermissionGrantSkipped,
			"message": "Import completion evidence is incomplete or could not be saved; no current-user permission change was attempted.",
			"hint":    "Inspect the import result and receipt before granting permissions. Task result: " + driveImportTaskResultCommand(runtime, status.Ticket),
		}
	}
	return common.AutoGrantCurrentUserDrivePermission(runtime, status.Token, status.DocType)
}
