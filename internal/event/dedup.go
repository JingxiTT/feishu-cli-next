// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package event

import (
	"sync"
	"time"
)

const (
	defaultDedupTTL = 5 * time.Minute
	defaultRingSize = 10000
)

// DedupFilter: seen map is sole authority; ring only bounds map size via overflow eviction.
type DedupFilter struct {
	seen map[string]dedupEntry
	ring []string
	pos  int
	ttl  time.Duration
	mu   sync.Mutex
}

type dedupEntry struct {
	seenAt time.Time
	slot   int
}

func NewDedupFilter() *DedupFilter {
	return NewDedupFilterWithSize(defaultRingSize, defaultDedupTTL)
}

func NewDedupFilterWithSize(ringSize int, ttl time.Duration) *DedupFilter {
	return &DedupFilter{
		seen: make(map[string]dedupEntry),
		ring: make([]string, ringSize),
		ttl:  ttl,
	}
}

func (d *DedupFilter) IsDuplicate(eventID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	if entry, ok := d.seen[eventID]; ok {
		if now.Sub(entry.seenAt) < d.ttl {
			return true
		}
		delete(d.seen, eventID)
	}

	// Slot ownership distinguishes reinserts even when the clock has not advanced.
	if old := d.ring[d.pos]; old != "" && d.seen[old].slot == d.pos {
		delete(d.seen, old)
	}
	d.seen[eventID] = dedupEntry{seenAt: now, slot: d.pos}
	d.ring[d.pos] = eventID
	d.pos = (d.pos + 1) % len(d.ring)

	if d.pos%1000 == 0 {
		d.cleanupExpired(now)
	}

	return false
}

func (d *DedupFilter) cleanupExpired(now time.Time) {
	for id, entry := range d.seen {
		if now.Sub(entry.seenAt) >= d.ttl {
			delete(d.seen, id)
		}
	}
}
