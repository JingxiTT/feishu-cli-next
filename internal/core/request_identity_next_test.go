// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package core

import (
	"context"
	"testing"
)

func TestRequestIdentityContextShadowsWithoutMutatingParent(t *testing.T) {
	parent, cancel := context.WithCancel(WithRequestIdentity(context.Background(), AsUser))
	child := WithRequestIdentity(parent, AsBot)
	if got, ok := RequestIdentityFromContext(child); !ok || got != AsBot {
		t.Fatal(got, ok)
	}
	if got, ok := RequestIdentityFromContext(parent); !ok || got != AsUser {
		t.Fatal("parent identity changed")
	}
	if _, ok := RequestIdentityFromContext(WithRequestIdentity(parent, AsAuto)); ok {
		t.Fatal("unresolved identity accepted")
	}
	if _, ok := RequestIdentityFromContext(context.Background()); ok {
		t.Fatal("missing identity invented")
	}
	cancel()
	if child.Err() != context.Canceled {
		t.Fatal("request identity lost cancellation")
	}
}
