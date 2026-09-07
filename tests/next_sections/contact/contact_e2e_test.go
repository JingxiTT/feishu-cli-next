// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package contact_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/tidwall/gjson"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }

func requireSuccess(t *testing.T, result nextfixture.Result) {
	t.Helper()
	if result.ExitCode != 0 || result.Stderr != "" || !gjson.Get(result.Stdout, "ok").Bool() {
		t.Fatalf("CLI failed: %+v", result)
	}
	if strings.Contains(result.Stdout, `"mutation"`) {
		t.Fatalf("read generated mutation evidence: %s", result.Stdout)
	}
}

func body(t *testing.T, r *http.Request) string {
	t.Helper()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
	}
	return string(b)
}

func respond(w http.ResponseWriter, data string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"code":0,"data":%s}`, data)
}

func TestContactGetUserIdentityRoutesAndFullProfile(t *testing.T) {
	for _, identity := range []string{"user", "bot"} {
		for _, idType := range []string{"open_id", "union_id", "user_id"} {
			t.Run(identity+"/"+idType, func(t *testing.T) {
				var requests atomic.Int32
				f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if r.URL.Query().Get("user_id_type") != idType {
						t.Errorf("id type: %s", r.URL)
					}
					profile := `{"user_id":"fixture_id","name":"同名 😀","department_ids":["001","002"],"custom_attrs":[{"type":"TEXT","value":{"text":""}}],"status":{"is_frozen":false},"email":null}`
					if identity == "bot" {
						if r.Method != "GET" || r.URL.Path != "/open-apis/contact/v3/users/fixture_id" {
							t.Errorf("bot route: %s %s", r.Method, r.URL)
						}
						respond(w, `{"user":`+profile+`}`)
					} else {
						if r.Method != "POST" || r.URL.Path != "/open-apis/contact/v3/users/basic_batch" || gjson.Get(body(t, r), "user_ids.0").String() != "fixture_id" {
							t.Errorf("user route: %s %s", r.Method, r.URL)
						}
						respond(w, `{"users":[`+profile+`]}`)
					}
				}))
				result := f.Run("contact", "+get-user", "--as", identity, "--user-id", "fixture_id", "--user-id-type", idType)
				requireSuccess(t, result)
				for _, field := range []string{"data.user.department_ids.0", "data.user.custom_attrs.0.value.text", "data.user.email", "data.user.status.is_frozen"} {
					if !gjson.Get(result.Stdout, field).Exists() {
						t.Errorf("lost field %s: %s", field, result.Stdout)
					}
				}
				if requests.Load() != 1 {
					t.Errorf("requests: %d", requests.Load())
				}
			})
		}
	}
	f := nextfixture.NewFixture(t, nil)
	self := f.Run("contact", "+get-user", "--as", "user")
	requireSuccess(t, self)
	if gjson.Get(self.Stdout, "data.user.open_id").String() != "ou_foundation_fixture" || gjson.Get(self.Stdout, "data.user.tenant_key").String() != "foundation-fixture-tenant" {
		t.Fatalf("self binding: %s", self.Stdout)
	}
}

const userMatches = `{"items":[{"id":"ou_same_a","display_info":"<h>Same</h>\nDepartment A\n[Contacted yesterday]","meta_data":{"i18n_names":{"zh_cn":"同名","en_us":"Same"},"mail_address":"","enterprise_mail_address":"a@example.invalid","is_registered":true,"chat_id":"oc_a","is_cross_tenant":false,"description":"签名 😀"}},{"id":"ou_same_b","display_info":"<h>Same</h>\nDepartment B","meta_data":{"i18n_names":{"en_us":"Same"},"chat_id":"","is_cross_tenant":true}}],"has_more":true,"page_token":"not-exposed","notice":"narrow your search"}`

func TestContactSearchUsersPreservesAmbiguityAndBounds(t *testing.T) {
	var requests atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != "POST" || r.URL.Path != "/open-apis/contact/v3/users/search" || r.URL.Query().Get("page_size") != "30" {
			t.Errorf("search route: %s %s", r.Method, r.URL)
		}
		b := body(t, r)
		if gjson.Get(b, "query").String() != "Same" || gjson.Get(b, "filter.user_ids.0").String() != "ou_foundation_fixture" {
			t.Errorf("search input: %s", b)
		}
		for _, flag := range []string{"has_contact", "has_enterprise_email", "exclude_outer_contact", "is_resigned"} {
			if !gjson.Get(b, "filter."+flag).Bool() {
				t.Errorf("missing filter %s: %s", flag, b)
			}
		}
		respond(w, userMatches)
	}))
	result := f.Run("contact", "+search-user", "--as", "user", "--query", "Same", "--user-ids", "me,ou_same_b", "--lang", "en-US", "--page-size", "30", "--has-chatted", "--has-enterprise-email", "--exclude-external-users", "--left-organization")
	requireSuccess(t, result)
	if len(gjson.Get(result.Stdout, "data.users").Array()) != 2 || !gjson.Get(result.Stdout, "data.has_more").Bool() || gjson.Get(result.Stdout, "data.notice").String() != "narrow your search" {
		t.Fatalf("lost ambiguity/extent: %s", result.Stdout)
	}
	for i, id := range []string{"ou_same_a", "ou_same_b"} {
		u := gjson.Get(result.Stdout, fmt.Sprintf("data.users.%d", i))
		if u.Get("open_id").String() != id || u.Get("localized_name").String() != "Same" {
			t.Fatalf("selected or reordered same-name candidates: %s", result.Stdout)
		}
	}
	if gjson.Get(result.Stdout, "data.users.0.signature").String() != "签名 😀" || !gjson.Get(result.Stdout, "data.users.1.is_cross_tenant").Bool() || gjson.Get(result.Stdout, "data.users.1.p2p_chat_id").String() != "" {
		t.Fatalf("lost profile facts: %s", result.Stdout)
	}
	if requests.Load() != 1 || strings.Contains(result.Stdout, "not-exposed") || strings.Contains(result.Stdout, "recipient_verified") {
		t.Fatalf("search paginated or invented verification: %d %s", requests.Load(), result.Stdout)
	}
}

func TestContactSearchBotsRequestAndProjection(t *testing.T) {
	var requests atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		b := body(t, r)
		if r.Method != "POST" || r.URL.Path != "/open-apis/bot/v4/bot/search" || r.URL.Query().Get("page_size") != "1" || gjson.Get(b, "query").String() != "Helper" || !gjson.Get(b, "filter.has_chatter").Bool() || gjson.Get(b, "filter.chat_ids.0").String() != "oc_fixture" {
			t.Errorf("bot request: %s %s %s", r.Method, r.URL, b)
		}
		respond(w, `{"items":[{"id":"ou_bot","display_info":"<h>Helper</h>\nA &amp; B","meta_data":{"tenant_id":"tenant_fixture","chat_id":"","enable_join_group":false,"is_agent":true}}],"has_more":true,"notice":"limited"}`)
	}))
	result := f.Run("contact", "+search-bot", "--as", "user", "--query", "Helper", "--chat-ids", "oc_fixture", "--has-chatted", "--page-size", "1")
	requireSuccess(t, result)
	for path, want := range map[string]string{"data.bots.0.open_id": "ou_bot", "data.bots.0.tenant_id": "tenant_fixture", "data.bots.0.name": "Helper", "data.bots.0.description": "A & B", "data.notice": "limited"} {
		if gjson.Get(result.Stdout, path).String() != want {
			t.Errorf("%s: %s", path, result.Stdout)
		}
	}
	if requests.Load() != 1 || !gjson.Get(result.Stdout, "data.has_more").Bool() || !gjson.Get(result.Stdout, "data.bots.0.is_agent").Bool() || !gjson.Get(result.Stdout, "data.bots.0.chat_id").Exists() {
		t.Fatalf("bot projection: %s", result.Stdout)
	}
}

func TestContactFanoutPartialAllFailedAndEmpty(t *testing.T) {
	for _, command := range []string{"+search-user", "+search-bot"} {
		t.Run(command, func(t *testing.T) {
			var requests atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				query := gjson.Get(body(t, r), "query").String()
				if query == "bad" || query == "bad2" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"code":123456,"msg":"synthetic recoverable rejection","log_id":"synthetic-log"}`)
					return
				}
				respond(w, `{"items":[],"has_more":true,"notice":"incomplete"}`)
			}))
			partial := f.Run("contact", command, "--as", "user", "--queries", "ok,bad,ok", "--result-fields", "notice")
			requireSuccess(t, partial)
			summaries := gjson.Get(partial.Stdout, "data.queries").Array()
			if len(summaries) != 2 || summaries[0].Get("query").String() != "ok" || !summaries[0].Get("has_more").Bool() || summaries[1].Get("error").String() == "" || requests.Load() != 2 {
				t.Fatalf("partial failure lost or dedup changed: %s requests=%d", partial.Stdout, requests.Load())
			}
			failed := f.Run("contact", command, "--as", "user", "--queries", "bad,bad2")
			if failed.ExitCode == 0 || failed.Stdout != "" || gjson.Get(failed.Stderr, "error.code").Int() != 123456 {
				t.Fatalf("all failed classification: %+v", failed)
			}
			if requests.Load() != 4 {
				t.Fatalf("unexpected retry: %d", requests.Load())
			}
		})
	}
}

