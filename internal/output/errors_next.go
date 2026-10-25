// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package output

var typedErrorIndent = "  "

// SetCompactErrors writes typed error envelopes as single-line JSON when compact is true.
func SetCompactErrors(compact bool) {
	typedErrorIndent = "  "
	if compact {
		typedErrorIndent = ""
	}
}
