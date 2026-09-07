// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package business_test

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }

type methodCase struct {
	command, method, path, body string
	flags                       []string
	query                       url.Values
	highRisk, bot               bool
}

// Explicit wire expectations are independent of production metadata/URL builders.
// Every documented body field is exercised, including empty and false values.
var methods = []methodCase{
	{command: "approval approvals get", method: "GET", path: "/open-apis/approval/v4/approvals/definition_fixture/detail", flags: []string{"--approval-code", "definition_fixture", "--locale", "zh-TW"}, query: url.Values{"locale": {"zh-TW"}}},
	{command: "approval approvals search", method: "POST", path: "/open-apis/approval/v4/approvals/search_launchable", body: `{"keyword":"请假 👋","locale":"ja-JP","page_size":2,"page_token":"cursor_01"}`},
	{command: "approval instances get", method: "GET", path: "/open-apis/approval/v4/instances/detail", flags: []string{"--instance-code", "instance_fixture", "--locale", "de-DE", "--user-id-type", "union_id"}, query: url.Values{"instance_code": {"instance_fixture"}, "locale": {"de-DE"}, "user_id_type": {"union_id"}}},
	{command: "approval instances create", method: "POST", path: "/open-apis/approval/v4/instances/initiate", highRisk: true, body: `{"approval_code":"definition_fixture","uuid":"business-idempotency-fixture","form":"[{\"id\":\"field\",\"type\":\"input\",\"value\":\"  审批 👋  \"}]","node_approver_list":[{"key":"custom_node","value":["ou_synthetic"]}],"node_cc_list":[{"key":"custom_cc","value":[]}]}`},
	{command: "approval instances cancel", method: "POST", path: "/open-apis/approval/v4/instances/recall", highRisk: true, body: `{"instance_code":"instance_fixture"}`},
	{command: "approval instances cc", method: "POST", path: "/open-apis/approval/v4/instances/add_cc", highRisk: true, flags: []string{"--user-id-type", "open_id"}, query: url.Values{"user_id_type": {"open_id"}}, body: `{"instance_code":"instance_fixture","cc_user_ids":["ou_synthetic"],"comment":""}`},
	{command: "approval instances initiated", method: "GET", path: "/open-apis/approval/v4/instances/initiated", flags: []string{"--page-size", "2", "--page-token", "cursor_01", "--locale", "ja-JP", "--definition-code", "definition_fixture", "--start-timestamp", "001788775200000", "--end-timestamp", "1788861600000", "--keyword", "审批 👋", "--user-id-type", "user_id"}, query: url.Values{"page_size": {"2"}, "page_token": {"cursor_01"}, "locale": {"ja-JP"}, "definition_code": {"definition_fixture"}, "start_timestamp": {"001788775200000"}, "end_timestamp": {"1788861600000"}, "keyword": {"审批 👋"}, "user_id_type": {"user_id"}}},
	{command: "approval tasks approve", method: "POST", path: "/open-apis/approval/v4/tasks/pass", highRisk: true, body: `{"instance_code":"instance_fixture","task_id":"001234567890123456789","form":"[{\"id\":\"field\",\"type\":\"input\",\"value\":\"\"}]","comment":"同意\n  retain spaces  "}`},
	{command: "approval tasks reject", method: "POST", path: "/open-apis/approval/v4/tasks/refuse", highRisk: true, body: `{"instance_code":"instance_fixture","task_id":"task_fixture","comment":"拒绝"}`},
	{command: "approval tasks transfer", method: "POST", path: "/open-apis/approval/v4/tasks/forward", highRisk: true, flags: []string{"--user-id-type", "union_id"}, query: url.Values{"user_id_type": {"union_id"}}, body: `{"instance_code":"instance_fixture","task_id":"task_fixture","transfer_user_id":"on_synthetic","comment":"转交"}`},
	{command: "approval tasks query", method: "GET", path: "/open-apis/approval/v4/tasks", flags: []string{"--topic", "18", "--page-size", "2", "--page-token", "cursor_01", "--locale", "en-US", "--definition-code", "definition_fixture", "--start-timestamp", "001788775200000", "--end-timestamp", "1788861600000", "--keyword", "审批 👋", "--user-id-type", "open_id"}, query: url.Values{"topic": {"18"}, "page_size": {"2"}, "page_token": {"cursor_01"}, "locale": {"en-US"}, "definition_code": {"definition_fixture"}, "start_timestamp": {"001788775200000"}, "end_timestamp": {"1788861600000"}, "keyword": {"审批 👋"}, "user_id_type": {"open_id"}}},
	{command: "approval tasks add_sign", method: "POST", path: "/open-apis/approval/v4/tasks/add_sign", highRisk: true, flags: []string{"--user-id-type", "user_id"}, query: url.Values{"user_id_type": {"user_id"}}, body: `{"instance_code":"instance_fixture","task_id":"task_fixture","comment":"加签","add_sign_user_ids":["user_synthetic"],"add_sign_type":3,"approval_method":3}`},
	{command: "approval tasks rollback", method: "POST", path: "/open-apis/approval/v4/tasks/rollback", highRisk: true, body: `{"instance_code":"instance_fixture","task_id":"task_fixture","comment":"退回","node_ids":["START","custom_node"]}`},
	{command: "approval tasks remind", method: "POST", path: "/open-apis/approval/v4/instances/remind", highRisk: true, body: `{"instance_code":"instance_fixture","task_ids":["task_fixture","task_other"],"comment":"催办"}`},
	{command: "attendance user_tasks query", method: "POST", path: "/open-apis/attendance/v1/user_tasks/query", bot: true, flags: []string{"--employee-type", "employee_no", "--ignore-invalid-users=false", "--include-terminated-user=true"}, query: url.Values{"employee_type": {"employee_no"}, "ignore_invalid_users": {"false"}, "include_terminated_user": {"true"}}, body: `{"user_ids":["000001"],"check_date_from":20260901,"check_date_to":20260907,"need_overtime_result":false}`},
	{command: "mindnotes nodes create", method: "POST", path: "/open-apis/mindnote/v1/mindnotes/mind_fixture/nodes", bot: true, flags: []string{"--mindnote-id", "mind_fixture", "--user-id-type", "union_id"}, query: url.Values{"user_id_type": {"union_id"}}, body: `{"client_token":"mind-idempotency-fixture","nodes":[{"node_id":"node_fixture","parent_id":"root_fixture","finish":false,"highlight":"grey","images":[{"token":"img_synthetic"}],"texts":[{"element_type":"text","text":{"content":"  Unicode 👋\n"},"style":{"bold":true,"italic":false}},{"element_type":"link","link":{"url":"https://example.invalid/?q=%20","content":"link"}},{"element_type":"user","mention_user":{"user_id":"on_synthetic"}},{"element_type":"doc","mention_doc":{"token":"doc_synthetic"}}],"notes":[{"element_type":"text","text":{"content":""},"style":{}},{"element_type":"link","link":{"url":"https://example.invalid"}},{"element_type":"user","mention_user":{"user_id":"on_synthetic"}},{"element_type":"doc","mention_doc":{"token":"doc_synthetic"}}]}]}`},
	{command: "mindnotes nodes list", method: "GET", path: "/open-apis/mindnote/v1/mindnotes/mind_fixture/nodes", flags: []string{"--mindnote-id", "mind_fixture", "--user-id-type", "open_id"}, query: url.Values{"user_id_type": {"open_id"}}},
}

