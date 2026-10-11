---
name: lark-sheets
version: 3.1.8
description: "飞书电子表格：创建和操作电子表格。支持创建表格、管理工作表与行列结构（增删/合并/调整尺寸/隐藏/冻结）、读写单元格（值/公式/样式/批注/单元格图片）、查找替换、多操作批量更新，以及图表、透视表、条件格式、筛选器、迷你图、浮动图片等对象的创建与维护。当用户需要创建电子表格、管理工作表、批量读写或编辑数据、统计汇总与可视化、表格美化、公式计算（含 Excel 公式迁移）、金融/财务建模（DCF、三张表、预算、Sensitivity 等）等任务时使用。若用户是想按名称或关键词搜索云空间（云盘/云存储）里的表格文件，请改用 lark-drive 的 drive +search 先定位资源。当用户给出 doubao.com 的 /sheets/ URL/token 时，也应直接使用本 skill，不要因为域名不是飞书而回退到 WebFetch；路由依据是 URL 路径模式和 token，而不是域名。"
metadata:
  requires:
    bins: ["lark-cli"]
    siblings: ["lark-shared"]
  cliHelp: "lark-cli sheets --help"
---

# sheets

开始前读取一次 [`lark-shared`](../lark-shared/SKILL.md)，遵守认证、身份和高风险规则。复用本任务已读的同版本指导；已有命令/参数指引足够时可直接执行，缺少要求时只读下表对应 reference 或该命令 `--help`。不要按 Excel/Google Sheets/OpenAPI 习惯猜命令或 flag。

**spreadsheet** 是工作簿（顶层容器），**sheet** 是工作表/子表/tab（稳定 `sheet_id`）。对象 reference_id 是各自主键，区别于图片上传句柄 image_uri。找文件名/关键词走 `lark-drive drive +search`；已有 `/sheets/`、`/spreadsheets/`、`/wiki/` URL 按路径/token 路由，含 doubao.com；wiki 自动解析底层表。

## 飞书表格编辑准则（所有编辑生效）

1. **最小改动**：只改授权的格/列；其余值、结构、表名、合并、格式保持。补齐只写空格，已有可疑值只备注；不得擅自删/改名/隐藏/移动原 sheet。中间结果放数据右侧已核实的空区或新 sheet。保留原数字显示格式；仅原格式已截断精度/科学计数法时修显示，不改底层值。
2. **真实写回并验证**：ok 仅表示请求接受。写后按任务回读 `+csv-get` / `+cells-get` / 对象 list，确认首/中/末与边界、无截断/溢出/科学计数法；公式用 `--include formula` 核对落格，筛选/排序核前几行，删除确认已空。回读中的「值(样式)」注记和残留引号不能当原值写回。
3. **读全再写**：先确认表头、真实数据首末行及中间空白/隐藏行，物理 row_count 只是上界，current_region 可能在空行截断或含标题/汇总/签名。坐标用 actual_range / row_indices / col_indices 或 annotated_csv 的 `[row=N]`，不能用数组下标、序号列或数逗号。规则见 [`read data`](references/lark-sheets-read-data.md)。
4. **可推导值用公式**，别静默写本地计算的常量；输入可能空时先判空（IFERROR 拦不住空格被当 0）。聚合起点跳过表头、终点覆盖真实末行，且不能引用自身/间接依赖。公式生成前读 [`formula translation`](references/lark-sheets-formula-translation.md)，落表后用 [`+formula-verify`](references/lark-sheets-formula-verify.md) 诊断；日期/查找/数值转换用 IFERROR，抽查首末各 5 行。同一方案失败 3 次可降级静态值，明确说明原因及不随输入更新。
5. **续写继承原样式**：字体、底色（含交替色）、四边框、对齐、合并、行高列宽都延续。优先 `+dim-insert --inherit-style before/after`；它选择继承侧、不表示插入方向，**行高不继承**，另读相邻 row_height 后补 `+rows-resize`。空白/非相邻区先 `+cells-get --include style`；扩展标题合并到新末列，分组合并先核边界。
6. **写入合批按目的选**：同类操作多区域用该命令复数参数（--ranges/map/--writes）；美化一次 `+styles-put`；只有跨类型、有顺序依赖的链用 `+batch-update`。分组汇总优先原生透视表，图表用原生对象，避免静态图片/假透视表替代可交互结果。
7. **新内容可读、可计算**：新增表头有区分度；用户指定标题、表名、备注、图例逐字保留。新增计算列/汇总行显式设 number_format（比率/百分比/金额按语义并与原列一致）。日期先扫全列确认月日顺序，不逐格猜；长文本换行并给足列宽；图表含标题、轴标签和图例。单位/口径/来源放副标题或字段名，不占原表头/数据格。涉及样式先读 [`visual standards`](references/lark-sheets-visual-standards.md)。
8. **事实与提取保真**：提取同行源列的连续原文片段，多段全列；无固定模式的语义任务逐行判断，别硬套偏移/正则。源内事实缺失留空并按原约定备注，不能编造；外部参数说明来源、单位和不确定项，推算标明未经核验。
9. **逐点验收**：把需求拆成可验证项，批量任务 assert 实际条数等于预期。替换/标注/删除后复查残留（单次有上限），样本含首/中/末；样式也验收。每项“已完成”都能指出产物位置；已知问题、未完成项、静态降级逐项说明，文字不替代产物。

