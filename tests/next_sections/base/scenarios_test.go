// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package baseprocess

func scenarios() []scenario {
	s := []scenario{
		one(scoped("base-get"), ex("GET", bp, "", "", `{"base_token":"bas_fixture","name":"私有夹具"}`), `{"base":{"base_token":"bas_fixture","name":"私有夹具"}}`),
		one(args("base-create", "--name", "私有夹具"), ex("POST", "/open-apis/base/v3/bases", "", `{"name":"私有夹具"}`, `{"base_token":"bas_created"}`), `{"created":true,"base":{"base_token":"bas_created"}}`),
		one(scoped("base-copy", "--name", "副本", "--without-content"), ex("POST", bp+"/copy", "", `{"name":"副本","without_content":true}`, `{"base_token":"bas_copy"}`), `{"copied":true,"base":{"base_token":"bas_copy"}}`),
		one(scoped("base-block-list", "--parent-id", "blk_parent", "--type", "table"), ex("POST", bp+"/blocks/list", "", `{"parent_id":"blk_parent"}`, `{"blocks":[{"id":"tbl_fixture","type":"table"},{"id":"dsh_fixture","type":"dashboard"}]}`), `{"blocks":[{"id":"tbl_fixture","type":"table"}],"total":1}`),
		one(scoped("base-block-create", "--name", "Folder", "--type", "folder", "--parent-id", "blk_parent"), ex("POST", bp+"/blocks", "", `{"name":"Folder","type":"folder","parent_id":"blk_parent"}`, `{"id":"blk_new"}`), `{"created":true,"block":{"id":"blk_new"}}`),
		one(scoped("base-block-move", "--block-id", "blk_fixture", "--after-id", "blk_after"), ex("POST", bp+"/blocks/blk_fixture/move", "", `{"parent_id":null,"after_id":"blk_after"}`, `{"id":"blk_fixture"}`), `{"moved":true}`),
		one(scoped("base-block-rename", "--block-id", "blk_fixture", "--name", "重命名"), ex("POST", bp+"/blocks/blk_fixture/rename", "", `{"name":"重命名"}`, `{"id":"blk_fixture","name":"重命名"}`), `{"renamed":true,"block":{"name":"重命名"}}`),
		one(scoped("base-block-delete", "--block-id", "blk_fixture", "--yes"), ex("DELETE", bp+"/blocks/blk_fixture", "", "", `{}`), `{"deleted":true}`),
		one(scoped("table-list", "--offset", "2", "--limit", "3"), ex("GET", bp+"/tables", "offset=2&limit=3", "", `{"tables":[{"id":"tbl_fixture","name":"Data"}],"total":4}`), `{"tables":[{"id":"tbl_fixture"}],"total":4}`),
		{"table-get-with-schema", table("table-get"), []exchange{ex("GET", tp, "", "", `{"id":"tbl_fixture","name":"Data"}`), ex("GET", tp+"/fields", "offset=0&limit=100", "", `{"fields":[{"id":"fld_fixture","type":"text"}],"total":1}`), ex("GET", tp+"/views", "offset=0&limit=100", "", `{"views":[{"id":"vew_fixture","type":"grid"}],"total":1}`)}, `{"table":{"id":"tbl_fixture"},"fields":[{"id":"fld_fixture","type":"text"}],"views":[{"id":"vew_fixture","type":"grid"}]}`},
		{"table-create-with-view", scoped("table-create", "--name", "Data", "--fields", `[{"name":"标题","type":"text"}]`, "--view", `{"name":"Grid","type":"grid"}`), []exchange{ex("POST", bp+"/tables", "", `{"name":"Data","fields":[{"name":"标题","type":"text"}]}`, `{"id":"tbl_fixture","fields":[{"id":"fld_fixture","name":"标题"}]}`), ex("POST", tp+"/views", "", `{"name":"Grid","type":"grid"}`, `{"id":"vew_fixture"}`)}, `{"table":{"id":"tbl_fixture"},"fields":[{"id":"fld_fixture"}],"views":[{"id":"vew_fixture"}]}`},
		one(table("table-update", "--name", "新表"), ex("PATCH", tp, "", `{"name":"新表"}`, `{"id":"tbl_fixture","name":"新表"}`), `{"updated":true,"table":{"name":"新表"}}`),
		one(table("table-delete", "--yes"), ex("DELETE", tp, "", "", `{}`), `{"deleted":true,"table_id":"tbl_fixture"}`),
		one(table("table-copy", "--name", "Copy", "--range", "schema"), ex("POST", tp+"/copy", "", `{"name":"Copy","range":"schema"}`, `{"table":{"id":"tbl_copy","name":"Copy"},"state":"success"}`), `{"completed":true,"state":"success","table":{"id":"tbl_copy"}}`),
		one(scoped("table-copy-status", "--task-id", "task_fixture"), ex("POST", bp+"/copy_table_state", "", `{"task_id":"task_fixture"}`, `{"table_id":"tbl_copy","state":"process"}`), `{"completed":false,"state":"process","task_id":"task_fixture"}`),
		one(table("field-list", "--offset", "2", "--limit", "3"), ex("GET", tp+"/fields", "offset=2&limit=3", "", `{"fields":[{"id":"fld_fixture","type":"number"}],"total":4}`), `{"fields":[{"id":"fld_fixture","type":"number"}],"total":4}`),
		one(table("field-get", "--field-id", "fld_fixture"), ex("GET", tp+"/fields/fld_fixture", "", "", `{"id":"fld_fixture","type":"text","name":"标题"}`), `{"field":{"id":"fld_fixture","name":"标题"}}`),
		one(table("field-create", "--json", `{"name":"标题","type":"text"}`), ex("POST", tp+"/fields", "", `{"name":"标题","type":"text"}`, `{"id":"fld_fixture","name":"标题","type":"text"}`), `{"created":true,"field":{"id":"fld_fixture","type":"text"}}`),
		one(table("field-update", "--field-id", "fld_fixture", "--json", `{"name":"新标题","type":"text"}`, "--yes"), ex("PUT", tp+"/fields/fld_fixture", "", `{"name":"新标题","type":"text"}`, `{"id":"fld_fixture","name":"新标题"}`), `{"updated":true,"field":{"name":"新标题"}}`),
		one(table("field-delete", "--field-id", "fld_fixture", "--yes"), ex("DELETE", tp+"/fields/fld_fixture", "", "", `{}`), `{"deleted":true}`),
		one(table("field-search-options", "--field-id", "fld_fixture", "--keyword", "进行", "--offset", "2", "--limit", "3"), ex("GET", tp+"/fields/fld_fixture/options", "offset=2&limit=3&query=进行", "", `{"options":[{"id":"opt_fixture","name":"进行中"}],"total":4}`), `{"options":[{"id":"opt_fixture","name":"进行中"}],"total":4}`),
		one(table("field-extension-get", "--field-id", "fld_fixture"), ex("GET", tp+"/fields/fld_fixture/field_extensions", "", "", `{"current_extension":null}`), `{"current_extension":null}`),
		one(table("field-extension-update", "--field-id", "fld_fixture", "--json", "{}", "--yes"), ex("PUT", tp+"/fields/fld_fixture/field_extensions", "", `{}`, `{"current_extension":null}`), `{"current_extension":null}`),
		one(table("field-extension-update-cells", "--field-id", "fld_fixture", "--type", "row", "--record-id", "rec_fixture", "--yes"), ex("POST", tp+"/fields/fld_fixture/field_extensions/update_cells", "", `{"type":"row","record_ids":["rec_fixture"]}`, `{"task_id":"task_cells"}`), `{"task_id":"task_cells"}`),
		one(table("view-list", "--limit", "3", "--offset", "2"), ex("GET", tp+"/views", "offset=2&limit=3", "", `{"views":[{"id":"vew_fixture"}],"total":4}`), `{"views":[{"id":"vew_fixture"}],"total":4}`),
		one(view("view-get"), ex("GET", tp+"/views/vew_fixture", "", "", `{"id":"vew_fixture","type":"grid"}`), `{"view":{"id":"vew_fixture","type":"grid"}}`),
		one(table("view-create", "--json", `{"name":"Grid","type":"grid"}`), ex("POST", tp+"/views", "", `{"name":"Grid","type":"grid"}`, `{"id":"vew_fixture"}`), `{"views":[{"id":"vew_fixture"}]}`),
		one(view("view-delete", "--yes"), ex("DELETE", tp+"/views/vew_fixture", "", "", `{}`), `{"deleted":true}`),
		one(view("view-rename", "--name", "视图"), ex("PATCH", tp+"/views/vew_fixture", "", `{"name":"视图"}`, `{"id":"vew_fixture","name":"视图"}`), `{"view":{"name":"视图"}}`),
		one(table("record-list", "--format", "json", "--limit", "3", "--offset", "2", "--field-id", "标题", "--view-id", "vew_fixture"), ex("GET", tp+"/records", "offset=2&limit=3&field_id=标题&view_id=vew_fixture", "", `{"timezone":"Asia/Shanghai","rev":42,"fields":["标题"],"field_id_list":["fld_title"],"field_type_list":["text"],"record_id_list":["rec_fixture"],"data":[["原文\n第二行"]],"has_more":false}`), `{"timezone":"Asia/Shanghai","rev":42,"fields":["标题"],"field_id_list":["fld_title"],"field_type_list":["text"],"record_id_list":["rec_fixture"],"data":[["原文\n第二行"]],"has_more":false}`),
		one(table("record-get", "--format", "json", "--record-id", "rec_fixture", "--field-id", "标题"), ex("POST", tp+"/records/batch_get", "", `{"record_id_list":["rec_fixture"],"select_fields":["标题"]}`, `{"timezone":"Asia/Shanghai","rev":42,"fields":["标题"],"field_id_list":["fld_title"],"field_type_list":["text"],"record_id_list":["rec_fixture"],"data":[["原文\n第二行"]],"has_more":false}`), `{"timezone":"Asia/Shanghai","rev":42,"fields":["标题"],"field_id_list":["fld_title"],"field_type_list":["text"],"record_id_list":["rec_fixture"],"data":[["原文\n第二行"]],"has_more":false}`),
		one(table("record-search", "--format", "json", "--json", `{"filter":{"logic":"and","conditions":[["标题","==","原文"]]},"limit":3,"offset":2}`), ex("POST", tp+"/records/search", "", `{"filter":{"logic":"and","conditions":[["标题","==","原文"]]},"limit":3,"offset":2}`, `{"timezone":"Asia/Shanghai","rev":42,"fields":["标题"],"field_id_list":["fld_title"],"field_type_list":["text"],"record_id_list":["rec_fixture"],"data":[["原文\n第二行"]],"has_more":false}`), `{"timezone":"Asia/Shanghai","rev":42,"fields":["标题"],"field_id_list":["fld_title"],"field_type_list":["text"],"record_id_list":["rec_fixture"],"data":[["原文\n第二行"]],"has_more":false}`),
		one(table("record-upsert", "--json", `{"标题":"原文","金额":12.5,"清空":null}`), ex("POST", tp+"/records", "", `{"标题":"原文","金额":12.5,"清空":null}`, `{"record_id":"rec_fixture","fields":{"标题":"原文","金额":12.5,"清空":null}}`), `{"created":true,"record":{"record_id":"rec_fixture","fields":{"金额":12.5,"清空":null}}}`),
		{"record-upsert-update", table("record-upsert", "--record-id", "rec_fixture", "--json", `{"标题":"更新"}`), []exchange{ex("PATCH", tp+"/records/rec_fixture", "", `{"标题":"更新"}`, `{"record_id":"rec_fixture","fields":{"标题":"更新"}}`)}, `{"updated":true,"record":{"fields":{"标题":"更新"}}}`},
		one(table("record-batch-create", "--json", `{"create_records":[{"标题":"甲"},{"标题":"乙"}]}`), ex("POST", tp+"/records/batch_create", "", `{"create_records":[{"标题":"甲"},{"标题":"乙"}]}`, `{"record_id_list":["rec_a","rec_b"]}`), `{"record_id_list":["rec_a","rec_b"]}`),
		one(table("record-batch-update", "--json", `{"update_records":{"rec_fixture":{"标题":"改"}}}`), ex("POST", tp+"/records/batch_update", "", `{"update_records":{"rec_fixture":{"标题":"改"}}}`, `{"ignored_fields":[]}`), `{"ignored_fields":[]}`),
		one(table("record-delete", "--record-id", "rec_fixture", "--yes"), ex("POST", tp+"/records/batch_delete", "", `{"record_id_list":["rec_fixture"]}`, `{"record_id_list":["rec_fixture"]}`), `{"record_id_list":["rec_fixture"]}`),
		one(table("record-share-link-create", "--record-id", "rec_fixture", "--record-id", "rec_fixture"), ex("POST", tp+"/records/share_links/batch", "", `{"record_ids":["rec_fixture"]}`, `{"share_links":[{"record_id":"rec_fixture","url":"https://example.invalid/record/share_fixture"}]}`), `{"share_links":[{"record_id":"rec_fixture","url":"https://example.invalid/record/share_fixture"}]}`),
		one(table("record-history-list", "--record-id", "rec_fixture", "--page-size", "2", "--max-version", "9"), ex("GET", bp+"/record_history", "table_id=tbl_fixture&record_id=rec_fixture&page_size=2&max_version=9", "", `{"items":[{"version":8,"record_id":"rec_fixture"}],"has_more":false}`), `{"items":[{"version":8,"record_id":"rec_fixture"}]}`),
	}
	for _, p := range []struct{ flag, path, body string }{
		{"filter", "filter", `{"logic":"and","conditions":[["标题","==","原文"]]}`},
		{"visible-fields", "visible_fields", `{"fields":["fld_fixture"]}`},
		{"group", "group", `{"group_config":[{"field":"fld_fixture","desc":false}]}`},
		{"sort", "sort", `{"sort_config":[{"field":"fld_fixture","desc":true}]}`},
		{"timebar", "timebar", `{"start_field":"fld_start","end_field":"fld_end"}`},
		{"card", "card", `{"cover_field":"fld_cover"}`},
	} {
		path := tp + "/views/vew_fixture/" + p.path
		s = append(s, one(view("view-get-"+p.flag), ex("GET", path, "", "", p.body), `{`+`"`+p.path+`":`+p.body+`}`))
		s = append(s, one(view("view-set-"+p.flag, "--json", p.body), ex("PUT", path, "", p.body, p.body), `{`+`"`+p.path+`":`+p.body+`}`))
	}
	s = append(s, formScenarios()...)
	s = append(s, automationScenarios()...)
	s = append(s, appScenarios()...)
	s = append(s, resolveScenarios()...)
	return s
}
