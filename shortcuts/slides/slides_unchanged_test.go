// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package slides

import (
	"net/http"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/httpmock"
)

func TestUpdateSlideSkipUnchangedPreservesRichPage(t *testing.T) {
	f, out, _, reg := cmdutil.TestFactory(t, slidesTestConfig(t, ""))
	content := `<slide id="p1"><data><shape id="b1"><content><p>你好  café</p></content></shape><svg xmlns="http://www.w3.org/2000/svg"><path d="M0 0"/></svg></data><note id="n1">note</note></slide>`
	read := &httpmock.Stub{Method: "GET", URL: "/open-apis/slides_ai/v1/xml_presentations/pres_abc/slide", Body: map[string]interface{}{"code": 0, "data": map[string]interface{}{"revision_id": 7, "slide": map[string]interface{}{"slide_id": "p1", "content": content}}}}
	read.OnMatch = func(r *http.Request) {
		if r.URL.Query().Get("revision_id") != "-1" {
			t.Error("missing latest read")
		}
	}
	reg.Register(read)
	if err := runSlidesShortcut(t, f, out, SlidesUpdateSlide, []string{"+update-slide", "--presentation", "pres_abc", "--slide-id", "p1", "--content", content, "--skip-unchanged", "--as", "user"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"unchanged": true`) && !strings.Contains(out.String(), `"unchanged":true`) {
		t.Fatal(out.String())
	}
}

func TestUpdateSlideSkipUnchangedRejectsMismatchedRead(t *testing.T) {
	f, out, _, reg := cmdutil.TestFactory(t, slidesTestConfig(t, ""))
	reg.Register(&httpmock.Stub{Method: "GET", URL: "/open-apis/slides_ai/v1/xml_presentations/pres_abc/slide", Body: map[string]interface{}{"code": 0, "data": map[string]interface{}{"slide": map[string]interface{}{"slide_id": "p2", "content": `<slide id="p1"/>`}}}})
	err := runSlidesShortcut(t, f, out, SlidesUpdateSlide, []string{"+update-slide", "--presentation", "pres_abc", "--slide-id", "p1", "--content", `<slide id="p1"/>`, "--skip-unchanged", "--as", "user"})
	p, ok := errs.ProblemOf(err)
	if !ok || p.Subtype != errs.SubtypeInvalidResponse {
		t.Fatalf("error = %v", err)
	}
}
