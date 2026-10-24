// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package doc

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/shortcuts/common"
	"github.com/larksuite/cli/shortcuts/doc/internal/docworkspace"
)

type docsPushPlanKey struct{}
type docsPushPlan struct {
	Saved docsWorkspace
	Edits []docworkspace.Edit
}
type docsPushChangedBlock struct {
	BlockID string `json:"block_id"`
	Text    string `json:"text"`
}
type docsPushResult struct {
	DocumentID    string                 `json:"document_id"`
	RevisionID    int64                  `json:"revision_id"`
	Verified      bool                   `json:"verified"`
	Unchanged     bool                   `json:"unchanged"`
	Outcome       string                 `json:"outcome"`
	ChangedBlocks []docsPushChangedBlock `json:"changed_blocks"`
	AttemptedID   string                 `json:"attempted_block_id,omitempty"`
	Hint          string                 `json:"hint"`
}

var DocsPush = common.Shortcut{
	Service: "docs", Command: "+push", Description: "Apply supported local Markdown edits to their original native blocks",
	Risk: "write", Scopes: []string{"docx:document:readonly", "docx:document:write_only"},
	ConditionalScopes: []string{"tenant:tenant:readonly"}, AuthTypes: []string{"user", "bot"},
	Flags: []common.Flag{{Name: "file", Desc: "edited .md working copy with its original .feishu.json sidecar", Required: true}},
	Validate: func(_ context.Context, runtime *common.RuntimeContext) error {
		saved, edits, err := loadDocsWorkspace(runtime)
		if err != nil {
			return err
		}
		// Capture the validated local input once per invocation. The immutable
		// plan is shared by dry-run/execute, never by another command invocation.
		runtime.Cmd.SetContext(context.WithValue(runtime.Cmd.Context(), docsPushPlanKey{}, docsPushPlan{saved, edits}))
		return nil
	},
	DryRun: func(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
		plan, _ := runtime.Cmd.Context().Value(docsPushPlanKey{}).(docsPushPlan)
		preview := common.NewDryRunAPI().
			Set("binding", "Resolve and verify current account identity and tenant against the sidecar; identity lookups may be required").
			Set("unchanged", len(plan.Edits) == 0).
			Set("concurrency", "fresh comparisons detect observed conflicts; native revision parameters do not provide compare-and-swap")
		path := docsWorkspacePath(plan.Saved.DocumentID)
		docsWorkspacePreviewSnapshot(preview, path)
		for i, edit := range plan.Edits {
			if i > 0 {
				docsWorkspacePreviewSnapshot(preview, path)
			}
			preview.PATCH(path + "/blocks/" + url.PathEscape(edit.BlockID)).
				Params(map[string]interface{}{"document_revision_id": "<verified current revision>"}).
				Body(map[string]interface{}{"update_text_elements": map[string]interface{}{"elements": edit.Elements}}).
				Desc("Apply one block edit after comparing the current full snapshot against expected state")
			docsWorkspacePreviewSnapshot(preview, path)
		}
		return preview
	},
	Execute: executeDocsPush,
}