func argsFor(tc methodCase, identity string) []string {
	args := append(strings.Fields(tc.command), tc.flags...)
	args = append(args, "--as", identity)
	if tc.highRisk {
		args = append(args, "--yes")
	}
	if tc.body != "" {
		args = append(args, "--data", tc.body)
	}
	return args
}
func reply(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, body)
}
func success(t *testing.T, r nextfixture.Result) {
	t.Helper()
	require.Equal(t, 0, r.ExitCode, "stdout=%s stderr=%s", r.Stdout, r.Stderr)
	require.True(t, gjson.Get(r.Stdout, "ok").Bool(), r.Stdout)
}
func outcome(t *testing.T, r nextfixture.Result, want string) {
	t.Helper()
	require.Contains(t, r.Stderr, `"outcome":"`+want+`"`, r.Stderr)
}

func businessRead(tc methodCase) bool {
	return tc.method == "GET" || tc.command == "approval approvals search" || tc.command == "attendance user_tasks query"
}
func unknown(t *testing.T, r nextfixture.Result) {
	t.Helper()
	require.Equal(t, 22, r.ExitCode, "stdout=%s stderr=%s", r.Stdout, r.Stderr)
	require.False(t, gjson.Get(r.Stdout, "ok").Bool())
	outcome(t, r, "unknown")
}

