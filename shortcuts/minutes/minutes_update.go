// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package minutes

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/validate"
	"github.com/larksuite/cli/shortcuts/common"
)

const minutesUpdateNoEditPermissionCode = 2091005

// MinutesUpdate updates the title (topic) of a minute.
var MinutesUpdate = common.Shortcut{
	Service:           "minutes",
	Command:           "+update",
	Description:       "Update a minute's title",
	Risk:              "write",
	Scopes:            []string{"minutes:minutes:update"},
	ConditionalScopes: []string{minutesDetailBasicScope},
	AuthTypes:         []string{"user"},
	HasFormat:         true,
	Flags: []common.Flag{
		{Name: "skip-unchanged", Type: "bool", Desc: "read current content and skip an identical replacement (requires read scope)"},
		{Name: "minute-token", Desc: "minute token", Required: true},
		{Name: "topic", Desc: "new minute title", Required: true},
	},
	Validate: func(ctx context.Context, runtime *common.RuntimeContext) error {
		minuteToken := strings.TrimSpace(runtime.Str("minute-token"))
		if minuteToken == "" {
			return errs.NewValidationError(errs.SubtypeInvalidArgument, "--minute-token is required").WithParam("--minute-token")
		}
		if err := validate.ResourceName(minuteToken, "--minute-token"); err != nil {
			return errs.NewValidationError(errs.SubtypeInvalidArgument, "%s", err).WithParam("--minute-token")
		}
		if strings.TrimSpace(runtime.Str("topic")) == "" {
			return errs.NewValidationError(errs.SubtypeInvalidArgument, "--topic is required").WithParam("--topic")
		}
		return minutesUnchangedScope(runtime, minutesDetailBasicScope)
	},
	DryRun: func(ctx context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
		minuteToken := strings.TrimSpace(runtime.Str("minute-token"))
		preview := common.NewDryRunAPI()
		if runtime.Bool("skip-unchanged") {
			preview.GET(fmt.Sprintf("/open-apis/minutes/v1/minutes/%s", validate.EncodePathSegment(runtime.Str("minute-token"))))
		}
		return preview.
			PATCH(fmt.Sprintf("/open-apis/minutes/v1/minutes/%s", validate.EncodePathSegment(minuteToken))).
			Body(map[string]interface{}{"topic": runtime.Str("topic")})
	},
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		minuteToken := strings.TrimSpace(runtime.Str("minute-token"))
		topic := runtime.Str("topic")

		if runtime.Bool("skip-unchanged") {
			same, err := minutesTitleUnchanged(runtime, minuteToken, topic)
			if err != nil {
				return err
			}
			if same {
				runtime.OutFormat(map[string]any{"minute_token": minuteToken, "topic": topic, "unchanged": true}, nil, nil)
				return nil
			}
		}

		body := map[string]interface{}{
			"topic": topic,
		}

		_, err := runtime.CallAPITyped(http.MethodPatch,
			fmt.Sprintf("/open-apis/minutes/v1/minutes/%s", validate.EncodePathSegment(minuteToken)),
			nil, body)
		if err != nil {
			return minutesUpdateError(err, minuteToken)
		}

		outData := map[string]interface{}{
			"minute_token": minuteToken,
			"topic":        topic,
		}

		runtime.OutFormat(outData, nil, nil)
		return nil
	},
}

func minutesUpdateError(err error, minuteToken string) error {
	p, ok := errs.ProblemOf(err)
	if !ok || p.Code != minutesUpdateNoEditPermissionCode {
		return err
	}
	p.Message = fmt.Sprintf("No edit permission for minute %q: cannot update the title.", minuteToken)
	p.Hint = fmt.Sprintf("Ask the user before running: minutes +apply-permission --minute-token %s --perm edit", minuteToken)
	return err
}
