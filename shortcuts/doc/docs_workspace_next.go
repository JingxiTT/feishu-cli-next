// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package doc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/extension/fileio"
	"github.com/larksuite/cli/shortcuts/common"
	"github.com/larksuite/cli/shortcuts/doc/internal/docworkspace"
)

const docsWorkspaceSchema = "feishu-cli-next/docs-workspace/v1"
const docsWorkspaceLimit = 16 << 20
const docsWorkspaceBlockLimit = 10000

type docsWorkspace struct {
	Schema     string                 `json:"schema"`
	Binding    common.ArtifactBinding `json:"binding"`
	DocumentID string                 `json:"document_id"`
	RevisionID int64                  `json:"revision_id"`
	Blocks     []json.RawMessage      `json:"blocks"`
}

type docsWorkspaceSnapshot struct {
	DocumentID string
	RevisionID int64
	Blocks     []json.RawMessage
}

func docsWorkspaceError(message string) *errs.ValidationError {
	return errs.NewValidationError(errs.SubtypeFailedPrecondition, message).
		WithHint("Keep your local edits and pull a fresh working copy to another file before reconciling them.")
}

func docsWorkspaceIDValid(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

func docsWorkspacePath(documentID string) string {
	return "/open-apis/docx/v1/documents/" + url.PathEscape(documentID)
}

// Decode the typed projection after the shared API boundary has retained JSON
// numbers and classified response failures. Unknown native fields stay in raw blocks.
func docsWorkspaceAPI(runtime *common.RuntimeContext, method, path string, params map[string]interface{}, body, into interface{}) error {
	data, err := runtime.CallAPITyped(method, path, params, body)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(data)
	if err != nil || len(raw) > docsWorkspaceLimit || data == nil {
		return errs.NewInternalError(errs.SubtypeInvalidResponse, "invalid or oversized native document response").WithCause(err)
	}
	if err := json.Unmarshal(raw, into); err != nil {
		return errs.NewInternalError(errs.SubtypeInvalidResponse, "invalid native document response").WithCause(err)
	}
	return nil
}

func docsWorkspaceRevision(runtime *common.RuntimeContext, documentID string) (int64, error) {
	var reply struct {
		Document struct {
			ID       string `json:"document_id"`
			Revision *int64 `json:"revision_id"`
		} `json:"document"`
	}
	if err := docsWorkspaceAPI(runtime, "GET", docsWorkspacePath(documentID), nil, nil, &reply); err != nil {
		return 0, err
	}
	if reply.Document.ID != documentID || reply.Document.Revision == nil || *reply.Document.Revision < 0 {
		return 0, errs.NewInternalError(errs.SubtypeInvalidResponse, "native document identity or revision is missing")
	}
	return *reply.Document.Revision, nil
}

func readDocsWorkspaceSnapshot(runtime *common.RuntimeContext, documentID string) (docsWorkspaceSnapshot, error) {
	// Only a completed read whose revision moved is reconciled. Transport,
	// pagination and validation failures return immediately; no mutation repeats.
	for attempt := 0; ; attempt++ {
		snapshot, latest, err := readDocsWorkspaceSnapshotAttempt(runtime, documentID)
		if err != nil {
			return snapshot, err
		}
		if latest == snapshot.RevisionID {
			return snapshot, nil
		}
		if attempt == 2 {
			return snapshot, docsWorkspaceError("document changed while reading its native blocks after 3 snapshot attempts").
				WithHint(fmt.Sprintf("Final read observed revision %d before reading and %d after reading. Keep local edits and pull a fresh working copy before reconciling; no mutation was retried.", snapshot.RevisionID, latest))
		}
	}
}

func readDocsWorkspaceSnapshotAttempt(runtime *common.RuntimeContext, documentID string) (docsWorkspaceSnapshot, int64, error) {
	snapshot := docsWorkspaceSnapshot{DocumentID: documentID}
	if !docsWorkspaceIDValid(documentID) {
		return snapshot, 0, docsWorkspaceError("invalid document identity")
	}
	revision, err := docsWorkspaceRevision(runtime, documentID)
	if err != nil {
		return snapshot, 0, err
	}
	snapshot.RevisionID = revision
	seenPages, seenBlocks := map[string]bool{}, map[string]bool{}
	token, size := "", 0
	for {
		params := map[string]interface{}{"document_revision_id": strconv.FormatInt(revision, 10), "page_size": 500}
		if token != "" {
			params["page_token"] = token
		}
		var page struct {
			Items     []json.RawMessage `json:"items"`
			HasMore   *bool             `json:"has_more"`
			PageToken string            `json:"page_token"`
		}
		if err := docsWorkspaceAPI(runtime, "GET", docsWorkspacePath(documentID)+"/blocks", params, nil, &page); err != nil {
			return snapshot, 0, err
		}
		if page.HasMore == nil {
			return snapshot, 0, errs.NewInternalError(errs.SubtypeInvalidResponse, "native block pagination completeness is missing")
		}
		for _, raw := range page.Items {
			var block struct {
				ID string `json:"block_id"`
			}
			if json.Unmarshal(raw, &block) != nil || !docsWorkspaceIDValid(block.ID) || seenBlocks[block.ID] {
				return snapshot, 0, errs.NewInternalError(errs.SubtypeInvalidResponse, "native blocks contain invalid or duplicate identities")
			}
			seenBlocks[block.ID] = true
			size += len(raw)
			snapshot.Blocks = append(snapshot.Blocks, raw)
			if len(snapshot.Blocks) > docsWorkspaceBlockLimit || size > docsWorkspaceLimit {
				return snapshot, 0, errs.NewValidationError(errs.SubtypeFailedPrecondition, "document exceeds local workspace limits")
			}
		}
		if !*page.HasMore {
			break
		}
		if len(page.Items) == 0 || page.PageToken == "" || seenPages[page.PageToken] {
			return snapshot, 0, errs.NewInternalError(errs.SubtypeInvalidResponse, "native block pagination did not advance")
		}
		token = page.PageToken
		seenPages[token] = true
	}
	if len(snapshot.Blocks) == 0 {
		return snapshot, 0, errs.NewInternalError(errs.SubtypeInvalidResponse, "native document contains no blocks")
	}
	latest, err := docsWorkspaceRevision(runtime, documentID)
	if err != nil {
		return snapshot, 0, err
	}
	return snapshot, latest, nil
}

func docsWorkspaceJSONEqual(left, right interface{}) bool {
	decode := func(value interface{}) (interface{}, error) {
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		var result interface{}
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		err = decoder.Decode(&result)
		return result, err
	}
	a, errA := decode(left)
	b, errB := decode(right)
	return errA == nil && errB == nil && reflect.DeepEqual(a, b)
}

const docsWorkspaceAtPathHint = "--file names the saved workspace path: use --file ./note.md without @. For a literal @-prefixed filename, use --file ./@note.md."

const docsWorkspaceMissingHint = "the .md working copy and its .feishu.json sidecar come from `docs +pull --doc <doc> --output <file>`; run it to create both"

func docsWorkspaceFilePaths(runtime *common.RuntimeContext, name, param string, output bool) (string, string, error) {
	if name == "" || !strings.EqualFold(filepath.Ext(name), ".md") {
		return "", "", errs.NewValidationError(errs.SubtypeInvalidArgument, "working copy must use a .md file").WithParam(param)
	}
	if runtime.FileIO() == nil {
		return "", "", errs.NewValidationError(errs.SubtypeFailedPrecondition, "local document workflow requires FileIO")
	}
	sidecar := name + ".feishu.json"
	for _, path := range []string{name, sidecar} {
		if output {
			if _, err := runtime.ResolveSavePath(path); err != nil {
				return "", "", common.WrapSaveErrorTypedForFlag(err, param)
			}
			if _, err := runtime.FileIO().Stat(path); err == nil {
				return "", "", errs.NewValidationError(errs.SubtypeFailedPrecondition, "working copy or sidecar already exists").WithParam(param)
			} else if !errors.Is(err, fs.ErrNotExist) {
				return "", "", common.WrapInputStatErrorTyped(err)
			}
		} else if err := runtime.ValidatePath(path); err != nil {
			return "", "", common.WrapInputStatErrorTyped(err)
		}
	}
	if output {
		if _, ok := runtime.FileIO().(fileio.ExclusiveFileIO); !ok {
			return "", "", errs.NewValidationError(errs.SubtypeFailedPrecondition, "working copy requires an exclusive FileIO provider").WithParam(param)
		}
	}
	return name, sidecar, nil
}

func readDocsWorkspaceFile(runtime *common.RuntimeContext, path string) ([]byte, error) {
	file, err := runtime.FileIO().Open(path)
	if err != nil {
		return nil, common.WrapInputStatErrorTyped(err)
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, docsWorkspaceLimit+1))
	if err != nil {
		return nil, errs.NewInternalError(errs.SubtypeFileIO, "cannot read document working copy").WithCause(err)
	}
	if len(raw) > docsWorkspaceLimit {
		return nil, docsWorkspaceError("working copy exceeds the size limit")
	}
	return raw, nil
}