func TestEveryBusinessMethodWireAndFileInput(t *testing.T) {
	require.Len(t, methods, 17)
	for _, tc := range methods {
		t.Run(tc.command, func(t *testing.T) {
			identities := []string{"user"}
			if tc.bot {
				identities = append(identities, "bot")
			}
			for _, identity := range identities {
				t.Run(identity, func(t *testing.T) {
					var requests atomic.Int32
					f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						requests.Add(1)
						require.Equal(t, tc.method, r.Method)
						require.Equal(t, tc.path, r.URL.Path)
						wantQuery := tc.query
						if wantQuery == nil {
							wantQuery = url.Values{}
						}
						require.Equal(t, wantQuery, r.URL.Query())
						body, err := io.ReadAll(r.Body)
						require.NoError(t, err)
						if tc.body == "" {
							require.Empty(t, body)
						} else {
							require.JSONEq(t, tc.body, string(body))
						}
						reply(w, `{"code":0,"data":{"id":"0001","instance_code":"instance_fixture","ids":["node_fixture"],"client_token":"mind-idempotency-fixture","text":"  Unicode 👋\n","empty":"","nullable":null,"enabled":false,"integer":9007199254740993,"items":[]}}`)
					}))
					args := argsFor(tc, identity)
					if tc.body != "" {
						path := filepath.Join(f.WorkDir, "body.json")
						require.NoError(t, os.WriteFile(path, []byte(tc.body), 0600))
						args[len(args)-1] = "@" + path
					}
					r := f.Run(args...)
					if tc.command == "mindnotes nodes create" {
						unknown(t, r)
					} else {
						success(t, r)
					}
					require.Equal(t, int32(1), requests.Load())
					require.Equal(t, identity, gjson.Get(r.Stdout, "identity").String())
					require.Equal(t, "9007199254740993", gjson.Get(r.Stdout, "data.integer").Raw)
					require.Equal(t, "  Unicode 👋\n", gjson.Get(r.Stdout, "data.text").String())
					require.Equal(t, "null", gjson.Get(r.Stdout, "data.nullable").Raw)
					require.Equal(t, "false", gjson.Get(r.Stdout, "data.enabled").Raw)
					require.Equal(t, "", gjson.Get(r.Stdout, "data.empty").String())
					if !businessRead(tc) {
						if tc.command != "mindnotes nodes create" {
							outcome(t, r, "confirmed_applied")
						}
					} else {
						require.NotContains(t, r.Stderr, `"mutation"`)
					}
				})
			}
		})
	}
}

func TestEveryBusinessMethodIdentityPermissionAndUnknown(t *testing.T) {
	for _, tc := range methods {
		t.Run(tc.command, func(t *testing.T) {
			if !tc.bot {
				f := nextfixture.NewFixture(t, nil)
				r := f.Run(argsFor(tc, "bot")...)
				require.NotZero(t, r.ExitCode)
				require.Empty(t, r.Stdout)
				require.Contains(t, r.Stderr, "identity")
			}
			for _, mode := range []string{"scope", "lost_response", "domain_error"} {
				t.Run(mode, func(t *testing.T) {
					var requests atomic.Int32
					f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						requests.Add(1)
						switch mode {
						case "scope":
							reply(w, `{"code":99991679,"msg":"synthetic missing scope","error":{"log_id":"log_synthetic"}}`)
						case "domain_error":
							reply(w, `{"code":60012,"msg":"synthetic domain rejection"}`)
						case "lost_response":
							c, _, err := w.(http.Hijacker).Hijack()
							require.NoError(t, err)
							_ = c.Close()
						}
					}))
					r := f.Run(argsFor(tc, "user")...)
					require.NotZero(t, r.ExitCode, "%+v", r)
					if tc.method == "GET" && mode == "lost_response" {
						require.Equal(t, int32(2), requests.Load(), "inherited read transport reconnect")
					} else {
						require.Equal(t, int32(1), requests.Load(), "never replay mutations")
					}
					require.False(t, gjson.Get(r.Stdout, "ok").Bool(), r.Stdout)
					if !businessRead(tc) {
						if mode == "scope" {
							require.Equal(t, 20, r.ExitCode)
							outcome(t, r, "confirmed_unapplied")
						} else {
							require.Equal(t, 22, r.ExitCode)
							outcome(t, r, "unknown")
						}
					} else {
						require.NotContains(t, r.Stderr, `"mutation"`)
					}
					if mode == "scope" {
						require.Contains(t, r.Stderr, "99991679")
					}
				})
			}
		})
	}
}

func TestBusinessValidationAndDryRunNeverDispatch(t *testing.T) {
	for _, tc := range methods {
		t.Run(tc.command, func(t *testing.T) {
			f := nextfixture.NewFixture(t, nil)
			r := f.Run(append(argsFor(tc, "user"), "--dry-run")...)
			success(t, r)
			require.Contains(t, r.Stdout, tc.path)
			if tc.body != "" {
				args := argsFor(tc, "user")
				args[len(args)-1] = "{broken"
				r = f.Run(args...)
				require.Equal(t, 2, r.ExitCode)
				require.Empty(t, r.Stdout)
				require.Contains(t, r.Stderr, "invalid_argument")
				args[len(args)-1] = "@" + filepath.Join(f.WorkDir, "missing.json")
				r = f.Run(args...)
				require.NotZero(t, r.ExitCode)
				require.Empty(t, r.Stdout)
			}
			if tc.highRisk {
				args := []string{}
				for _, arg := range argsFor(tc, "user") {
					if arg != "--yes" {
						args = append(args, arg)
					}
				}
				r = f.Run(args...)
				require.NotZero(t, r.ExitCode)
				require.Empty(t, r.Stdout)
				require.Contains(t, r.Stderr, "yes")
			}
		})
	}
	f := nextfixture.NewFixture(t, nil)
	for _, args := range [][]string{
		{"approval", "approvals", "get"}, {"approval", "instances", "get"}, {"approval", "tasks", "query"},
		{"attendance", "user_tasks", "query"}, {"mindnotes", "nodes", "list"}, {"mindnotes", "nodes", "create"},
		{"mindnotes", "nodes", "list", "--mindnote-id", "../bad"},
		{"approval", "tasks", "query", "--topic", "1", "--page-size", "not-an-int"},
		{"attendance", "user_tasks", "query", "--employee-type", "employee_id", "--ignore-invalid-users=not-bool"},
	} {
		r := f.Run(append(args, "--as", "user")...)
		require.Equal(t, 2, r.ExitCode, "%+v", r)
		require.Empty(t, r.Stdout)
	}
}

