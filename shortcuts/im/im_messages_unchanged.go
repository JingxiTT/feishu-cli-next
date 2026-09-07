// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package im

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/validate"
	"github.com/larksuite/cli/shortcuts/common"
)

type editableMessage struct {
	MessageID   string `json:"message_id"`
	ChatID      string `json:"chat_id"`
	MessageType string `json:"msg_type"`
	UpdateTime  string `json:"update_time"`
	Deleted     bool   `json:"deleted"`
	Body        struct {
		Content string `json:"content"`
	} `json:"body"`
}

// unchangedMessage only suppresses a full replacement when the backend returned
// the same target, type and complete JSON value. The opt-in read must succeed
// before any replacement is dispatched. This point-in-time comparison is not
// a conditional write or concurrency guarantee.
func unchangedMessage(runtime *common.RuntimeContext, id, kind, content string) (editableMessage, bool, error) {
	data, err := runtime.DoAPIJSONTyped(http.MethodGet,
		fmt.Sprintf("/open-apis/im/v1/messages/%s", validate.EncodePathSegment(id)), nil, nil)
	if err != nil {
		return editableMessage{}, false, err
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return editableMessage{}, false, errs.NewInternalError(errs.SubtypeInvalidResponse, "cannot decode message preflight").WithCause(err)
	}
	var result struct {
		Items []editableMessage `json:"items"`
	}
	if err := json.Unmarshal(encoded, &result); err != nil {
		return editableMessage{}, false, errs.NewInternalError(errs.SubtypeInvalidResponse, "cannot decode message preflight").WithCause(err)
	}
	if len(result.Items) != 1 {
		return editableMessage{}, false, errs.NewInternalError(errs.SubtypeInvalidResponse, "message preflight did not return exactly one readable message")
	}
	current := result.Items[0]
	if current.MessageID != id || current.MessageType == "" || current.Deleted || !json.Valid([]byte(current.Body.Content)) {
		return editableMessage{}, false, errs.NewInternalError(errs.SubtypeInvalidResponse, "message preflight returned a different, deleted or unreadable target")
	}
	return current, current.MessageType == kind && equalMessageJSON(current.Body.Content, content), nil
}

func equalMessageJSON(left, right string) bool {
	decode := func(raw string) (any, bool) {
		if !json.Valid([]byte(raw)) {
			return nil, false
		}
		decoder := json.NewDecoder(strings.NewReader(raw))
		decoder.UseNumber()
		var value any
		err := decoder.Decode(&value)
		return value, err == nil
	}
	a, validA := decode(left)
	b, validB := decode(right)
	return validA && validB && reflect.DeepEqual(a, b)
}
