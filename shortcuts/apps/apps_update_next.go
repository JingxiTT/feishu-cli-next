// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"encoding/json"

	"github.com/larksuite/cli/errs"
)

// appsUpdateUnchanged compares only explicitly requested replacement fields.
// Missing fields cannot prove equality; an absent app object is not a target.
func appsUpdateUnchanged(appID string, current, requested map[string]interface{}) (bool, error) {
	var response struct {
		App *struct {
			AppID       string  `json:"app_id"`
			Name        *string `json:"name"`
			Description *string `json:"description"`
		} `json:"app"`
	}
	raw, err := json.Marshal(current)
	if err != nil {
		return false, errs.NewInternalError(errs.SubtypeInvalidResponse, "cannot decode current app").WithCause(err)
	}
	if err = json.Unmarshal(raw, &response); err != nil {
		return false, errs.NewInternalError(errs.SubtypeInvalidResponse, "cannot decode current app").WithCause(err)
	}
	if response.App == nil || response.App.AppID != appID {
		return false, errs.NewInternalError(errs.SubtypeInvalidResponse, "current app response does not identify the requested app")
	}
	for key, value := range requested {
		var observed *string
		switch key {
		case "name":
			observed = response.App.Name
		case "description":
			observed = response.App.Description
		default:
			return false, nil
		}
		if observed == nil || *observed != value {
			return false, nil
		}
	}
	return len(requested) > 0, nil
}
