// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package core

import (
	"context"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/vfs"
)

// WithConfigLock serializes configuration mutations across goroutines and processes.
// Load, validation, dependent credential changes, and SaveMultiAppConfig must all
// happen inside fn. It is not reentrant; never hold it across interactive input.
func WithConfigLock(fn func() error) (err error) {
	if err := vfs.MkdirAll(GetConfigDir(), 0700); err != nil {
		return errs.NewInternalError(errs.SubtypeStorage, "prepare config lock").WithCause(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	lock := flock.New(filepath.Join(GetConfigDir(), "config.lock"))
	locked, err := lock.TryLockContext(ctx, 25*time.Millisecond)
	if err != nil {
		return errs.NewInternalError(errs.SubtypeStorage, "acquire config lock").WithCause(err)
	}
	if !locked {
		return errs.NewInternalError(errs.SubtypeStorage, "timed out waiting for config lock").WithCause(context.DeadlineExceeded).WithRetryable()
	}
	defer func() {
		if unlockErr := lock.Unlock(); err == nil && unlockErr != nil {
			err = errs.NewInternalError(errs.SubtypeStorage, "release config lock").WithCause(unlockErr)
		}
	}()
	return fn()
}
