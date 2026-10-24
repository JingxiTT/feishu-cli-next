// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package output

import "io"

// agentEnvelope returns the envelope raw and jq output encode. They bypass
// WriteJSON, so the honest mutation outcome is attached here while their
// encoding and filtering contracts stay unchanged. Other formats get env as is.
func agentEnvelope(out io.Writer, env Envelope, opts EmitOptions) interface{} {
	if !opts.Raw && opts.JQ == "" {
		return env
	}
	mutation := agentOptions(out).mutation()
	if mutation == nil {
		return env
	}
	if mutation.unsuccessful() {
		env.OK = false
	}
	return struct {
		Envelope
		Mutation *mutationResult `json:"mutation"`
	}{env, mutation}
}