func loadDocsWorkspace(runtime *common.RuntimeContext) (docsWorkspace, []docworkspace.Edit, error) {
	var saved docsWorkspace
	file, sidecar, err := docsWorkspaceFilePaths(runtime, runtime.Str("file"), "--file", false)
	if err != nil {
		return saved, nil, err
	}
	// --file names a path, unlike content flags that resolve @file bytes.
	// Keep existing literal @ filenames valid; clarify only missing input.
	hint := docsWorkspaceMissingHint
	if strings.HasPrefix(file, "@") {
		hint = docsWorkspaceAtPathHint
	}
	if _, err := runtime.FileIO().Stat(file); errors.Is(err, fs.ErrNotExist) {
		return saved, nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "working copy markdown not found: %s", file).WithParam("--file").WithCause(err).WithHint(hint)
	}
	raw, err := readDocsWorkspaceFile(runtime, sidecar)
	if err != nil {
		var input *errs.ValidationError
		if errors.Is(err, fs.ErrNotExist) && errors.As(err, &input) {
			input.WithHint(hint)
		}
		return saved, nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&saved); err != nil {
		return saved, nil, docsWorkspaceError("invalid document workspace sidecar").WithParam("--file").WithCause(err)
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	var revision *int64
	_ = json.Unmarshal(fields["revision_id"], &revision)
	var trailing interface{}
	if revision == nil || decoder.Decode(&trailing) != io.EOF || saved.Schema != docsWorkspaceSchema || !docsWorkspaceIDValid(saved.DocumentID) || saved.RevisionID < 0 || len(saved.Blocks) == 0 || len(saved.Blocks) > docsWorkspaceBlockLimit {
		return saved, nil, docsWorkspaceError("invalid or unsupported document workspace sidecar").WithParam("--file")
	}
	if err := common.ValidateArtifactBinding(saved.Binding, saved.Binding, "--file", true); err != nil {
		return saved, nil, err
	}
	edited, err := readDocsWorkspaceFile(runtime, file)
	if err != nil {
		return saved, nil, err
	}
	edits, err := docworkspace.Plan(saved.Blocks, string(edited))
	if err != nil {
		return saved, nil, docsWorkspaceError("local document edit is unsupported or ambiguous").WithParam("--file").WithCause(err)
	}
	for _, edit := range edits {
		if _, err := docsWorkspaceEditedBlocks(saved.Blocks, edit); err != nil {
			return saved, nil, err
		}
	}
	return saved, edits, nil
}

func saveDocsWorkspaceFile(runtime *common.RuntimeContext, path, contentType string, data []byte) error {
	exclusive, ok := runtime.FileIO().(fileio.ExclusiveFileIO)
	if !ok {
		return errs.NewValidationError(errs.SubtypeFailedPrecondition, "exclusive FileIO is required")
	}
	result, err := exclusive.SaveExclusive(path, fileio.SaveOptions{ContentType: contentType, ContentLength: int64(len(data))}, bytes.NewReader(data))
	if err != nil {
		return common.WrapSaveErrorTypedForFlag(err, "--output")
	}
	if result == nil || result.Size() != int64(len(data)) {
		return errs.NewInternalError(errs.SubtypeFileIO, "working copy file save did not confirm the complete content")
	}
	return nil
}

func docsWorkspacePreviewSnapshot(preview *common.DryRunAPI, path string) {
	preview.GET(path).Desc("Read the latest native document revision").
		GET(path + "/blocks").Params(map[string]interface{}{"document_revision_id": "<observed current revision>", "page_size": 500}).
		Desc("Read every native block page pinned to the observed current revision").
		GET(path).Desc("Reread latest revision; repeat the complete read at most 3 times only when its revision changed")
}
