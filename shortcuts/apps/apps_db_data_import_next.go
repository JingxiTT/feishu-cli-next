// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"encoding/json"
	"strings"

	"github.com/larksuite/cli/errs"
)

// Generic result artifacts carry bindings and extent, not an import row schema.
// Keep the native CSV/JSON import route explicit rather than silently importing
// an artifact envelope as a database record.
func rejectResultArtifactImport(content []byte) error {
	var header struct {
		Schema string `json:"schema"`
	}
	if json.Unmarshal(content, &header) == nil && strings.HasPrefix(header.Schema, "feishu-cli-next/result/") {
		return errs.NewValidationError(errs.SubtypeInvalidArgument, "result artifacts are not native database import payloads").WithParam("--file").WithHint("Use a native CSV/JSON data export; verify artifact binding and completeness before converting one to import rows.")
	}
	return nil
}
