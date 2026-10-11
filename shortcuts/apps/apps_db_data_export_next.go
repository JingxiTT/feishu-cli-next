// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
)

// countExportRows measures serialized rows without interpreting database values.
// SQL dumps and malformed content do not establish a row count or completeness.
func countExportRows(body []byte, format string) (int, bool) {
	switch format {
	case "csv":
		reader := csv.NewReader(bytes.NewReader(body))
		reader.FieldsPerRecord = -1
		count := -1 // header
		for {
			_, err := reader.Read()
			if err == io.EOF {
				return max(count, 0), true
			}
			if err != nil {
				return 0, false
			}
			count++
		}
	case "json":
		var rows []json.RawMessage
		if err := json.Unmarshal(body, &rows); err == nil {
			return len(rows), true
		}
		return 0, false
	default:
		return 0, false
	}
}