func TestFrozenBusinessSchemaAndRouting(t *testing.T) {
	file, err := os.Open("../../../docs/reference/meta_data.json.gz")
	require.NoError(t, err)
	defer file.Close()
	zipped, err := gzip.NewReader(file)
	require.NoError(t, err)
	defer zipped.Close()
	var catalog struct {
		Services []struct {
			Name      string `json:"name"`
			Resources map[string]struct {
				Methods map[string]json.RawMessage `json:"methods"`
			} `json:"resources"`
		} `json:"services"`
	}
	require.NoError(t, json.NewDecoder(zipped).Decode(&catalog))
	f := nextfixture.NewFixture(t, nil)
	seen := 0
	for _, s := range catalog.Services {
		if s.Name != "approval" && s.Name != "attendance" && s.Name != "mindnotes" {
			continue
		}
		for resource, r := range s.Resources {
			for method, raw := range r.Methods {
				seen++
				t.Run(s.Name+"."+resource+"."+method, func(t *testing.T) {
					result := f.Run("schema", s.Name+"."+resource+"."+method)
					require.Equal(t, 0, result.ExitCode, result.Stderr)
					for _, pair := range [][2]string{{"parameters", "inputSchema.properties.params"}, {"requestBody", "inputSchema.properties.data"}, {"responseBody", "outputSchema"}} {
						assertFields(t, gjson.GetBytes(raw, pair[0]), gjson.Get(result.Stdout, pair[1]), pair[0] != "responseBody")
					}
					require.Equal(t, gjson.GetBytes(raw, "risk").String(), gjson.Get(result.Stdout, "_meta.risk").String())
					require.JSONEq(t, gjson.GetBytes(raw, "scopes").Raw, gjson.Get(result.Stdout, "_meta.scopes").Raw)
					identities := []string{}
					for _, identity := range gjson.GetBytes(raw, "accessTokens").Array() {
						value := identity.String()
						if value == "tenant" {
							value = "bot"
						}
						identities = append(identities, value)
					}
					actualIdentities := []string{}
					for _, identity := range gjson.Get(result.Stdout, "_meta.access_tokens").Array() {
						actualIdentities = append(actualIdentities, identity.String())
					}
					require.ElementsMatch(t, identities, actualIdentities)

					help := f.Run(s.Name, resource, method, "--help")
					require.Equal(t, 0, help.ExitCode, help.Stderr)
					for key := range gjson.GetBytes(raw, "parameters").Map() {
						require.Contains(t, help.Stdout, "--"+strings.ReplaceAll(key, "_", "-"))
					}
				})
			}
		}
	}
	require.Equal(t, 17, seen)
	for _, group := range []string{"approval", "approval approvals", "approval instances", "approval tasks", "attendance", "attendance user_tasks", "mindnotes", "mindnotes nodes"} {
		r := f.Run(append(strings.Fields(group), "--help")...)
		require.Equal(t, 0, r.ExitCode, r.Stderr)
	}
	for _, domain := range []string{"approval", "attendance", "mindnotes"} {
		r := f.Run("discover", domain)
		require.Equal(t, 0, r.ExitCode, r.Stderr)
		for _, tc := range methods {
			if strings.HasPrefix(tc.command, domain+" ") {
				require.Contains(t, r.Stdout, tc.command)
			}
		}
	}
}

// Recursively retain every frozen field, type, required marker and enum option.
func assertFields(t *testing.T, fields, schema gjson.Result, input bool) {
	t.Helper()
	for name, field := range fields.Map() {
		actual := schema.Get("properties." + name)
		require.True(t, actual.Exists(), name)
		require.Equal(t, field.Get("type").String(), actual.Get("type").String(), name)
		if input && field.Get("required").Bool() {
			require.Contains(t, schema.Get("required").Raw, `"`+name+`"`)
		}
		for _, option := range field.Get("options").Array() {
			values := []string{}
			for _, value := range actual.Get("enum").Array() {
				values = append(values, value.String())
			}
			require.Contains(t, values, option.Get("value").String(), name)
		}
		children := field.Get("properties")
		if children.Exists() {
			if field.Get("type").String() == "array" {
				actual = actual.Get("items")
			}
			assertFields(t, children, actual, input)
		}
	}
}

