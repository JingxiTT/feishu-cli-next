// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package doc

import (
	"context"
	"encoding/json"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/shortcuts/common"
	"github.com/larksuite/cli/shortcuts/doc/internal/docworkspace"
)

var DocsPull = common.Shortcut{
	Service: "docs", Command: "+pull", Description: "Read a document as Markdown or save a local working copy",
	Risk: "read", Scopes: []string{"docx:document:readonly"},
	ConditionalScopes: []string{"wiki:node:retrieve", "tenant:tenant:readonly"},
	AuthTypes:         []string{"user", "bot"},
	Flags: []common.Flag{
		{Name: "doc", Desc: "document URL or token", Required: true},
		{Name: "output", Desc: "exclusive .md working-copy path; also saves a .feishu.json native sidecar"},
		{Name: "scope", Desc: "read-only selection; saved working copies require full", Default: "full", Enum: []string{"full", "outline", "section", "range", "keyword"}},
		{Name: "start-block-id", Desc: "range start or section heading native block ID"},
		{Name: "end-block-id", Desc: "inclusive range end native block ID"},
		{Name: "keyword", Desc: "case-insensitive literal text to find within individual blocks"},
	},
	Validate: func(_ context.Context, runtime *common.RuntimeContext) error {
		if _, err := parseDocumentRef(runtime.Str("doc")); err != nil {
			return err
		}
		selection := docsPullSelection(runtime)
		if err := selection.Validate(); err != nil {
			return err
		}
		if runtime.Str("output") != "" {
			if selection.Scope != "full" {
				return errs.NewValidationError(errs.SubtypeInvalidArgument, "saved working copies require --scope full").WithParam("--output")
			}
			_, _, err := docsWorkspaceFilePaths(runtime, runtime.Str("output"), "--output", true)
			return err
		}
		return nil
	},
	DryRun: func(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
		preview := common.NewDryRunAPI().Set("doc", runtime.Str("doc"))
		preview.Set("selection", docsPullSelection(runtime)).Set("selection_execution", "read the complete native snapshot, then select locally; no mutation")
		path := "/open-apis/docx/v1/documents/:document_id"
		if ref, err := parseDocumentRef(runtime.Str("doc")); err == nil {
			if ref.Kind == "wiki" {
				preview.GET("/open-apis/wiki/v2/spaces/node_by_token").Params(map[string]interface{}{"token": ref.Token}).Desc("Resolve the wiki node to its native document")
			} else if ref.Kind == "docx" {
				preview.Set("document_id", ref.Token)
				path = docsWorkspacePath(ref.Token)
			}
		}
		if path := runtime.Str("output"); path != "" {
			preview.Set("binding", "Resolve and verify the current account identity and tenant; identity lookups may be required").
				File(cmdutil.DryRunFileIntent{Name: path, IfExists: "fail"}).
				File(cmdutil.DryRunFileIntent{Name: path + ".feishu.json", IfExists: "fail"})
		}
		docsWorkspacePreviewSnapshot(preview, path)
		return preview
	},
	Execute: executeDocsPull,
}

func executeDocsPull(_ context.Context, runtime *common.RuntimeContext) error {
	file, sidecar := "", ""
	var binding common.ArtifactBinding
	if runtime.Str("output") != "" {
		var err error
		file, sidecar, err = docsWorkspaceFilePaths(runtime, runtime.Str("output"), "--output", true)
		if err != nil {
			return err
		}
		binding, err = runtime.ResolveArtifactBinding()
		if err != nil {
			return err
		}
	}
	documentID, err := resolveDocxDocumentID(runtime, runtime.Str("doc"))
	if err != nil {
		return err
	}
	snapshot, err := readDocsWorkspaceSnapshot(runtime, documentID)
	if err != nil {
		return err
	}
	var markdown string
	returned := len(snapshot.Blocks)
	if file == "" {
		markdown, returned, err = docworkspace.Select(snapshot.Blocks, docsPullSelection(runtime))
	} else {
		// Keep the existing v1 editable projection stable for saved workspaces.
		markdown, err = docworkspace.Render(snapshot.Blocks)
	}
	if err != nil {
		return err
	}
	if file == "" {
		runtime.Out(map[string]interface{}{"document_id": documentID, "revision_id": snapshot.RevisionID, "markdown": markdown,
			"scope": runtime.Str("scope"), "complete": returned == len(snapshot.Blocks), "total_blocks": len(snapshot.Blocks), "returned_blocks": returned}, nil)
		return nil
	}
	saved := docsWorkspace{Schema: docsWorkspaceSchema, Binding: binding, DocumentID: documentID, RevisionID: snapshot.RevisionID, Blocks: snapshot.Blocks}
	raw, err := json.Marshal(saved)
	if err != nil {
		return errs.NewInternalError(errs.SubtypeInvalidResponse, "cannot encode document workspace").WithCause(err)
	}
	if len(raw) > docsWorkspaceLimit || len(markdown) > docsWorkspaceLimit {
		return errs.NewValidationError(errs.SubtypeFailedPrecondition, "document workspace exceeds local file size limits")
	}
	if err := saveDocsWorkspaceFile(runtime, sidecar, "application/json", raw); err != nil {
		return err
	}
	if err := saveDocsWorkspaceFile(runtime, file, "text/markdown; charset=utf-8", []byte(markdown)); err != nil {
		return errs.NewInternalError(errs.SubtypeFileIO, "native sidecar was saved but Markdown publication failed; the working-copy pair is incomplete").WithCause(err).
			WithHint("Keep the saved sidecar for inspection; pull to a different unused file path.")
	}
	runtime.Out(map[string]interface{}{"document_id": documentID, "revision_id": snapshot.RevisionID, "file": file, "sidecar": sidecar, "block_count": len(snapshot.Blocks)}, nil)
	return nil
}

func docsPullSelection(runtime *common.RuntimeContext) docworkspace.Selection {
	return docworkspace.Selection{Scope: runtime.Str("scope"), StartID: runtime.Str("start-block-id"), EndID: runtime.Str("end-block-id"), Keyword: runtime.Str("keyword")}
}
