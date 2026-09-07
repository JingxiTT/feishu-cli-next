// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package baseprocess

func appScenarios() []scenario {
	dp := bp + "/dashboards/dsh_fixture"
	pp := ap + "/pages/pag_fixture"
	return []scenario{
		one(scoped("dashboard-list", "--page-size", "2", "--page-token", "next"), ex("GET", bp+"/dashboards", "page_size=2&page_token=next", "", `{"items":[{"id":"dsh_fixture"}],"has_more":true,"page_token":"later"}`), `{"items":[{"id":"dsh_fixture"}],"has_more":true,"page_token":"later"}`),
		one(dash("dashboard-get"), ex("GET", dp, "", "", `{"id":"dsh_fixture","name":"看板"}`), `{"dashboard":{"id":"dsh_fixture","name":"看板"}}`),
		one(scoped("dashboard-create", "--name", "看板", "--theme-style", "cloudBlue"), ex("POST", bp+"/dashboards", "", `{"name":"看板","theme":{"theme_style":"cloudBlue"}}`, `{"id":"dsh_fixture"}`), `{"created":true,"dashboard":{"id":"dsh_fixture"}}`),
		one(dash("dashboard-update", "--name", "新看板"), ex("PATCH", dp, "", `{"name":"新看板"}`, `{"id":"dsh_fixture","name":"新看板"}`), `{"updated":true,"dashboard":{"name":"新看板"}}`),
		one(dash("dashboard-delete", "--yes"), ex("DELETE", dp, "", "", `{}`), `{"deleted":true,"dashboard_id":"dsh_fixture"}`),
		one(dash("dashboard-arrange", "--user-id-type", "union_id"), ex("POST", dp+"/arrange", "user_id_type=union_id", `{}`, `{"block_ids":["blk_fixture"]}`), `{"arranged":true,"block_ids":["blk_fixture"]}`),
		one(dash("dashboard-share-get"), ex("GET", dp+"/share", "", "", `{"enabled":false,"settings":{"show_source":false}}`), `{"enabled":false,"settings":{"show_source":false}}`),
		one(dash("dashboard-share-update", "--show-source=false"), ex("PATCH", dp+"/share", "", `{"settings":{"show_source":false}}`, `{"enabled":false,"settings":{"show_source":false}}`), `{"enabled":false,"settings":{"show_source":false}}`),
		one(dash("dashboard-block-list", "--page-size", "2", "--page-token", "next"), ex("GET", dp+"/blocks", "page_size=2&page_token=next", "", `{"items":[{"id":"blk_fixture","type":"text"}],"page_token":"later","has_more":true}`), `{"items":[{"id":"blk_fixture","type":"text"}],"page_token":"later","has_more":true}`),
		one(dash("dashboard-block-get", "--block-id", "blk_fixture", "--user-id-type", "union_id"), ex("GET", dp+"/blocks/blk_fixture", "user_id_type=union_id", "", `{"id":"blk_fixture","data_config":{"text":"# 原文"}}`), `{"block":{"id":"blk_fixture","data_config":{"text":"# 原文"}}}`),
		one(scoped("dashboard-block-get-data", "--block-id", "cht_fixture"), ex("GET", bp+"/dashboards/blocks/cht_fixture/data", "", "", `{"dimensions":["状态"],"data":[["完成",3]]}`), `{"dimensions":["状态"],"data":[["完成",3]]}`),
		one(dash("dashboard-block-create", "--name", "说明", "--type", "text", "--data-config", `{"text":"# 原文\n内容"}`, "--position", `{"x":0,"y":1,"w":6,"h":4}`), ex("POST", dp+"/blocks", "", `{"name":"说明","type":"text","data_config":{"text":"# 原文\n内容"},"position":{"x":0,"y":1,"w":6,"h":4}}`, `{"id":"blk_fixture"}`), `{"created":true,"block":{"id":"blk_fixture"}}`),
		one(dash("dashboard-block-update", "--block-id", "blk_fixture", "--name", "移动", "--position", `{"x":6,"y":1,"w":6,"h":4}`), ex("PATCH", dp+"/blocks/blk_fixture", "", `{"name":"移动","position":{"x":6,"y":1,"w":6,"h":4}}`, `{"id":"blk_fixture","position":{"x":6,"y":1,"w":6,"h":4}}`), `{"updated":true,"block":{"position":{"x":6,"y":1,"w":6,"h":4}}}`),
		one(dash("dashboard-block-delete", "--block-id", "blk_fixture", "--yes"), ex("DELETE", dp+"/blocks/blk_fixture", "", "", `{}`), `{"deleted":true,"block_id":"blk_fixture"}`),
		one(args("workspace-create", "--name", "合成工作区"), ex("POST", "/open-apis/base/v3/workspaces", "", `{"name":"合成工作区"}`, `{"workspace_token":"wsp_fixture"}`), `{"workspace":{"workspace_token":"wsp_fixture"},"created":true}`),
		one(args("workspace-entity-list", "--workspace-token", "wsp_fixture", "--type", "baseapp", "--page-size", "2", "--page-token", "next"), ex("GET", "/open-apis/base/v3/workspaces/wsp_fixture/entities", "entity_type=baseapp&page_size=2&page_token=next", "", `{"items":[{"entity_token":"app_fixture","entity_type":"baseapp"}],"has_more":false}`), `{"items":[{"entity_token":"app_fixture","entity_type":"baseapp"}]}`),
		one(args("workspace-move-in", "--workspace-token", "wsp_fixture", "--entity-token", "bas_fixture"), ex("POST", "/open-apis/base/v3/workspaces/wsp_fixture/move_in", "", `{"entity_token":"bas_fixture"}`, `{"entity_token":"bas_fixture"}`), `{"entity_token":"bas_fixture"}`),
		one(args("app-create", "--name", "应用", "--workspace-token", "wsp_fixture", "--theme-style", "cloudBlue"), ex("POST", "/open-apis/base/v3/base_apps", "", `{"name":"应用","workspace_token":"wsp_fixture","theme":{"theme_style":"cloudBlue"}}`, `{"app_token":"app_fixture"}`), `{"created":true,"app":{"app_token":"app_fixture"},"workspace_token":"wsp_fixture"}`),
		one(app("app-get"), ex("GET", ap, "", "", `{"app_token":"app_fixture","workspace_token":"wsp_fixture"}`), `{"app_token":"app_fixture","workspace_token":"wsp_fixture"}`),
		one(app("app-page-list", "--page-size", "2", "--page-token", "next"), ex("GET", ap+"/pages", "page_size=2&page_token=next", "", `{"items":[{"id":"pag_fixture"}],"has_more":true,"page_token":"later"}`), `{"items":[{"id":"pag_fixture"}],"has_more":true,"page_token":"later"}`),
		one(page("app-page-get"), ex("GET", pp, "", "", `{"id":"pag_fixture","name":"页面"}`), `{"page":{"id":"pag_fixture","name":"页面"}}`),
		{"app-page-create-with-uniqueness-read", app("app-page-create", "--name", "页面"), []exchange{ex("GET", ap+"/pages", "page_size=100", "", `{"items":[{"id":"pag_old","name":"Existing"}],"has_more":false}`), ex("POST", ap+"/pages", "", `{"name":"页面"}`, `{"id":"pag_fixture"}`)}, `{"created":true,"page":{"id":"pag_fixture"}}`},
		{"app-page-update-with-uniqueness-read", page("app-page-update", "--name", "新页面"), []exchange{ex("GET", ap+"/pages", "page_size=100", "", `{"items":[{"id":"pag_fixture","name":"页面"}],"has_more":false}`), ex("PATCH", pp, "", `{"name":"新页面"}`, `{"id":"pag_fixture","name":"新页面"}`)}, `{"updated":true,"page":{"name":"新页面"}}`},
		one(page("app-page-delete", "--yes"), ex("DELETE", pp, "", "", `{}`), `{"deleted":true,"page_id":"pag_fixture"}`),
		one(page("app-block-list", "--page-size", "2", "--page-token", "next"), ex("GET", pp+"/blocks", "page_size=2&page_token=next", "", `{"items":[{"id":"blk_fixture","type":"text"}],"has_more":false}`), `{"items":[{"id":"blk_fixture","type":"text"}]}`),
		one(page("app-block-get", "--block-id", "blk_fixture"), ex("GET", pp+"/blocks/blk_fixture", "", "", `{"id":"blk_fixture","type":"text","data_config":{"text":"# 原文"}}`), `{"block":{"id":"blk_fixture","data_config":{"text":"# 原文"}}}`),
		one(app("app-block-get-data", "--base-token", "bas_fixture", "--block-id", "cht_fixture"), ex("GET", ap+"/blocks/cht_fixture/data", "base_token=bas_fixture", "", `{"data":[[3]],"dimensions":["Count"]}`), `{"data":[[3]],"dimensions":["Count"]}`),
		{"app-block-create-with-uniqueness-read", page("app-block-create", "--name", "说明", "--type", "text", "--data-config", `{"text":"# 原文"}`), []exchange{ex("GET", pp+"/blocks", "page_size=100", "", `{"items":[],"has_more":false}`), ex("POST", pp+"/blocks", "", `{"name":"说明","type":"text","data_config":{"text":"# 原文"}}`, `{"id":"blk_fixture"}`)}, `{"created":true,"block":{"id":"blk_fixture"}}`},
		{"app-block-update-with-current-type-read", page("app-block-update", "--block-id", "blk_fixture", "--data-config", `{"text":"# 新内容"}`), []exchange{ex("GET", pp+"/blocks/blk_fixture", "", "", `{"id":"blk_fixture","type":"text"}`), ex("PATCH", pp+"/blocks/blk_fixture", "", `{"data_config":{"text":"# 新内容"}}`, `{"id":"blk_fixture","data_config":{"text":"# 新内容"}}`)}, `{"updated":true,"block":{"data_config":{"text":"# 新内容"}}}`},
	}
}

func resolveScenarios() []scenario {
	return []scenario{
		{"url-resolve-local", args("url-resolve", "--url", "https://example.invalid/base/bas_fixture"), nil, `{"base_token":"bas_fixture","resource_type":"bitable"}`},
		{"url-resolve-wiki", args("url-resolve", "--url", "https://example.invalid/wiki/wik_fixture"), []exchange{ex("GET", "/open-apis/wiki/v2/spaces/get_node", "token=wik_fixture", "", `{"node":{"obj_token":"bas_fixture","obj_type":"bitable"}}`)}, `{"base_token":"bas_fixture"}`},
		one(args("title-resolve", "--title", "计划"), ex("POST", "/open-apis/search/v2/doc_wiki/search", "", `{"query":"计划","page_size":5,"doc_filter":{"doc_types":["BITABLE"]},"wiki_filter":{"doc_types":["BITABLE"]}}`, `{"res_units":[{"title_highlighted":"<h>计划</h>","result_meta":{"doc_types":"BITABLE","token":"bas_fixture","url":"https://example.invalid/base/bas_fixture"}}]}`), `{"base_token":"bas_fixture","title":"计划"}`),
	}
}