func TestApprovalPaginationAndReadArtifacts(t *testing.T) {
	for _, operation := range []struct {
		command, path, collection string
		extra                     []string
	}{
		{"approval instances initiated", "/open-apis/approval/v4/instances/initiated", "instances", nil},
		{"approval tasks query", "/open-apis/approval/v4/tasks", "tasks", []string{"--topic", "17"}},
	} {
		t.Run(operation.command, func(t *testing.T) {
			for _, limit := range []string{"1", "0"} {
				t.Run("limit_"+limit, func(t *testing.T) {
					var pages atomic.Int32
					f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						require.Equal(t, "GET", r.Method)
						require.Equal(t, operation.path, r.URL.Path)
						require.Equal(t, "2", r.URL.Query().Get("page_size"))
						if len(operation.extra) > 0 {
							require.Equal(t, "17", r.URL.Query().Get("topic"))
						}
						n := pages.Add(1)
						if n == 1 {
							require.Empty(t, r.URL.Query().Get("page_token"))
							reply(w, `{"code":0,"data":{"`+operation.collection+`":[{"instance_code":"instance_1","task_id":"task_1","form":"[]"}],"has_more":true,"page_token":"opaque_next","count":2}}`)
						} else {
							require.Equal(t, "opaque_next", r.URL.Query().Get("page_token"))
							reply(w, `{"code":0,"data":{"`+operation.collection+`":[{"instance_code":"instance_2","task_id":"task_2","form":"[\"保留\"]"}],"has_more":false,"page_token":"","count":2}}`)
						}
					}))
					args := append(strings.Fields(operation.command), operation.extra...)
					args = append(args, "--as", "user", "--page-size", "2", "--page-all", "--page-delay", "0", "--page-limit", limit)
					r := f.Run(args...)
					success(t, r)
					require.Contains(t, r.Stdout, "instance_1")
					if limit == "1" {
						require.Equal(t, int32(1), pages.Load())
						require.Contains(t, r.Stdout, "opaque_next")
						require.Contains(t, r.Stdout, "true")
					} else {
						require.Equal(t, int32(2), pages.Load())
						require.Contains(t, r.Stdout, "instance_2")
					}
				})
			}
		})
	}
	// Search keeps its documented body cursor; no typed pagination parameter is
	// declared. The next page is explicitly supplied through the JSON body.
	var searches atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/open-apis/approval/v4/approvals/search_launchable", r.URL.Path)
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		if searches.Add(1) == 1 {
			require.Equal(t, "", gjson.GetBytes(raw, "page_token").String())
			reply(w, `{"code":0,"data":{"approvals":[{"approval_code":"definition_1"}],"has_more":true,"page_token":"body_next"}}`)
		} else {
			require.Equal(t, "body_next", gjson.GetBytes(raw, "page_token").String())
			reply(w, `{"code":0,"data":{"approvals":[{"approval_code":"definition_2"}],"has_more":false}}`)
		}
	}))
	r := f.Run("approval", "approvals", "search", "--as", "user", "--data", `{"page_size":1}`)
	success(t, r)
	require.Contains(t, r.Stdout, "body_next")
	r = f.Run("approval", "approvals", "search", "--as", "user", "--data", `{"page_size":1,"page_token":"body_next"}`)
	success(t, r)
	require.Contains(t, r.Stdout, "definition_2")
	for _, tc := range methods {
		if tc.method != "GET" {
			continue
		}
		t.Run(tc.command+"_artifact", func(t *testing.T) {
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reply(w, `{"code":0,"data":{"nodes":[{"node_id":"node_1","texts":[{"element_type":"text","text":{"content":"保留"}}]}],"form":"[\"escaped\"]","title":"标题","has_more":true,"page_token":"next"}}`)
			}))
			path := filepath.Join(f.WorkDir, "full.json")
			r := f.Run(append(argsFor(tc, "user"), "--save-result", path, "--result-fields", "title")...)
			success(t, r)
			raw, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, "feishu-cli-next/result/v1", gjson.GetBytes(raw, "schema").String())
			require.Equal(t, "保留", gjson.GetBytes(raw, "result.data.nodes.0.texts.0.text.content").String())
			require.Equal(t, "user", gjson.GetBytes(raw, "binding.identity").String())
			require.Equal(t, "next", gjson.GetBytes(raw, "result.data.page_token").String())
			require.Contains(t, r.Stdout, "next")
		})
	}
}

