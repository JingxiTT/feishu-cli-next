// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package slides

import (
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/httpmock"
)

func TestReplacePagesStopAfterDeleteFailureRetainsRecoveryIDs(t *testing.T) {
	f, out, _, reg := cmdutil.TestFactory(t, slidesTestConfig(t, ""))
	reg.Register(&httpmock.Stub{Method: "POST", URL: "/open-apis/slides_ai/v1/xml_presentations/pres_abc/slide", Body: map[string]interface{}{"code": 0, "data": map[string]interface{}{"slide_id": "new1", "revision_id": 11}}})
	reg.Register(&httpmock.Stub{Method: "DELETE", URL: "/open-apis/slides_ai/v1/xml_presentations/pres_abc/slide", Body: map[string]interface{}{"code": 3350001, "msg": "invalid param", "data": map[string]interface{}{}}})
	err := runSlidesShortcut(t, f, out, SlidesReplacePages, []string{"+replace-pages", "--presentation", "pres_abc", "--pages", `[{"slide_id":"old1","content":"<slide/>"}]`, "--as", "user"})
	p, ok := errs.ProblemOf(err)
	if !ok || p.Code != 3350001 || !strings.Contains(p.Hint, "new1") || !strings.Contains(p.Hint, "old1") || !strings.Contains(p.Hint, "before retrying") {
		t.Fatalf("error=%v problem=%#v", err, p)
	}
}

func TestReplacePagesStopAfterDeleteFailureEmitsObservedEffects(t *testing.T) {
	f, out, _, reg := cmdutil.TestFactory(t, slidesTestConfig(t, ""))
	reg.Register(&httpmock.Stub{Method: "POST", URL: "/open-apis/slides_ai/v1/xml_presentations/pres_abc/slide", Body: map[string]interface{}{"code": 0, "data": map[string]interface{}{"slide_id": "new1", "revision_id": 11}}})
	reg.Register(&httpmock.Stub{Method: "DELETE", URL: "/open-apis/slides_ai/v1/xml_presentations/pres_abc/slide", Body: map[string]interface{}{"code": 3350001, "msg": "invalid param", "data": map[string]interface{}{}}})
	err := runSlidesShortcut(t, f, out, SlidesReplacePages, []string{"+replace-pages", "--presentation", "pres_abc", "--pages", `[{"slide_id":"old1","content":"<slide/>"}]`, "--as", "user"})
	p, ok := errs.ProblemOf(err)
	if !ok || p.Code != 3350001 {
		t.Fatal(err)
	}
	env := decodeReplacePagesEnvelope(t, out)
	if env.OK {
		t.Fatal("partial result reported success")
	}
	results := env.Data["results"].([]interface{})
	first := results[0].(map[string]interface{})
	if first["old_slide_id"] != "old1" || first["new_slide_id"] != "new1" || first["status"] != "delete_failed" {
		t.Fatal(first)
	}
}