func TestContactProfileTypesArtifactAndNoClobber(t *testing.T) {
	var requests atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		b := body(t, r)
		if r.Method != "POST" || r.URL.Path != "/open-apis/profile/v2/user_profiles/batch_query" || r.URL.Query().Get("user_id_type") != "union_id" || gjson.Get(b, "user_ids.0").String() != "on_fixture" || !gjson.Get(b, "query_option.include_personal_status").Bool() || !gjson.Get(b, "query_option.include_description").Bool() {
			t.Errorf("profile request: %s %s %s", r.Method, r.URL, b)
		}
		respond(w, `{"user_profiles":[{"user_id":"on_fixture","personal_status":{"personal_status_id":"001","title":"会议 😀","i18n_title":{"en_us":"Meeting"},"icon_key":"GeneralInMeetingBusy","effective_interval":{"start_time":"0","end_time":"9007199254740993"},"is_not_disturb_mode":false},"description":{"default_locale":"zh_cn","default_value":"","i18n_value":{"ja_jp":"日本語","en_us":null}}}]}`)
	}))
	artifact := filepath.Join(f.WorkDir, "profile.json")
	args := []string{"contact", "user_profiles", "batch_query", "--as", "user", "--user-id-type", "union_id", "--data", `{"user_ids":["on_fixture"],"query_option":{"include_personal_status":true,"include_description":true}}`}
	result := f.Run(append(args, "--save-result", artifact)...)
	requireSuccess(t, result)
	data, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]json.RawMessage
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if gjson.Get(s, "schema").String() != "feishu-cli-next/result/v1" || gjson.Get(s, "binding.user_id").String() != "ou_foundation_fixture" || !gjson.Get(s, "binding.tenant_verified").Bool() {
		t.Fatalf("artifact binding: %s", s)
	}
	base := "result.data.user_profiles.0"
	for _, field := range []string{"personal_status.personal_status_id", "personal_status.is_not_disturb_mode", "description.default_value", "description.i18n_value.en_us"} {
		if !gjson.Get(s, base+"."+field).Exists() {
			t.Errorf("lost field: %s", field)
		}
	}
	if gjson.Get(s, base+".personal_status.effective_interval.end_time").String() != "9007199254740993" || gjson.Get(s, base+".description.default_value").String() != "" {
		t.Fatalf("changed types: %s", s)
	}
	again := f.Run(append(args, "--save-result", artifact)...)
	if again.ExitCode == 0 || requests.Load() != 1 {
		t.Fatalf("artifact collision reached backend: %+v requests=%d", again, requests.Load())
	}
	after, err := os.ReadFile(artifact)
	if err != nil || string(after) != s {
		t.Fatal("artifact was clobbered")
	}
}