func TestBusinessRawParamsOverlayAndPartialData(t *testing.T) {
	var requests atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		require.Equal(t, url.Values{"employee_type": {"employee_id"}, "ignore_invalid_users": {"false"}, "future_parameter": {"raw retained"}}, r.URL.Query())
		reply(w, `{"code":0,"data":{"user_task_results":[{"user_id":"0001","records":[{"check_in_record_id":"9007199254740993","task_shift_type":1}]}],"invalid_user_ids":["0002"],"unauthorized_user_ids":["0003"]}}`)
	}))
	path := filepath.Join(f.WorkDir, "params.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"employee_type":"employee_no","ignore_invalid_users":true,"future_parameter":"raw retained"}`), 0600))
	r := f.Run("attendance", "user_tasks", "query", "--as", "bot", "--params", "@"+path, "--employee-type", "employee_id", "--ignore-invalid-users=false", "--data", `{"user_ids":["0001","0002","0003"],"check_date_from":20260901,"check_date_to":20260907}`)
	// Keep the actual backend partial-result fields visible. Frozen metadata
	// declares write risk; the exact known read route bypasses mutation receipts.
	success(t, r)
	require.NotContains(t, r.Stderr, `"mutation"`)
	require.False(t, gjson.Get(r.Stdout, "mutation").Exists())
	require.Equal(t, int32(1), requests.Load())
	require.Equal(t, "0002", gjson.Get(r.Stdout, "data.invalid_user_ids.0").String())
	require.Equal(t, "0003", gjson.Get(r.Stdout, "data.unauthorized_user_ids.0").String())
	require.Equal(t, "9007199254740993", gjson.Get(r.Stdout, "data.user_task_results.0.records.0.check_in_record_id").String())
}

func TestBusinessLive(t *testing.T) {
	for _, tc := range methods {
		t.Run(tc.command, func(t *testing.T) {
			// Selectable, explicit blockers; never load personal profiles or create a
			// purported private fixture before validating backend visibility.
			reason := "no verified current-account/tenant-bound, run-owned fixture with private visibility and cleanup"
			if tc.highRisk {
				reason = "approval workflow may notify/assign other people; no verified Ying Wu-only participants, private workflow, and owned cleanup"
			}
			if strings.HasPrefix(tc.command, "attendance ") {
				reason = "no account-bound synthetic workforce fixture or verified current employee ID; other employee records are outside the test fixture"
			}
			t.Skip("BLOCKED live: " + reason)
		})
	}
}

func TestMindnoteRichContentReadbackAndExplicitReplay(t *testing.T) {
	tc := methods[15]
	var saved json.RawMessage
	var writes atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, tc.path, r.URL.Path)
		if r.Method == "POST" {
			writes.Add(1)
			raw, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Equal(t, "mind-idempotency-fixture", gjson.GetBytes(raw, "client_token").String())
			saved = append(saved[:0], []byte(gjson.GetBytes(raw, "nodes").Raw)...)
			reply(w, `{"code":0,"data":{"client_token":"mind-idempotency-fixture","ids":["node_fixture"]}}`)
		} else {
			reply(w, `{"code":0,"data":{"nodes":`+string(saved)+`}}`)
		}
	}))
	for i := 0; i < 2; i++ {
		r := f.Run(argsFor(tc, "user")...)
		unknown(t, r)
		require.False(t, gjson.Get(r.Stdout, "data.unchanged").Bool(), "no fabricated CLI no-op proof")
	}
	require.Equal(t, int32(2), writes.Load(), "explicit replay is two requests; idempotency belongs to backend")
	r := f.Run("mindnotes", "nodes", "list", "--mindnote-id", "mind_fixture", "--as", "user")
	success(t, r)
	require.JSONEq(t, gjson.Get(tc.body, "nodes").Raw, gjson.Get(r.Stdout, "data.nodes").Raw)
	require.Equal(t, int32(2), writes.Load(), "readback must not write")
}

func TestApprovalPageFailureDoesNotClaimComplete(t *testing.T) {
	for _, operation := range []struct {
		resource, method, path, collection string
		extra                              []string
	}{
		{"instances", "initiated", "/open-apis/approval/v4/instances/initiated", "instances", nil},
		{"tasks", "query", "/open-apis/approval/v4/tasks", "tasks", []string{"--topic", "1"}},
	} {
		t.Run(operation.resource, func(t *testing.T) {
			for _, mode := range []string{"scope", "lost_response"} {
				t.Run(mode, func(t *testing.T) {
					var pages atomic.Int32
					f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						require.Equal(t, operation.path, r.URL.Path)
						if pages.Add(1) == 1 {
							reply(w, `{"code":0,"data":{"`+operation.collection+`":[ {"instance_code":"first"}],"has_more":true,"page_token":"next"}}`)
						} else {
							require.Equal(t, "next", r.URL.Query().Get("page_token"))
							if mode == "scope" {
								reply(w, `{"code":99991679,"msg":"synthetic permission revoked"}`)
							} else {
								c, _, err := w.(http.Hijacker).Hijack()
								require.NoError(t, err)
								_ = c.Close()
							}
						}
					}))
					path := filepath.Join(f.WorkDir, "incomplete.json")
					args := []string{"approval", operation.resource, operation.method, "--as", "user", "--page-all", "--page-delay", "0", "--save-result", path}
					r := f.Run(append(args, operation.extra...)...)
					require.NotZero(t, r.ExitCode, "stdout=%s stderr=%s", r.Stdout, r.Stderr)
					require.False(t, gjson.Get(r.Stdout, "ok").Bool())
					require.False(t, gjson.Get(r.Stdout, "artifact.complete").Bool())
					if mode == "scope" {
						require.Equal(t, int32(2), pages.Load())
						require.Contains(t, r.Stderr, "99991679")
					} else {
						require.Equal(t, int32(3), pages.Load())
					}
					_, err := os.Stat(path)
					require.True(t, os.IsNotExist(err), "failure must not save a successful artifact")
				})
			}
		})
	}
}

