// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

//go:build linux

package keychain

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/internal/vfs"
)

type firstKeyFS struct {
	vfs.FS
	count atomic.Int32
	ready chan struct{}
}

func (f *firstKeyFS) ReadFile(path string) ([]byte, error) {
	b, e := f.FS.ReadFile(path)
	if filepath.Base(path) == "master.key" && errors.Is(e, os.ErrNotExist) {
		if f.count.Add(1) == 2 {
			close(f.ready)
		}
		<-f.ready
	}
	return b, e
}
func TestConcurrentFirstSecrets(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_DATA_DIR", t.TempDir())
	old := vfs.DefaultFS
	vfs.DefaultFS = &firstKeyFS{FS: old, ready: make(chan struct{})}
	defer func() { vfs.DefaultFS = old }()
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, account := range []string{"writer-a", "writer-b"} {
		wg.Add(1)
		go func(a string) { defer wg.Done(); errs <- platformSet("test-service", a, "synthetic secret") }(account)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	for _, a := range []string{"writer-a", "writer-b"} {
		if secret, e := platformGet("test-service", a); e != nil || secret != "synthetic secret" {
			t.Fatalf("both saves succeeded, but %s is unreadable: %v", a, e)
		}
	}
}
