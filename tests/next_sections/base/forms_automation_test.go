// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package baseprocess

func formScenarios() []scenario {
	fp := tp + "/forms/frm_fixture"
	form := func(command string, flags ...string) []string {
		return table(command, append([]string{"--form-id", "frm_fixture"}, flags...)...)
	}
	return []scenario{
		one(table("form-create", "--name", "问卷", "--description", "原文"), ex("POST", tp+"/forms", "", `{"name":"问卷","description":"原文"}`, `{"id":"frm_fixture","name":"问卷"}`), `{"id":"frm_fixture","name":"问卷"}`),
		one(form("form-get"), ex("GET", fp, "", "", `{"id":"frm_fixture","name":"问卷"}`), `{"id":"frm_fixture","name":"问卷"}`),
		one(form("form-update", "--name", "新问卷"), ex("PATCH", fp, "", `{"name":"新问卷"}`, `{"id":"frm_fixture","name":"新问卷"}`), `{"id":"frm_fixture","name":"新问卷"}`),
		one(form("form-delete", "--yes"), ex("DELETE", fp, "", "", `{}`), `{"deleted":true}`),
		{"form-list-cursor-pages", table("form-list", "--page-size", "1"), []exchange{ex("GET", tp+"/forms", "page_size=1", "", `{"forms":[{"id":"frm_a"}],"has_more":true,"page_token":"next"}`), ex("GET", tp+"/forms", "page_size=1&page_token=next", "", `{"forms":[{"id":"frm_b"}],"has_more":false}`)}, `{"forms":[{"id":"frm_a"},{"id":"frm_b"}],"total":2}`},
		one(form("form-questions-list"), ex("GET", fp+"/questions", "", "", `{"questions":[{"id":"fld_question","title":"姓名","required":true}],"total":1}`), `{"questions":[{"id":"fld_question","title":"姓名","required":true}],"total":1}`),
		one(form("form-questions-create", "--questions", `[{"title":"姓名","type":"text","required":true}]`), ex("POST", fp+"/questions", "", `{"questions":[{"title":"姓名","type":"text","required":true}]}`, `{"questions":[{"id":"fld_question","title":"姓名"}]}`), `{"questions":[{"id":"fld_question","title":"姓名"}]}`),
		one(form("form-questions-update", "--questions", `[{"id":"fld_question","title":"别名","description":"","required":false,"visible_rule":null}]`), ex("PATCH", fp+"/questions", "", `{"questions":[{"id":"fld_question","title":"别名","description":"","required":false,"visible_rule":null}]}`, `{"items":[{"id":"fld_question","title":"别名","required":false}]}`), `{"questions":[{"id":"fld_question","title":"别名","required":false}]}`),
		one(form("form-questions-delete", "--question-ids", `["fld_question"]`, "--keep-field", "--yes"), ex("DELETE", fp+"/questions", "", `{"question_ids":["fld_question"],"keep_field":true}`, `{}`), `{"deleted":true,"question_ids":["fld_question"],"keep_field":true}`),
		one(args("form-detail", "--share-token", "shr_fixture"), ex("POST", "/open-apis/base/v3/bases/tables/forms/detail", "", `{"share_token":"shr_fixture"}`, `{"id":"frm_fixture","questions":[{"id":"fld_question"}]}`), `{"id":"frm_fixture","questions":[{"id":"fld_question"}]}`),
		one(args("form-submit", "--share-token", "shr_fixture", "--json", `{"fields":{"姓名":"合成用户","评分":5}}`, "--yes"), ex("POST", "/open-apis/base/v3/bases/tables/forms/submit", "", `{"share_token":"shr_fixture","content":{"姓名":"合成用户","评分":5}}`, `{"record_id":"rec_submit"}`), `{"record_id":"rec_submit"}`),
		one(form("form-share-get"), ex("GET", fp+"/share", "", "", `{"enabled":false,"settings":{"require_login":true}}`), `{"enabled":false,"settings":{"require_login":true}}`),
		one(form("form-share-update", "--allow-anonymous=false"), ex("PATCH", fp+"/share", "", `{"settings":{"allow_anonymous":false}}`, `{"enabled":false,"settings":{"allow_anonymous":false}}`), `{"enabled":false,"settings":{"allow_anonymous":false}}`),
	}
}