func TestApprovalCreateActionReadbackPreservesFormAndHistory(t *testing.T) {
	for _, action := range []struct{ command, path, status, event string }{
		{"approval tasks approve", "/open-apis/approval/v4/tasks/pass", "APPROVED", "PASS"},
		{"approval tasks reject", "/open-apis/approval/v4/tasks/refuse", "REJECTED", "REJECT"},
		{"approval instances cancel", "/open-apis/approval/v4/instances/recall", "CANCELED", "CANCEL"},
	} {
		t.Run(action.status, func(t *testing.T) {
			var instance map[string]any
			var mutations atomic.Int32
			form := `[{"id":"amount","type":"amount","value":"001.00"},{"id":"detail","type":"textarea","value":"多行\n  保留  "}]`
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/open-apis/approval/v4/instances/initiate":
					require.Nil(t, instance)
					var request map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
					require.Equal(t, form, request["form"])
					require.Equal(t, "business-flow-token", request["uuid"])
					mutations.Add(1)
					instance = map[string]any{"instance_code": "instance_owned", "definition_code": "definition_fixture", "definition_name": "测试 审批", "status": "PENDING", "form": request["form"], "user_id": "ou_synthetic", "department_id": "dept_synthetic", "start_time": "001788775200000", "end_time": "0", "serial_number": "0000123", "reverted": false, "tasks": []any{map[string]any{"id": "task_owned", "status": "PENDING", "user_id": "ou_synthetic", "type": "SEQUENTIAL", "start_time": "1788775200000"}}, "operation_records": []any{}, "comments": []any{map[string]any{"id": "comment_1", "user_id": "ou_synthetic", "comment": "", "create_time": "1788775200000", "files": []any{map[string]any{"type": "attachment", "url": "https://example.invalid/file", "file_size": 9007199254740993, "title": "附件"}}}}}
					reply(w, `{"code":0,"data":{"instance_code":"instance_owned","instance_link":"https://example.invalid/approval/instance_owned"}}`)
				case action.path:
					require.NotNil(t, instance)
					var request map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
					require.Equal(t, "instance_owned", request["instance_code"])
					if action.event != "CANCEL" {
						require.Equal(t, "task_owned", request["task_id"])
					}
					mutations.Add(1)
					instance["status"] = action.status
					instance["operation_records"] = []any{map[string]any{"type": action.event, "create_time": "1788775300000", "task_id": "task_owned", "comment": request["comment"]}}
					reply(w, `{"code":0,"data":{}}`)
				case "/open-apis/approval/v4/instances/detail":
					require.Equal(t, "instance_owned", r.URL.Query().Get("instance_code"))
					raw, err := json.Marshal(map[string]any{"code": 0, "data": instance})
					require.NoError(t, err)
					reply(w, string(raw))
				default:
					t.Errorf("unexpected request %s", r.URL.Path)
					http.Error(w, "unexpected", 500)
				}
			}))
			payload, err := json.Marshal(map[string]any{"approval_code": "definition_fixture", "uuid": "business-flow-token", "form": form})
			require.NoError(t, err)
			created := f.Run("approval", "instances", "create", "--as", "user", "--yes", "--data", string(payload))
			success(t, created)
			id := gjson.Get(created.Stdout, "data.instance_code").String()
			require.Equal(t, "instance_owned", id)
			actionBody, err := json.Marshal(map[string]any{"instance_code": id, "task_id": "task_owned", "comment": "  完成 👋  "})
			require.NoError(t, err)
			actionArgs := append(strings.Fields(action.command), "--as", "user", "--yes", "--data", string(actionBody))
			success(t, f.Run(actionArgs...))
			require.Equal(t, action.status, instance["status"], "independently observed fixture state")
			read := f.Run("approval", "instances", "get", "--as", "user", "--instance-code", id)
			success(t, read)
			require.Equal(t, action.status, gjson.Get(read.Stdout, "data.status").String())
			require.Equal(t, form, gjson.Get(read.Stdout, "data.form").String())
			require.Equal(t, action.event, gjson.Get(read.Stdout, "data.operation_records.0.type").String())
			require.Equal(t, "  完成 👋  ", gjson.Get(read.Stdout, "data.operation_records.0.comment").String())
			require.Equal(t, "9007199254740993", gjson.Get(read.Stdout, "data.comments.0.files.0.file_size").Raw)
			require.Equal(t, "0000123", gjson.Get(read.Stdout, "data.serial_number").String())
			require.Equal(t, int32(2), mutations.Load(), "readback does not mutate")
		})
	}
}

