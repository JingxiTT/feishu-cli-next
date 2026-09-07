// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextfoundation

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLocalDiscoveryDoesNotResolveUserOrTenantIdentity(t *testing.T) {
	tests := []struct {
		name string
		args []string
		exit int
	}{
		{"guide", []string{"guide", "show"}, 0},
		{"discover", []string{"discover", "contact union_id"}, 0},
		{"root help", []string{"--help"}, 0},
		{"domain help", []string{"contact", "--help"}, 0},
		{"auth help", []string{"auth", "login", "--help"}, 0},
		{"schema", []string{"schema", "im", "images", "create"}, 0},
		{"invalid discovery flag", []string{"discover", "--limit", "0"}, 2},
		{"invalid domain flag", []string{"contact", "+get-user", "--not-a-flag"}, 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := NewFixture(t, nil)
			result := fixture.Run(test.args...)
			if result.ExitCode != test.exit {
				t.Fatalf("unexpected result: %+v", result)
			}
			if test.exit == 0 {
				if result.Stdout == "" || result.Stderr != "" {
					t.Fatalf("local output: %+v", result)
				}
			} else {
				var envelope struct {
					Error struct {
						Type string `json:"type"`
					} `json:"error"`
				}
				if json.Unmarshal([]byte(result.Stderr), &envelope) != nil || envelope.Error.Type != "validation" || strings.TrimSpace(result.Stdout) != "" {
					t.Fatalf("local validation: %+v", result)
				}
			}
			if fixture.IdentityRequests.Load() != 0 || fixture.TenantIdentityRequests.Load() != 0 {
				t.Fatalf("local command resolved identities: user=%d tenant=%d", fixture.IdentityRequests.Load(), fixture.TenantIdentityRequests.Load())
			}
		})
	}
}