func automationScenarios() []scenario {
	s := []scenario{
		one(scoped("role-create", "--json", `{"name":"Synthetic reviewer","role_type":"custom_role"}`), ex("POST", bp+"/roles", "", `{"name":"Synthetic reviewer","role_type":"custom_role"}`, `{"role_id":"rol_fixture","name":"Synthetic reviewer"}`), `{"role_id":"rol_fixture","name":"Synthetic reviewer"}`),
		one(scoped("role-get", "--role-id", "rol_fixture"), ex("GET", bp+"/roles/rol_fixture", "", "", `"{\"role_id\":\"rol_fixture\",\"name\":\"Synthetic reviewer\"}"`), `{"role_id":"rol_fixture","name":"Synthetic reviewer"}`),
		one(scoped("role-list"), ex("GET", bp+"/roles", "", "", `{"roles":[{"role_id":"rol_fixture"}]}`), `{"roles":[{"role_id":"rol_fixture"}]}`),
		one(scoped("role-update", "--role-id", "rol_fixture", "--json", `{"name":"Updated reviewer"}`, "--yes"), ex("PUT", bp+"/roles/rol_fixture", "", `{"name":"Updated reviewer"}`, `{"role_id":"rol_fixture","name":"Updated reviewer"}`), `{"role_id":"rol_fixture","name":"Updated reviewer"}`),
		one(scoped("role-delete", "--role-id", "rol_fixture", "--yes"), ex("DELETE", bp+"/roles/rol_fixture", "", "{}", `null`), `{"success":true}`),
		one(scoped("advperm-enable"), ex("PUT", bp+"/advperm/enable", "enable=true", "", `null`), `{"success":true}`),
		one(scoped("advperm-disable", "--yes"), ex("PUT", bp+"/advperm/enable", "enable=false", "", `null`), `{"success":true}`),
		{"workflow-list-cursor-pages", scoped("workflow-list", "--page-size", "1", "--status", "disabled"), []exchange{ex("POST", bp+"/workflows/list", "", `{"page_size":1,"status":"disabled"}`, `{"items":[{"workflow_id":"wkf_a"}],"has_more":true,"page_token":"next"}`), ex("POST", bp+"/workflows/list", "", `{"page_size":1,"status":"disabled","page_token":"next"}`, `{"items":[{"workflow_id":"wkf_b"}],"has_more":false}`)}, `{"items":[{"workflow_id":"wkf_a"},{"workflow_id":"wkf_b"}],"total":2}`},
		one(scoped("workflow-get", "--workflow-id", "wkf_fixture", "--user-id-type", "union_id"), ex("GET", bp+"/workflows/wkf_fixture", "user_id_type=union_id", "", `{"workflow_id":"wkf_fixture","status":"disabled"}`), `{"workflow_id":"wkf_fixture","status":"disabled"}`),
		one(scoped("workflow-create", "--json", `{"name":"Synthetic disabled workflow","steps":[]}`), ex("POST", bp+"/workflows", "", `{"name":"Synthetic disabled workflow","steps":[]}`, `{"workflow_id":"wkf_fixture","status":"disabled"}`), `{"workflow_id":"wkf_fixture","status":"disabled"}`),
		one(scoped("workflow-update", "--workflow-id", "wkf_fixture", "--json", `{"name":"Updated workflow","steps":[]}`), ex("PUT", bp+"/workflows/wkf_fixture", "", `{"name":"Updated workflow","steps":[]}`, `{"workflow_id":"wkf_fixture","name":"Updated workflow"}`), `{"workflow_id":"wkf_fixture","name":"Updated workflow"}`),
		one(scoped("workflow-enable", "--workflow-id", "wkf_fixture"), ex("PATCH", bp+"/workflows/wkf_fixture/enable", "", `{}`, `{"workflow_id":"wkf_fixture","status":"enabled"}`), `{"status":"enabled"}`),
		one(scoped("workflow-disable", "--workflow-id", "wkf_fixture"), ex("PATCH", bp+"/workflows/wkf_fixture/disable", "", `{}`, `{"workflow_id":"wkf_fixture","status":"disabled"}`), `{"status":"disabled"}`),
		one(scoped("data-query", "--dsl", `{"measures":[{"field":"fld_amount","aggregate":"sum"}],"table_id":"tbl_fixture"}`), ex("POST", bp+"/data/query", "", `{"measures":[{"field":"fld_amount","aggregate":"sum"}],"table_id":"tbl_fixture"}`, `{"data":[[12.5]],"fields":["sum"]}`), `{"data":[[12.5]],"fields":["sum"]}`),
		one(args("template-categories"), ex("GET", "/open-apis/base/v3/bases/templates/category", "", "", `{"categories":[{"key":"synthetic","name":"Synthetic"}]}`), `{"categories":[{"key":"synthetic"}]}`),
		one(args("template-list", "--category-key", "synthetic", "--limit", "2", "--offset", "cursor"), ex("GET", "/open-apis/base/v3/bases/templates", "category_key=synthetic&limit=2&offset=cursor", "", `{"templates":[{"id":"tpl_fixture"}],"offset":"next"}`), `{"templates":[{"id":"tpl_fixture"}],"offset":"next"}`),
		one(args("template-search", "--keyword", "计划", "--limit", "2", "--offset", "cursor"), ex("GET", "/open-apis/base/v3/bases/templates/search", "keyword=计划&limit=2&offset=cursor", "", `{"templates":[{"id":"tpl_fixture","name":"计划"}]}`), `{"templates":[{"id":"tpl_fixture","name":"计划"}]}`),
	}
	for _, operation := range []string{"bind", "get", "unbind"} {
		method, body := "PUT", `{"workflow_id":""}`
		flags := []string{"--field-id", "按钮"}
		if operation == "bind" {
			body = `{"workflow_id":"wkf_fixture"}`
			flags = append(flags, "--workflow-id", "wkf_fixture")
		}
		if operation == "get" {
			method, body = "GET", ""
		}
		s = append(s, scenario{"button-rule-" + operation, table("button-rule-"+operation, flags...), []exchange{ex("GET", tp+"/fields/按钮", "", "", `{"id":"fld_button","type":"button"}`), ex(method, tp+"/fields/fld_button/button_rule", "", body, `{"workflow_id":"wkf_fixture"}`)}, `{"workflow_id":"wkf_fixture"}`})
	}
	return s
}
