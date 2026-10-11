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
	seen map[string]time.Time
	ring []dedupEntry
	pos  int
	ttl  time.Duration
	mu   sync.Mutex
}

type dedupEntry struct {
	id     string
	seenAt time.Time
}

func NewDedupFilter() *DedupFilter {
	return NewDedupFilterWithSize(defaultRingSize, defaultDedupTTL)
}

func NewDedupFilterWithSize(ringSize int, ttl time.Duration) *DedupFilter {
	return &DedupFilter{
		seen: make(map[string]time.Time),
		ring: make([]dedupEntry, ringSize),
		ttl:  ttl,
	}
}

func (d *DedupFilter) IsDuplicate(eventID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	if ts, ok := d.seen[eventID]; ok {
		if now.Sub(ts) < d.ttl {
			return true
		}
		delete(d.seen, eventID)
	}

	d.seen[eventID] = now

	if old := d.ring[d.pos]; old.id != "" && d.seen[old.id] == old.seenAt {
		delete(d.seen, old.id)
	}
	d.ring[d.pos] = dedupEntry{id: eventID, seenAt: now}
	d.pos = (d.pos + 1) % len(d.ring)

	if d.pos%1000 == 0 {
		d.cleanupExpired(now)
	}

	return false
}

func (d *DedupFilter) cleanupExpired(now time.Time) {
	for id, ts := range d.seen {
		if now.Sub(ts) >= d.ttl {
			delete(d.seen, id)
		}
	}
}
