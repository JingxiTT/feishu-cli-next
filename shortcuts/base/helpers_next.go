// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/larksuite/cli/shortcuts/common"
)

// parseBaseJSON retains arbitrary JSON numbers without a float64 conversion.
// It also rejects trailing values so @file and inline inputs share one document contract.
func parseBaseJSON(data []byte, target any) error {
	if !json.Valid(data) {
		var raw json.RawMessage
		return json.Unmarshal(data, &raw)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return err
		}
		return baseFlagErrorf("JSON input must contain exactly one value")
	}
	return nil
}

// baseV3ReadCall labels only a known read endpoint within a mutating workflow.
// The context marker is bound to the exact method and path, never the command.
func baseV3ReadCall(runtime *common.RuntimeContext, method, path string, params map[string]interface{}, data interface{}) (map[string]interface{}, error) {
	ctx := common.WithReadOnlyRequest(runtime.Ctx(), method, path)
	return baseV3CallContext(ctx, runtime, method, path, params, data)
}
