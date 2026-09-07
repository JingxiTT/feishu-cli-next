// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/validate"
	"github.com/larksuite/cli/shortcuts/common"
)

// AppsUpdate partially updates an app's name / description.
var AppsUpdate = common.Shortcut{
	Service:     appsService,
	Command:     "+update",
	Description: "Partially update an app (only provided fields are sent)",
	Risk:        "write",
	Tips: []string{
		`Example: lark-cli apps +update --app-id <app_id> --name "新名称"`,
		`Example: lark-cli apps +update --app-id <app_id> --description "..."`,
	},
	Scopes:            []string{"spark:app:write"},
	ConditionalScopes: []string{"spark:app:read"},
	AuthTypes:         []string{"user"},
	HasFormat:         true,
	Flags: []common.Flag{
		{Name: "app-id", Desc: "app ID", Required: true},
		{Name: "name", Desc: "new app display name"},
		{Name: "description", Desc: "new app description"},
	},
	Validate: func(ctx context.Context, rctx *common.RuntimeContext) error {
		if strings.TrimSpace(rctx.Str("app-id")) == "" {
			return appsValidationParamError("--app-id", "--app-id is required")
		}
		body := buildAppsUpdateBody(rctx)
		if len(body) == 0 {
			return appsValidationError("provide at least one of --name or --description").
				WithParams(
					appsInvalidParam("--name", "provide at least one of --name or --description"),
					appsInvalidParam("--description", "provide at least one of --name or --description"),
				)
		}
		return nil
	},
	DryRun: func(ctx context.Context, rctx *common.RuntimeContext) *common.DryRunAPI {
		appID := strings.TrimSpace(rctx.Str("app-id"))
		return common.NewDryRunAPI().
			PATCH(fmt.Sprintf("%s/apps/%s", apiBasePath, validate.EncodePathSegment(appID))).
			Desc("Update an app").
			Body(buildAppsUpdateBody(rctx))
	},
	Execute: func(ctx context.Context, rctx *common.RuntimeContext) error {
		appID := strings.TrimSpace(rctx.Str("app-id"))
		path := fmt.Sprintf("%s/apps/%s", apiBasePath, validate.EncodePathSegment(appID))
		body := buildAppsUpdateBody(rctx)
		if rctx.Factory.EnableAgentAttempts {
			current, err := rctx.CallAPITyped("GET", path, nil, nil)
			if err != nil {
				return withAppsHint(err, appIDListHint)
			}
			unchanged, err := appsUpdateUnchanged(appID, current, body)
			if err != nil {
				return err
			}
			if unchanged {
				rctx.OutFormat(current, nil, func(w io.Writer) { fmt.Fprintf(w, "unchanged: %s\n", appID) })
				return nil
			}
		}
		data, err := rctx.CallAPITyped("PATCH", path, nil, body)
		if err != nil {
			return withAppsHint(err, appIDListHint)
		}
		rctx.OutFormat(data, nil, func(w io.Writer) {
			fmt.Fprintf(w, "updated: %s\n", common.GetString(data, "app", "app_id"))
		})
		return nil
	},
}

func buildAppsUpdateBody(rctx *common.RuntimeContext) map[string]interface{} {
	body := map[string]interface{}{}
	if v := strings.TrimSpace(rctx.Str("name")); v != "" {
		body["name"] = v
	}
	if v := strings.TrimSpace(rctx.Str("description")); v != "" {
		body["description"] = v
	}
	return body
}

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
