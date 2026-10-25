// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package schema

import (
	"io"
	"sort"
	"strings"

	"github.com/larksuite/cli/internal/apicatalog"
	"github.com/larksuite/cli/internal/output"
)

const (
	schemaIndexHint      = "run schema SERVICE.RESOURCE.METHOD (e.g. schema im.messages.delete) for one method; add --all for every complete schema"
	schemaIndexDescRunes = 120
)

type schemaIndexEnvelope struct {
	OK   bool            `json:"ok"`
	Data any             `json:"data"`
	Meta schemaIndexMeta `json:"meta"`
}

type schemaIndexMeta struct {
	Count int    `json:"count"`
	Hint  string `json:"hint"`
}

type schemaMethodIndex struct {
	Method      string `json:"method"`
	Description string `json:"description"`
}

type schemaServiceIndex struct {
	Service string `json:"service"`
	Methods int    `json:"methods"`
}

// writeSchemaIndex prints a compact listing for a path that matches several
// methods. The bare catalog is grouped by service because its full method
// list alone is far larger than an agent's context budget.
func writeSchemaIndex(out io.Writer, target apicatalog.Target, refs []apicatalog.MethodRef) {
	if target.Kind == apicatalog.TargetAll {
		counts := make(map[string]int)
		for _, ref := range refs {
			counts[ref.Service.Name]++
		}
		data := make([]schemaServiceIndex, 0, len(counts))
		for name, n := range counts {
			data = append(data, schemaServiceIndex{Service: name, Methods: n})
		}
		sort.Slice(data, func(i, j int) bool { return data[i].Service < data[j].Service })
		output.PrintJson(out, schemaIndexEnvelope{OK: true, Data: data, Meta: schemaIndexMeta{Count: len(data), Hint: schemaIndexHint}})
		return
	}
	data := make([]schemaMethodIndex, 0, len(refs))
	for _, ref := range refs {
		data = append(data, schemaMethodIndex{Method: ref.SchemaPath(), Description: indexDescription(ref.Method.Description)})
	}
	sort.Slice(data, func(i, j int) bool { return data[i].Method < data[j].Method })
	output.PrintJson(out, schemaIndexEnvelope{OK: true, Data: data, Meta: schemaIndexMeta{Count: len(data), Hint: schemaIndexHint}})
}

func indexDescription(description string) string {
	text := strings.Join(strings.Fields(description), " ")
	runes := []rune(text)
	if len(runes) <= schemaIndexDescRunes {
		return text
	}
	return string(runes[:schemaIndexDescRunes]) + "..."
}
