// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package application

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/shortcuts/common"
)

// Keep the original maps for lossless output, projecting only fields used for
// targeting. Unknown fields in replacement objects must also prevent a no-op.
type slashCommandState struct {
	Command     string `json:"command"`
	ID          string `json:"command_id"`
	Description struct {
		DefaultValue string `json:"default_value"`
	} `json:"description"`
}

type slashCommandListState struct {
	Items   []json.RawMessage `json:"items"`
	HasMore bool              `json:"has_more"`
}

func decodeCommandState(raw interface{}) (slashCommandState, error) {
	var state slashCommandState
	encoded, err := json.Marshal(raw)
	if err == nil {
		err = json.Unmarshal(encoded, &state)
	}
	if err != nil {
		return state, errs.NewAPIError(errs.SubtypeInvalidResponse, "invalid slash command response").WithCause(err)
	}
	if state.ID == "" || state.Command == "" {
		return state, errs.NewAPIError(errs.SubtypeInvalidResponse, "slash command response is missing command or command_id")
	}
	return state, nil
}

func commandList(runtime *common.RuntimeContext) (map[string]interface{}, []interface{}, error) {
	data, err := runtime.CallAPITyped("GET", slashCommandBasePath, nil, nil)
	if err != nil {
		return nil, nil, err
	}
	var state slashCommandListState
	encoded, err := json.Marshal(data)
	if err == nil {
		err = json.Unmarshal(encoded, &state)
	}
	if err != nil {
		return nil, nil, errs.NewAPIError(errs.SubtypeInvalidResponse, "invalid slash command list response").WithCause(err)
	}
	if data == nil {
		return nil, nil, errs.NewAPIError(errs.SubtypeInvalidResponse, "missing slash command list data")
	}
	// The endpoint promises an unpaginated list. If it starts paginating, expose
	// its continuation on reads and reject by-name writes until a complete list exists.
	items, _ := data["items"].([]interface{})
	if items == nil {
		items = []interface{}{}
	}
	for _, item := range items {
		if _, err := decodeCommandState(item); err != nil {
			return nil, nil, err
		}
	}
	return data, items, nil
}

func resolveCommand(runtime *common.RuntimeContext, name string) (map[string]interface{}, error) {
	data, items, err := commandList(runtime)
	if err != nil {
		return nil, err
	}
	if more, _ := data["has_more"].(bool); more {
		return nil, errs.NewAPIError(errs.SubtypeInvalidResponse, "cannot resolve a slash command from an incomplete list")
	}
	var found map[string]interface{}
	for _, item := range items {
		state, err := decodeCommandState(item)
		if err != nil {
			return nil, err
		}
		if state.Command != name {
			continue
		}
		if found != nil {
			return nil, errs.NewAPIError(errs.SubtypeInvalidResponse, "slash command name %q is ambiguous", name).WithHint("inspect the list and select an explicit command_id")
		}
		found = item.(map[string]interface{})
	}
	if found == nil {
		return nil, commandNotFoundError(name)
	}
	return found, nil
}

func unchangedCommand(current, replacement map[string]interface{}) bool {
	for field, value := range replacement {
		previous, exists := current[field]
		if !exists {
			return false
		}
		oldJSON, oldErr := json.Marshal(previous)
		newJSON, newErr := json.Marshal(value)
		if oldErr != nil || newErr != nil || !bytes.Equal(oldJSON, newJSON) {
			return false
		}
	}
	return true
}

func outputUnchanged(runtime *common.RuntimeContext, current map[string]interface{}) error {
	current["action"] = "unchanged"
	current["unchanged"] = true
	runtime.OutFormat(current, nil, func(w io.Writer) {
		fmt.Fprintf(w, "unchanged /%v (command_id: %v)\n", current["command"], current["command_id"])
	})
	return nil
}
