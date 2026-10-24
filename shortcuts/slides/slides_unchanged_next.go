// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package slides

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/validate"
	"github.com/larksuite/cli/shortcuts/common"
)

// unchangedSlide uses a current, target-bound read. XML is not reserialized or
// canonicalized: text whitespace, namespaces, element IDs and all rich content
// remain significant. Only the existing update normalization of the page's note
// ID is ignored. This observation is not a concurrency guarantee.
func unchangedSlide(runtime *common.RuntimeContext, presentationID, slideID, content string) (int, bool, error) {
	data, err := runtime.CallAPITyped("GET", fmt.Sprintf("/open-apis/slides_ai/v1/xml_presentations/%s/slide", validate.EncodePathSegment(presentationID)), map[string]interface{}{"slide_id": slideID, "revision_id": -1}, nil)
	if err != nil {
		return 0, false, err
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return 0, false, errs.NewInternalError(errs.SubtypeInvalidResponse, "invalid slide preflight response").WithCause(err)
	}
	var current struct {
		RevisionID *int `json:"revision_id"`
		Slide      struct {
			SlideID string `json:"slide_id"`
			Content string `json:"content"`
		} `json:"slide"`
	}
	if err := json.Unmarshal(encoded, &current); err != nil {
		return 0, false, errs.NewInternalError(errs.SubtypeInvalidResponse, "invalid slide preflight response").WithCause(err)
	}
	rootID, xmlErr := checkSlideRoot(current.Slide.Content)
	if current.RevisionID == nil || *current.RevisionID < 0 || current.Slide.SlideID != slideID || rootID != slideID || xmlErr != nil {
		return 0, false, errs.NewInternalError(errs.SubtypeInvalidResponse, "slide preflight did not return the requested complete page")
	}
	return *current.RevisionID, stripSlideNoteID(strings.TrimSpace(current.Slide.Content)) == content, nil
}