## 场景 → 命令 / Reference

只进入当前操作所需的参考；公式和样式的前置规则仍按上节执行。完整 payload、枚举和条件细节留在参考中。

| 意图 | 命令 | 参考 / 分流 |
|---|---|---|
| 读纯值/CSV | `+csv-get` | [`read data`](references/lark-sheets-read-data.md)；range 可省略读全子表，大表有界分批落盘 |
| 读公式/样式/批注/校验 | `+cells-get --include value,formula,style,comment,data_validation` | 同上；只需值时优先 CSV；合并/行高另查 sheet-info |
| 写纯文本 CSV | `+csv-put --start-cell A1` | [`write cells`](references/lark-sheets-write-cells.md)；--range 别名取左上角；编号 001/日期标签 12.10 会数值化，改 typed 写入 |
| 已有表写带类型数据 | `+table-put --sheets` | 同上；完整 `{sheets:[...]}`，columns/data/dtypes/formats；量值保留 number/date，字面编号用 object；可同传 --styles，缺失 sheet 名会新建 |
| 新工作簿 + typed 数据 | `+workbook-create --sheets` | [`workbook`](references/lark-sheets-workbook.md)；同构协议，可同传 --styles；勿用 --values 把日期/数字变文本 |
| 公式/批注/富文本/单元格图片/精确值 | `+cells-set --range A1 --cells` | [`write cells`](references/lark-sheets-write-cells.md)；cells 恒为二维数组，跨区域/表用 --writes；记录绑定图片用 +cells-set-image |
| 小区域只改样式 | `+cells-set-style` | 同上；**不可用 +cells-set --copy-to-range 刷样式，会覆盖值** |
| 多区域/整表美化 | `+styles-put --styles` | [`styles put`](references/lark-sheets-styles-put.md)；样式/合并/行高列宽/冻结一次规格，不拼 batch-update |
| 图表 | `+chart-create-basic` / `+batch-chart-create` | [`chart`](references/lark-sheets-chart.md)；已有图用 +chart-data-update / +chart-config-update；高级字段才用 +chart-create/update 局部 properties。多图核数量；图片迁移后删除并复查旧浮动图 |
| 分组汇总/透视 | `+pivot-create` | [`pivot table`](references/lark-sheets-pivot-table.md)；默认新 sheet 零覆盖；勿用 SUMIF/脚本假透视 |
| 排序/合并/清除/移动/复制/填充 | `+range-sort` / `+cells-clear` 等 | [`range operations`](references/lark-sheets-range-operations.md)；排序原子移动整行值/样式/空值，不本地排后写回（cells-set 空值不覆盖）；清除 --scope=content/formats/all |
| 筛选行 | `+filter-create` | [`filter`](references/lark-sheets-filter.md)；不裁列；另存筛选结果用新 sheet，保留原表 |
| 保存多份筛选状态 | `+filter-view-create` | [`filter view`](references/lark-sheets-filter-view.md)；与 filter 独立，隐藏仅在进入该视图时生效 |
| 查找/替换 | `+cells-search --find` / `+cells-replace` | [`search replace`](references/lark-sheets-search-replace.md)；无 +find/+cells-find/--query |
| 条件高亮/数据条/色阶/重复值 | `+cond-format-create` | [`conditional format`](references/lark-sheets-conditional-format.md)；不用逐格固定样式替代规则 |
| sheet 布局/插删/隐藏/冻结/大纲 | `+sheet-info` / `+dim-insert` / `+dim-freeze` 等 | [`sheet structure`](references/lark-sheets-sheet-structure.md)；冻结是整份状态覆盖，未写的轴会解冻；数据“按字段分组”走 pivot |
| 行高/列宽 | `+rows-resize` / `+cols-resize` | [`range operations`](references/lark-sheets-range-operations.md)；美化时并入 styles-put；无 --dimension |
| 工作簿/sheet 清单与管理 | `+workbook-info` / `+sheet-create/copy` 等 | [`workbook`](references/lark-sheets-workbook.md)；sheet-copy 的 sheet-name 是源、title 是新名 |
| 导出 xlsx/单表 CSV | `+workbook-export` | [`workbook`](references/lark-sheets-workbook.md)；CSV 必填专有 --sheet-id，落盘须 --output-path，省略仅轮询并返回任务结果 |
| 当前版本号 | `+revision-get` | 同上；保存编辑前 revision 可用于 changeset 复核 |
| 导入 xlsx/xls/csv 为在线表 | `+workbook-import --file ./x.xlsx` | 同上；只新建独立表。已有表追加子表：先 info 找模板→sheet-copy 保留结构/样式→只改数据；无模板才 sheet-create。仅导 bitable 用 drive +import --type bitable |
| 自由装饰图片/logo | `+float-image-create` | [`float image`](references/lark-sheets-float-image.md)；随记录走的图用 cells-set-image |
| 单元格趋势线 | `+sparkline-create` 等 | [`sparkline`](references/lark-sheets-sparkline.md)；不是 SPARKLINE() 公式/静态图 |
| 有序跨类型写链/批量清除 | `+batch-update` / `+cells-batch-clear` | [`batch update`](references/lark-sheets-batch-update.md)；高风险门禁；失败按参考处置 |
| 历史/异步回滚 | `+history-list/revert/revert-status` | [`history`](references/lark-sheets-history.md)；查询状态，不重放不明回滚 |
| 复核编辑 changeset | `+changeset-get` | [`changeset`](references/lark-sheets-changeset.md)；版本差最多 20 |
| 旧命令 unknown subcommand | 查替代及协议差异 | [`legacy migration`](references/lark-sheets-legacy-command-migration.md)；旧 42 命令已删除，不凭名称迁移 payload |