func executeDocsPush(_ context.Context, runtime *common.RuntimeContext) error {
	plan, ok := runtime.Cmd.Context().Value(docsPushPlanKey{}).(docsPushPlan)
	if !ok {
		return errs.NewInternalError(errs.SubtypeUnknown, "document push has no validated local plan")
	}
	currentBinding, err := runtime.ResolveArtifactBinding()
	if err != nil {
		return err
	}
	if err := common.ValidateArtifactBinding(plan.Saved.Binding, currentBinding, "--file", true); err != nil {
		return err
	}
	expected := docsWorkspaceSnapshot{DocumentID: plan.Saved.DocumentID, RevisionID: plan.Saved.RevisionID, Blocks: plan.Saved.Blocks}
	result := docsPushResult{DocumentID: expected.DocumentID, RevisionID: expected.RevisionID,
		ChangedBlocks: []docsPushChangedBlock{}, Outcome: "unchanged", Unchanged: len(plan.Edits) == 0,
		Hint: "Pull a fresh working copy before further edits. Snapshot checks are not atomic concurrency protection."}
	for i := 0; i < len(plan.Edits) || i == 0; i++ {
		current, err := readDocsWorkspaceSnapshot(runtime, expected.DocumentID)
		if err != nil {
			return failDocsPush(runtime, result, err, false)
		}
		if current.RevisionID != expected.RevisionID || !docsWorkspaceJSONEqual(current.Blocks, expected.Blocks) {
			return failDocsPush(runtime, result, docsWorkspaceError("current document differs from the expected native snapshot"), false)
		}
		if len(plan.Edits) == 0 {
			result.Verified = true
			runtime.Out(result, nil)
			return nil
		}
		edit := plan.Edits[i]
		wanted, err := docsWorkspaceEditedBlocks(current.Blocks, edit)
		if err != nil {
			return failDocsPush(runtime, result, err, false)
		}
		result.AttemptedID = edit.BlockID
		var acknowledgement json.RawMessage
		err = docsWorkspaceAPI(runtime, "PATCH", docsWorkspacePath(expected.DocumentID)+"/blocks/"+url.PathEscape(edit.BlockID),
			map[string]interface{}{"document_revision_id": strconv.FormatInt(current.RevisionID, 10)},
			map[string]interface{}{"update_text_elements": map[string]interface{}{"elements": edit.Elements}}, &acknowledgement)
		if err != nil {
			return failDocsPush(runtime, result, err, true)
		}
		if summary := runtime.Factory.AgentAttemptSummary(); summary.Unknown > 0 || summary.Unapplied > 0 || summary.PersistenceFailed {
			return failDocsPush(runtime, result, docsWorkspaceError("mutation evidence is incomplete; inspect receipts before continuing"), true)
		}
		verified, err := readDocsWorkspaceSnapshot(runtime, expected.DocumentID)
		if err != nil {
			return failDocsPush(runtime, result, err, true)
		}
		if verified.RevisionID <= current.RevisionID || !docsWorkspaceJSONEqual(verified.Blocks, wanted) {
			return failDocsPush(runtime, result, docsWorkspaceError("native readback does not match the planned block edit"), true)
		}
		expected = verified
		result.RevisionID = verified.RevisionID
		result.ChangedBlocks = append(result.ChangedBlocks, docsPushChangedBlock{BlockID: edit.BlockID, Text: edit.Text})
		result.AttemptedID = ""
	}
	result.Outcome, result.Verified = "verified", true
	runtime.Out(result, nil)
	return nil
}

func failDocsPush(runtime *common.RuntimeContext, result docsPushResult, cause error, dispatched bool) error {
	if !dispatched && len(result.ChangedBlocks) == 0 {
		return cause
	}
	result.Outcome = "partial"
	if dispatched {
		result.Outcome = "unknown"
	}
	result.Verified, result.Unchanged = false, false
	_ = runtime.OutPartialFailure(result, nil)
	summary := runtime.Factory.AgentAttemptSummary()
	ids := make([]string, len(summary.Receipts))
	for i, receipt := range summary.Receipts {
		ids[i] = receipt.ID
	}
	return errs.NewMutationOutcomeError(result.Outcome, ids, summary.PersistenceFailed, cause)
}

func docsWorkspaceEditedBlocks(blocks []json.RawMessage, edit docworkspace.Edit) ([]json.RawMessage, error) {
	wanted := append([]json.RawMessage(nil), blocks...)
	for i, raw := range blocks {
		var block map[string]json.RawMessage
		if json.Unmarshal(raw, &block) != nil {
			return nil, docsWorkspaceError("invalid native block")
		}
		var id string
		if json.Unmarshal(block["block_id"], &id) != nil || id != edit.BlockID {
			continue
		}
		// Plan already validated the writable text element subset. Preserve all
		// other fields while constructing the expected full native readback.
		var kind int
		if json.Unmarshal(block["block_type"], &kind) != nil || kind < 2 || kind > 8 {
			return nil, docsWorkspaceError("planned native block type is unsupported")
		}
		name := "text"
		if kind > 2 {
			name = "heading" + strconv.Itoa(kind-2)
		}
		var text map[string]json.RawMessage
		if json.Unmarshal(block[name], &text) != nil || text == nil {
			return nil, docsWorkspaceError("invalid native text block")
		}
		elements, err := json.Marshal(edit.Elements)
		if err != nil {
			return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "cannot encode planned elements").WithCause(err)
		}
		text["elements"] = elements
		block[name], err = json.Marshal(text)
		if err != nil {
			return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "cannot encode expected text block").WithCause(err)
		}
		wanted[i], err = json.Marshal(block)
		if err != nil {
			return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "cannot encode expected native block").WithCause(err)
		}
		return wanted, nil
	}
	return nil, docsWorkspaceError("planned native text block is missing")
}