func TestContactDryRunsAndInvalidInputs(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	cases := []struct {
		args  []string
		route string
	}{
		{[]string{"+get-user", "--as", "user"}, "/authen/v1/user_info"},
		{[]string{"+get-user", "--as", "bot", "--user-id", "ou_fixture"}, "/contact/v3/users/ou_fixture"},
		{[]string{"+get-user", "--as", "user", "--user-id", "ou_fixture"}, "/contact/v3/users/basic_batch"},
		{[]string{"+search-user", "--as", "user", "--queries", "A,B"}, "/contact/v3/users/search"},
		{[]string{"+search-bot", "--as", "user", "--queries", "A,B"}, "/bot/v4/bot/search"},
		{[]string{"user_profiles", "batch_query", "--as", "user", "--data", `{"user_ids":["ou_fixture"]}`}, "/profile/v2/user_profiles/batch_query"},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			args := append([]string{"contact"}, tc.args...)
			result := f.Run(append(args, "--dry-run")...)
			if result.ExitCode != 0 || !strings.Contains(result.Stdout, tc.route) {
				t.Fatalf("dry run: %+v", result)
			}
		})
	}
	for _, args := range [][]string{
		{"+get-user", "--as", "bot"}, {"+search-user", "--as", "bot", "--query", "x"}, {"+search-user", "--as", "user"},
		{"+search-user", "--as", "user", "--query", "x", "--queries", "y"}, {"+search-user", "--as", "user", "--query", "x", "--page-size", "31"},
		{"+search-user", "--as", "user", "--has-chatted=false"}, {"+search-user", "--as", "user", "--user-ids", "invalid"},
		{"+search-bot", "--as", "user", "--has-chatted"}, {"+search-bot", "--as", "bot", "--query", "x"}, {"+search-bot", "--as", "user", "--query", "x", "--has-chatted=false"},
		{"user_profiles", "batch_query", "--as", "bot", "--data", `{"user_ids":["ou_fixture"]}`},
	} {
		result := f.Run(append([]string{"contact"}, args...)...)
		if result.ExitCode == 0 || result.Stdout != "" || !gjson.Get(result.Stderr, "error.type").Exists() {
			t.Fatalf("invalid %v: %+v", args, result)
		}
	}
}

func TestContactLiveAccountBoundReads(t *testing.T) {
	t.Skip("live blocked: use coordinator-verified effective current account and tenant; inherited live helpers are not bound fixtures; no recipient delivery is performed by contact reads")
}

func TestContactUserFanoutPreservesMissingScopeFailure(t *testing.T) {
	var requests atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":99991679,"msg":"Access denied. missing scope","log_id":"synthetic-log"}`)
	}))
	result := f.Run("contact", "+search-user", "--as", "user", "--queries", "a,b")
	if result.ExitCode != 3 || result.Stdout != "" || gjson.Get(result.Stderr, "error.type").String() != "authorization" || gjson.Get(result.Stderr, "error.subtype").String() != "missing_scope" || gjson.Get(result.Stderr, "error.code").Int() != 99991679 || gjson.Get(result.Stderr, "error.log_id").String() != "synthetic-log" || requests.Load() != 2 {
		t.Fatalf("scope classification/retry: %+v requests=%d", result, requests.Load())
	}
}
