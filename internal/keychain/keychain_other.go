// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

//go:build linux

package keychain

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/google/uuid"
	"github.com/larksuite/cli/internal/validate"
	"github.com/larksuite/cli/internal/vfs"
)

const masterKeyBytes = 32
const ivBytes = 12
const tagBytes = 16

// StorageDir returns the directory where encrypted files are stored.
func StorageDir(service string) string {
	if dir := os.Getenv("LARKSUITE_CLI_DATA_DIR"); dir != "" {
		safeDir, err := validate.SafeEnvDirPath(dir, "LARKSUITE_CLI_DATA_DIR")
		if err == nil {
			return filepath.Join(safeDir, service)
		}
	}
	home, err := vfs.UserHomeDir()
	if err != nil || home == "" {
		// If home is missing, fallback to relative path and print warning.
		// This matches the behavior in internal/core/config.go.
		fmt.Fprintf(os.Stderr, "warning: unable to determine home directory: %v\n", err)
	}
	xdgData := filepath.Join(home, ".local", "share")
	return filepath.Join(xdgData, service)
}

var safeFileNameRe = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

// safeFileName sanitizes an account name to be used as a safe file name.
func safeFileName(account string) string {
	return safeFileNameRe.ReplaceAllString(account, "_") + ".enc"
}

// getMasterKey retrieves the master key from the file system.
// If allowCreate is true, it generates and stores a new master key if one doesn't exist.
func getMasterKey(service string, allowCreate bool) ([]byte, error) {
	dir := StorageDir(service)
	keyPath := filepath.Join(dir, "master.key")

	key, err := vfs.ReadFile(keyPath)
	if err == nil && len(key) == masterKeyBytes {
		return key, nil
	}
	if err == nil && len(key) != masterKeyBytes {
		// Key file exists but is corrupted
		return nil, errors.New("keychain is corrupted")
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		// Real I/O error (permission denied, etc.) - propagate it
		return nil, err
	}

	if !allowCreate {
		return nil, errNotInitialized
	}

	if err := vfs.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}

	key = make([]byte, masterKeyBytes)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	tmpKeyPath := filepath.Join(dir, "master.key."+uuid.New().String()+".tmp")
	defer vfs.Remove(tmpKeyPath)

	if err := vfs.WriteFile(tmpKeyPath, key, 0600); err != nil {
		return nil, err
	}

	// Publish the complete key without replacing another process's winning key.
	if err := vfs.Link(tmpKeyPath, keyPath); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		existingKey, readErr := vfs.ReadFile(keyPath)
		if readErr == nil && len(existingKey) == masterKeyBytes {
			return existingKey, nil
		}
		return nil, err
	}

	return key, nil
}

// platformGet retrieves a value from the file system.
func platformGet(service, account string) (string, error) {
	path := filepath.Join(StorageDir(service), safeFileName(account))
	data, err := vfs.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	key, err := getMasterKey(service, false)
	if err != nil {
		return "", err
	}
	plaintext, err := decryptData(data, key)
	if err != nil {
		return "", err
	}
	return plaintext, nil
}

// platformSet stores a value in the file system.
func platformSet(service, account, data string) error {
	key, err := getMasterKey(service, true)
	if err != nil {
		return err
	}
	dir := StorageDir(service)
	if err := vfs.MkdirAll(dir, 0700); err != nil {
		return err
	}
	encrypted, err := encryptData(data, key)
	if err != nil {
		return err
	}

	targetPath := filepath.Join(dir, safeFileName(account))
	tmpPath := filepath.Join(dir, safeFileName(account)+"."+uuid.New().String()+".tmp")
	defer vfs.Remove(tmpPath)

	if err := vfs.WriteFile(tmpPath, encrypted, 0600); err != nil {
		return err
	}

	// Atomic rename to prevent file corruption during multi-process writes
	if err := vfs.Rename(tmpPath, targetPath); err != nil {
		return err
	}
	return nil
}

// platformRemove deletes a value from the file system.
func platformRemove(service, account string) error {
	err := vfs.Remove(filepath.Join(StorageDir(service), safeFileName(account)))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