## 公共 flag 速查

- 选择恰好一个 `--url` / `--spreadsheet-token`；操作具体 sheet 再选恰好一个 `--sheet-id` / `--sheet-name`。不知道名称先 `+workbook-info`，不猜 Sheet1。`--range` 用 `A1:B2`，不加 sheetId! 前缀，也不能替代 sheet selector。
- 创建/导入工作簿无定位参数。workbook-info、batch-update、styles-put、dropdown-update/delete、cells-batch-clear、sheet-create 不接受 sheet selector；pivot-create 用可省略的 target-sheet-id/name。workbook-export 不接受公共 sheet selector，CSV 仅用其专有 --sheet-id。各对象主键分别为 sheet-id/chart-id/pivot-table-id/rule-id/view-id/group-id/float-image-id。

### Agent 使用提示

- 复合 JSON 不确定先 `--print-schema --flag-name <flag>`，深层用点分路径切片（如 properties.snapshot.plotArea.axes），无需认证/其他必填 flag；简单标量数组/CSV 无此 schema。高级图表可 `+chart-create --print-example <type>` 取最小模板。
- 大/多行输入优先受支持的 stdin `-`，每次仅一个 flag；两份大 JSON 时另一个用 `@./file.json`。@file 只接受 cwd 下相对路径，不用绝对路径/切 cwd 绕过。UTF-8 无 BOM；临时文件遵守宿主落点要求，无规定用系统临时目录，勿污染用户项目。
- bash/zsh 用单引号或 heredoc，不用 `set +H`。PowerShell 用 UTF-8 文件 + `@./file.json` 或 Get-Content -Raw 管道；cmd.exe 用 @file，避免 inline JSON/套用 heredoc。

已有本版本用法时，常用读/写无需额外发现调用：

```bash
lark-cli sheets +csv-get --url 'https://.../sheets/shtXXX' --sheet-name '已确认的表名' --range A1:F30
lark-cli sheets +cells-set --url 'https://.../sheets/shtXXX' --sheet-name '已确认的表名' --range A1 --cells '[[{"value":"原文"}]]'
```

## 输出、读取规模与风险

- stdout 是数据、stderr 是诊断，解析 JSON 不合并 `2>&1`；失败先读 stderr 再调整。不要仅凭 help/猜测声称场景不支持，先取得实际调用的报错。
- 先看 warning_message、has_more/truncated；落盘回执仍须 complete，检查 unread_sheets，文件并不证明读全。只读小表可直接 CSV；大表用 workbook-info + 小窗口定边界，再有界导出/脚本分析，不把全表灌进上下文。隐藏行默认包含，skip-hidden 会跳坐标；按返回行列段读写。
- 仓库版可用只读 `scripts/lark_inspect_workbook.py` / `lark_detect_subtables.py` / `lark_profile_table.py` 降低读取成本；**不是前置依赖，二进制不含 scripts/**，缺失直接用 workbook-info / sheet-info / 小窗口 csv-get。画像风险、尾行/空区/隐藏区复核见 read-data，启发式追加列不能直接当安全写点。
- 翻译/打标/改写等用语义判断，脚本仅协调批次/行号/写回；通常每批 30 行，每批生成不超过约 300 行，及时写回，多批按依赖决定合批。
- **高风险**：batch-update、cells-clear、cells-batch-clear、sheet-delete、dim-delete、dropdown-delete 及 chart/pivot/cond-format/filter/filter-view/sparkline/float-image 删除。先 --dry-run 展示操作及影响，**获用户明确同意后**才在原 argv 末追加 --yes；exit 10 不得静默补 flag。完整协议按 lark-shared。
