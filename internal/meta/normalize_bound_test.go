// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package meta

import (
	"encoding/json"
	"testing"
)

func TestFieldBoundsPreserveJSONNumberGrammar(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  string
	}{
		{input: "0", want: "0"},
		{input: "-0", want: "-0"},
		{input: "1.2300", want: "1.2300"},
		{input: "-1E-10000", want: "-1E-10000"},
		{input: "9223372036854775808", want: "9223372036854775808"},
		{input: " \t\r\n1.25e+10\n ", want: "1.25e+10"},
		{input: ""},
		{input: " \t\r\n"},
		{input: "null"},
		{input: "true"},
		{input: `"123"`},
		{input: "[123]"},
		{input: `{"value":123}`},
		{input: "01"},
		{input: "+1"},
		{input: "-"},
		{input: "1."},
		{input: "1e"},
		{input: "NaN"},
		{input: "Infinity"},
		{input: "1 2"},
		{input: "1x"},
		{input: "\u00a01"},
		{input: "1\u00a0"},
		{input: "\x001"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			field := Field{Min: tt.input, Max: tt.input}
			for _, got := range []*json.Number{field.MinBound(), field.MaxBound()} {
				if tt.want == "" {
					if got != nil {
						t.Fatalf("bound(%q) = %q, want nil", tt.input, got.String())
					}
				} else if got == nil || got.String() != tt.want {
					t.Fatalf("bound(%q) = %v, want %q", tt.input, got, tt.want)
				}
			}
		})
	}
}

func TestFieldBoundsAvoidDecoderAllocations(t *testing.T) {
	field := Field{Max: "9223372036854775807"}
	var got *json.Number
	allocs := testing.AllocsPerRun(100, func() { got = field.MaxBound() })
	if got == nil || got.String() != field.Max {
		t.Fatalf("MaxBound() = %v, want %q", got, field.Max)
	}
	if allocs > 1 {
		t.Fatalf("MaxBound() allocated %g times, want only the returned number", allocs)
	}
}

func BenchmarkFieldBounds(b *testing.B) {
	for _, input := range []string{"100", "9223372036854775807", "1.25e+10"} {
		b.Run(input, func(b *testing.B) {
			field := Field{Max: input}
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if got := field.MaxBound(); got == nil || got.String() != input {
					b.Fatal(got)
				}
			}
		})
	}
}
