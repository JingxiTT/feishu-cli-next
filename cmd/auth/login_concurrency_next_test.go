// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package auth

import (
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/vfs"
)

type loginConfigFS struct {
	vfs.FS
	path                           string
	reads                          atomic.Int32
	firstRead, secondRead, release chan struct{}
}

func (f *loginConfigFS) ReadFile(path string) ([]byte, error) {
	data, err := f.FS.ReadFile(path)
	if path == f.path && err == nil {
		switch f.reads.Add(1) {
		case 1:
			close(f.firstRead)
			<-f.release
		case 2:
			close(f.secondRead)
		}
	}
	return data, err
}

func TestConcurrentLoginsPreserveBothProfiles(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	initial := &core.MultiAppConfig{CurrentApp: "a", Apps: []core.AppConfig{{Name: "a", AppId: "app-a"}, {Name: "b", AppId: "app-b"}}}
	if err := core.SaveMultiAppConfig(initial); err != nil {
		t.Fatal(err)
	}
	old := vfs.DefaultFS
	fs := &loginConfigFS{FS: old, path: core.GetConfigPath(), firstRead: make(chan struct{}), secondRead: make(chan struct{}), release: make(chan struct{})}
	vfs.DefaultFS = fs
	defer func() { vfs.DefaultFS = old }()
	done := make(chan error, 2)
	go func() { done <- syncLoginUserToProfile("a", "app-a", "ou_a", "Alice", io.Discard) }()
	<-fs.firstRead
	go func() { done <- syncLoginUserToProfile("b", "app-b", "ou_b", "Bob", io.Discard) }()
	// Without the transaction lock both writers read the same stale snapshot.
	select {
	case <-fs.secondRead:
	case <-time.After(100 * time.Millisecond):
	}
	close(fs.release)
	for i := 0; i < 2; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	saved, err := core.LoadMultiAppConfig()
	if err != nil {
		t.Fatal(err)
	}
	if saved.CurrentApp != "a" {
		t.Fatalf("changed active profile: %s", saved.CurrentApp)
	}
	for _, app := range saved.Apps {
		if len(app.Users) != 1 || app.Users[0].UserOpenId != "ou_"+app.Name {
			t.Fatalf("lost login for %s: %+v", app.Name, app.Users)
		}
	}
}
