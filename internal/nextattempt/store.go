// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

// Package nextattempt stores evidence, never instructions to replay a mutation.
package nextattempt

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"path/filepath"
	"regexp"
	"runtime"
	"time"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/lockfile"
	"github.com/larksuite/cli/internal/vfs"
)

const (
	Applied   = "confirmed_applied"
	Unapplied = "confirmed_unapplied"
	Unknown   = "unknown"
)

// Binding contains non-secret identifiers verified by the credential owner.
// Missing tenant evidence stays explicit; never infer it from an app ID.
type Binding struct {
	Environment    string `json:"environment"`
	Account        string `json:"account,omitempty"`
	AppID          string `json:"app_id"`
	Identity       string `json:"identity"`
	UserID         string `json:"user_id,omitempty"`
	TenantKey      string `json:"tenant_key,omitempty"`
	UserVerified   bool   `json:"user_verified,omitempty"`
	TenantVerified bool   `json:"tenant_verified"`
}

type Receipt struct {
	Version           int       `json:"version"`
	ID                string    `json:"id"`
	Binding           Binding   `json:"binding"`
	Method            string    `json:"method"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	Outcome           string    `json:"outcome"`
	Evidence          string    `json:"evidence"`
	HTTPStatus        int       `json:"http_status,omitempty"`
	APICode           *int      `json:"api_code,omitempty"`
	PersistenceFailed bool      `json:"persistence_failed,omitempty"`
}

// Store is CLI-owned local state, separate from user/workspace FileIO.
type Store struct{ Dir string }

var validID = regexp.MustCompile(`^[a-f0-9]{32}$`)

func stateError(err error) error {
	return errs.NewInternalError(errs.SubtypeFileIO, "mutation receipt state is unavailable").WithCause(err).
		WithHint("Inspect local receipt storage; do not retry a mutation with an unknown outcome.")
}
func invalidState() error {
	return errs.NewConfigError(errs.SubtypeInvalidConfig, "incompatible, unsafe, or corrupt mutation receipt state")
}
func (s Store) validateDir(create bool) error {
	if !filepath.IsAbs(s.Dir) || filepath.Clean(s.Dir) != s.Dir {
		return invalidState()
	}
	// Config roots may intentionally be symlinks (including macOS /var). Refuse
	// redirected state-owned components before MkdirAll can traverse them.
	for _, path := range []string{filepath.Dir(s.Dir), s.Dir} {
		info, err := vfs.Lstat(path)
		if errors.Is(err, fs.ErrNotExist) && create {
			continue
		}
		if err != nil {
			return stateError(err)
		}
		if !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 || runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0 {
			return invalidState()
		}
	}
	if create {
		if err := vfs.MkdirAll(s.Dir, 0700); err != nil {
			return stateError(err)
		}
	}
	info, err := vfs.Lstat(s.Dir)
	if err != nil {
		return stateError(err)
	}
	if !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 || runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0 {
		return invalidState()
	}
	return nil
}

func (s Store) path(id string) (string, error) {
	if !validID.MatchString(id) {
		return "", errs.NewValidationError(errs.SubtypeInvalidArgument, "invalid receipt ID").WithParam("id")
	}
	return filepath.Join(s.Dir, id+".json"), nil
}
func (s Store) lock(id string) (*lockfile.LockFile, error) {
	path, err := s.path(id)
	if err != nil {
		return nil, err
	}
	lockPath := path + ".lock"
	if info, err := vfs.Lstat(lockPath); err == nil {
		if !info.Mode().IsRegular() || runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0 {
			return nil, invalidState()
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, stateError(err)
	}
	lock := lockfile.New(lockPath)
	if err := lock.TryLock(); err != nil {
		return nil, stateError(err)
	}
	return lock, nil
}

func (s Store) begin(binding Binding, method string) (Receipt, *lockfile.LockFile, error) {
	if err := s.validateDir(true); err != nil {
		return Receipt{}, nil, err
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return Receipt{}, nil, stateError(err)
	}
	now := time.Now().UTC()
	r := Receipt{Version: 1, ID: hex.EncodeToString(random[:]), Binding: binding, Method: method,
		CreatedAt: now, UpdatedAt: now, Outcome: Unknown, Evidence: "dispatch_started"}
	lock, err := s.lock(r.ID)
	if err != nil {
		return Receipt{}, nil, err
	}
	if err := s.save(r, true); err != nil {
		lock.Unlock()
		return Receipt{}, nil, err
	}
	return r, lock, nil
}

// finish is called while the attempt lock remains held across dispatch.
func (s Store) finish(r Receipt) error {
	if err := s.validateDir(false); err != nil {
		return err
	}
	previous, err := s.read(r.ID)
	if err != nil {
		return err
	}
	if previous.Binding != r.Binding || previous.Method != r.Method || previous.Outcome != Unknown {
		return invalidState()
	}
	r.UpdatedAt = time.Now().UTC()
	return s.save(r, false)
}

func (s Store) save(r Receipt, create bool) error {
	path, err := s.path(r.ID)
	if err != nil {
		return err
	}
	body, err := json.Marshal(r)
	if err != nil {
		return stateError(err)
	}
	temp, err := vfs.CreateTemp(s.Dir, ".receipt-*")
	if err != nil {
		return stateError(err)
	}
	defer vfs.Remove(temp.Name())
	if _, err = temp.Write(body); err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err != nil {
		return stateError(err)
	}
	if closeErr != nil {
		return stateError(closeErr)
	}
	if create {
		err = vfs.Link(temp.Name(), path)
	} else {
		err = vfs.Rename(temp.Name(), path)
	}
	if err != nil {
		return stateError(err)
	}
	return s.syncDir()
}
func (s Store) syncDir() error {
	// Windows does not support directory fsync. File contents are still synced
	// before atomic replacement; sudden power-loss metadata durability is limited.
	if runtime.GOOS == "windows" {
		return nil
	}
	dir, err := vfs.Open(s.Dir)
	if err != nil {
		return stateError(err)
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return stateError(err)
	}
	return nil
}
func (s Store) read(id string) (Receipt, error) {
	path, err := s.path(id)
	if err != nil {
		return Receipt{}, err
	}
	info, err := vfs.Lstat(path)
	if err != nil {
		return Receipt{}, stateError(err)
	}
	if !info.Mode().IsRegular() || info.Size() > 16384 || runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0 {
		return Receipt{}, invalidState()
	}
	body, err := vfs.ReadFile(path)
	if err != nil {
		return Receipt{}, stateError(err)
	}
	var r Receipt
	if err := json.Unmarshal(body, &r); err != nil {
		return Receipt{}, invalidState()
	}
	if r.Version != 1 || r.ID != id || r.Method == "" || r.CreatedAt.IsZero() || r.Evidence == "" ||
		r.Binding.Environment == "" || r.Binding.AppID == "" || (r.Binding.Identity != "user" && r.Binding.Identity != "bot") ||
		(r.Binding.Identity == "user" && r.Binding.UserID == "") || (r.Binding.TenantVerified && r.Binding.TenantKey == "") ||
		(r.Outcome != Applied && r.Outcome != Unapplied && r.Outcome != Unknown) {
		return Receipt{}, invalidState()
	}
	return r, nil
}
func (s Store) Inspect(id string) (Receipt, error) {
	if err := s.validateDir(false); err != nil {
		return Receipt{}, err
	}
	return s.read(id)
}
func (s Store) List() ([]Receipt, error) {
	result := []Receipt{}
	if _, err := vfs.Lstat(s.Dir); errors.Is(err, fs.ErrNotExist) {
		return result, nil
	}
	if err := s.validateDir(false); err != nil {
		return nil, err
	}
	entries, err := vfs.ReadDir(s.Dir)
	if err != nil {
		return nil, stateError(err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := entry.Name()[:len(entry.Name())-5]
		r, err := s.read(id)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, nil
}

// Cleanup removes only the explicitly selected receipt. Lock files are retained
// to avoid inode-reuse races with concurrent readers/writers.
func (s Store) Cleanup(id string) error {
	if err := s.validateDir(false); err != nil {
		return err
	}
	lock, err := s.lock(id)
	if err != nil {
		return err
	}
	defer lock.Unlock()
	if _, err := s.read(id); err != nil {
		return err
	}
	path, err := s.path(id)
	if err != nil {
		return err
	}
	if err := vfs.Remove(path); err != nil {
		return stateError(err)
	}
	return s.syncDir()
}
