---
name: lark-openapi-explorer
version: 1.0.0
description: "飞书/Lark 原生 OpenAPI 探索：从官方文档库中挖掘未经 CLI 封装的原生 OpenAPI 接口。当用户的需求无法被现有 lark-* skill 或 lark-cli 已注册命令满足，需要查找并调用原生飞书 OpenAPI 时使用。"
metadata:
  requires:
    bins: ["lark-cli"]
---

# OpenAPI Explorer

先阅读一次 [`lark-shared`](../lark-shared/SKILL.md)，遵守认证、身份、输出与安全规则；复用本任务已读的同版本指导。本 skill 仅用于现有 shortcut / 已注册 API 无法满足的请求，不把已有命令绕成裸调。

## 先查本地能力

已知命令和参数时直接执行。未知命令用简短字面关键词做有界发现；仅缺 flag 要求时加 `--flags`：

```bash
feishu-cli-next discover 'im members' --limit 3
feishu-cli-next discover 'im members' --limit 3 --flags
```

结果不足时细化关键词；宿主没有 Next discover 时，直接检查 `lark-cli <service> --help` 及相关 resource 的帮助。单次未命中不能证明没有能力。已有方法只缺请求形状时，查**一个具体 method** 的 request schema；需要响应定义再省略 `--input-only`，不要裸跑 schema 倾倒全目录：

```bash
lark-cli schema im.chat.members.create --input-only
```

例如拉人进群已有 `im chat.members create`；按 [`lark-im`](../lark-im/SKILL.md) 的成员/身份指导执行，不为它抓取外部文档或另拼 raw API。仅确认本地能力不覆盖后，继续下方流程。

## 按需读取官方规范

按实际品牌选入口，未知默认飞书：

| 品牌 | 文档索引 |
|---|---|
| Feishu | `https://open.feishu.cn/llms.txt` |
| Lark | `https://open.larksuite.com/llms.txt` |

文档层级是 `llms.txt` → `llms-<module>.txt` → 单 API markdown。已有准确官方 API 文档链接时直接读取该页；缺模块/接口链接时逐层定位，**不要猜路径、字段或跳过所需规范**。只读取当前接口需要的材料，复用已读的规范。

从单 API 文档确认 HTTP 方法/路径、路径与查询参数、请求体类型和必填项、响应、身份/scope、错误与限制。官方文档为中文；用户用英文时用英文解释。

确认规范和用户意图后，通过 CLI 裸调，保留原始参数语义：

```bash
lark-cli api GET /open-apis/<path> --params '{"key":"value"}'
lark-cli api POST /open-apis/<path> --data '{"key":"value"}'
lark-cli api PUT /open-apis/<path> --data '{"key":"value"}'
lark-cli api DELETE /open-apis/<path>
```

写入/删除前确认用户意图，敏感操作说明影响范围；支持 dry-run 时先预览。遵守 lark-shared 的高风险审批与未知结果恢复要求，不能静默绕过确认门禁。

向用户给出一句话功能、HTTP 方法/路径、关键参数、scope、完整调用示例及相关限制；无需倾倒整页规范。如需把已验证的缺口固化成技能，见 [`lark-skill-maker`](../lark-skill-maker/SKILL.md)。
