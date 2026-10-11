// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package minutes

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/vfs/localfileio"
)

type concurrentMediaReader struct {
	io.Reader
	created bool
}

func (r *concurrentMediaReader) Read(p []byte) (int, error) {
	if !r.created {
		r.created = true
		if err := os.WriteFile("out.mp4", []byte("concurrent media"), 0600); err != nil {
			return 0, err
		}
	}
	return r.Reader.Read(p)
}

func TestDownloadPreservesFileCreatedDuringTransfer(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	chdir(t, t.TempDir())
	client := &http.Client{Transport: minutesRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"video/mp4"}},
			Body: io.NopCloser(&concurrentMediaReader{Reader: strings.NewReader("downloaded media")})}, nil
	})}
	_, err := downloadMediaFile(context.Background(), client, "https://192.0.2.1/media.mp4", "tok001",
		downloadOpts{fio: &localfileio.LocalFileIO{}, outputPath: "out.mp4"})
	problem, ok := errs.ProblemOf(err)
	if !ok || problem.Category != errs.CategoryValidation || problem.Subtype != errs.SubtypeFailedPrecondition || !errors.Is(err, fs.ErrExist) {
		t.Fatalf("wrong collision error: %#v, %v", problem, err)
	}
	data, err := os.ReadFile("out.mp4")
	if err != nil || string(data) != "concurrent media" {
		t.Fatalf("concurrent media changed: %q, %v", data, err)
	}
}