func TestMindnoteCreatedIDCompletenessAndUpdateUncertainty(t *testing.T) {
	creates := `{"client_token":"create-fixture","nodes":[{"parent_id":"root_fixture","texts":[{"element_type":"text","text":{"content":"一"}}]},{"parent_id":"root_fixture","texts":[{"element_type":"text","text":{"content":"二"}}]}]}`
	for _, tc := range []struct {
		name, body, response string
		applied              bool
	}{
		{"complete creates", creates, `{"ids":["created_one","created_two"],"client_token":"create-fixture"}`, true},
		{"missing IDs", creates, `{"client_token":"create-fixture"}`, false},
		{"short IDs", creates, `{"ids":["created_one"]}`, false},
		{"duplicate IDs", creates, `{"ids":["created_one","created_one"]}`, false},
		{"empty ID", creates, `{"ids":["created_one",""]}`, false},
		{"extra IDs", creates, `{"ids":["created_one","created_two","unrelated"]}`, false},
		{"wrong ID type", creates, `{"ids":["created_one",123]}`, false},
		{"update matching ID", `{"nodes":[{"node_id":"node_existing","finish":false}]}`, `{"ids":["node_existing"]}`, false},
		{"update unrelated ID", `{"nodes":[{"node_id":"node_existing","finish":false}]}`, `{"ids":["unrelated"]}`, false},
		{"mixed creates updates", `{"nodes":[{"node_id":"node_existing","finish":false},{"parent_id":"root_fixture"}]}`, `{"ids":["node_existing","created_one"]}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var writes atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "POST", r.Method)
				require.Equal(t, "/open-apis/mindnote/v1/mindnotes/mind_fixture/nodes", r.URL.Path)
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				require.JSONEq(t, tc.body, string(body))
				writes.Add(1)
				reply(w, `{"code":0,"data":`+tc.response+`}`)
			}))
			r := f.Run("mindnotes", "nodes", "create", "--mindnote-id", "mind_fixture", "--as", "user", "--data", tc.body)
			if tc.applied {
				success(t, r)
				outcome(t, r, "confirmed_applied")
			} else {
				unknown(t, r)
			}
			require.JSONEq(t, tc.response, gjson.Get(r.Stdout, "data").Raw, "retain full observed evidence")
			require.Equal(t, int32(1), writes.Load(), "never infer safe duplicate replay")
		})
	}
}

func TestApprovalMissingOrRepeatedContinuationFails(t *testing.T) {
	for _, operation := range []struct {
		resource, method, path, collection string
		extra                              []string
	}{
		{"instances", "initiated", "/open-apis/approval/v4/instances/initiated", "instances", nil},
		{"tasks", "query", "/open-apis/approval/v4/tasks", "tasks", []string{"--topic", "1"}},
	} {
		t.Run(operation.resource, func(t *testing.T) {
			for _, mode := range []string{"missing", "repeated"} {
				t.Run(mode, func(t *testing.T) {
					var pages atomic.Int32
					f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						require.Equal(t, operation.path, r.URL.Path)
						n := pages.Add(1)
						require.LessOrEqual(t, n, int32(2), "must stop a cyclic cursor before replaying again")
						token := ""
						if mode == "repeated" {
							token = `,"page_token":"cycle"`
						}
						if n == 2 {
							require.Equal(t, "cycle", r.URL.Query().Get("page_token"))
						}
						reply(w, `{"code":0,"data":{"`+operation.collection+`":[{"instance_code":"first"}],"has_more":true`+token+`}}`)
					}))
					path := filepath.Join(f.WorkDir, "incomplete.json")
					args := []string{"approval", operation.resource, operation.method, "--as", "user", "--page-all", "--page-delay", "0", "--save-result", path}
					r := f.Run(append(args, operation.extra...)...)
					require.Equal(t, 5, r.ExitCode, "stdout=%s stderr=%s", r.Stdout, r.Stderr)
					require.Contains(t, r.Stderr, "invalid_response")
					require.Empty(t, r.Stdout)
					wantPages := int32(1)
					if mode == "repeated" {
						wantPages = 2
					}
					require.Equal(t, wantPages, pages.Load())
					_, err := os.Stat(path)
					require.True(t, os.IsNotExist(err))
				})
			}
		})
	}
}
