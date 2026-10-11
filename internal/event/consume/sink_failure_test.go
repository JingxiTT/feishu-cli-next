// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package consume

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"syscall"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/event"
	"github.com/larksuite/cli/internal/event/adapter/localbus/protocol"
)

type fullDiskWriter struct{}

func (fullDiskWriter) Write([]byte) (int, error) { return 0, syscall.ENOSPC }
func TestSinkFailureReturnsError(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	go func() {
		defer b.Close()
		protocol.Encode(b, &protocol.Event{Type: protocol.MsgTypeEvent, Payload: json.RawMessage(`{"message":"must save"}`)})
	}()
	var emitted atomic.Int64
	last := false
	e := consumeLoop(context.Background(), a, bufio.NewReader(a), &event.KeyDefinition{Workers: 1}, Options{Out: fullDiskWriter{}, ErrOut: io.Discard, Quiet: true}, "audit", &last, &emitted)
	assertSinkFailure(t, e)
	if e == nil {
		t.Fatalf("disk-full output failure returned success; saved=%d", emitted.Load())
	}
}

func assertSinkFailure(t *testing.T, err error) {
	t.Helper()
	var typed *errs.InternalError
	if !errors.As(err, &typed) || typed.Subtype != errs.SubtypeFileIO || !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("expected typed sink error preserving ENOSPC, got %v", err)
	}
}
