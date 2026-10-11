// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package event

import (
	"testing"
	"time"
)

func TestDedupFilter_OldSlotPreservesReinsertedEntry(t *testing.T) {
	d := NewDedupFilterWithSize(3, time.Hour)
	d.IsDuplicate("A")
	d.IsDuplicate("B")
	firstSeenAt := d.seen["A"].seenAt
	d.seen["A"] = dedupEntry{seenAt: time.Now().Add(-2 * time.Hour), slot: 0}
	if d.IsDuplicate("A") {
		t.Fatal("expired A must be accepted")
	}
	// Model two insertions receiving the same timestamp on a coarse clock.
	fresh := d.seen["A"]
	fresh.seenAt = firstSeenAt
	d.seen["A"] = fresh
	d.IsDuplicate("C") // Evicts A's old slot, not its fresh entry.
	if !d.IsDuplicate("A") {
		t.Fatal("old ring slot removed the fresh duplicate check")
	}
}
