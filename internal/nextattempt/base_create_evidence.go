// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextattempt

import "net/url"

const (
	baseCreateEvidenceMaxIDs   = 10000
	baseCreateEvidenceMaxBytes = 4 << 20
)

type baseCreateScope struct {
	Binding      Binding
	Origin, Path string
}

// observeBaseCreates checks this invocation's already confirmed native Base
// creates. IDs and paths stay in memory only. The importer has a 10,000-row
// maximum; larger or oversized evidence remains unknown rather than evicting
// earlier observations and claiming complete application after a possible reuse.
func (j *Journal) observeBaseCreates(binding Binding, target *url.URL, ids []string) bool {
	// Verification flags describe the evidence quality, not a different actor.
	binding.UserVerified, binding.TenantVerified = false, false
	scope := baseCreateScope{Binding: binding, Origin: target.Scheme + "://" + target.Host, Path: target.Path}
	j.mu.Lock()
	defer j.mu.Unlock()
	seen := j.baseCreated[scope]
	size := 0
	if seen == nil {
		size = len(scope.Origin) + len(scope.Path) + len(binding.Environment) + len(binding.Account) + len(binding.AppID) + len(binding.Identity) + len(binding.UserID) + len(binding.TenantKey)
	}
	for _, id := range ids {
		if _, exists := seen[id]; exists {
			return false
		}
		size += len(id)
	}
	if len(ids) > baseCreateEvidenceMaxIDs-j.baseCreatedIDs || size > baseCreateEvidenceMaxBytes-j.baseCreatedSize {
		return false
	}
	if j.baseCreated == nil {
		j.baseCreated = make(map[baseCreateScope]map[string]struct{})
	}
	if seen == nil {
		seen = make(map[string]struct{}, len(ids))
		j.baseCreated[scope] = seen
	}
	for _, id := range ids {
		seen[id] = struct{}{}
	}
	j.baseCreatedIDs += len(ids)
	j.baseCreatedSize += size
	return true
}
