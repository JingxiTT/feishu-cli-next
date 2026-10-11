---
name: lark-im
version: 1.0.0
description: "飞书即时通讯：收发消息和管理群聊。发送和回复消息、搜索聊天记录、管理群聊成员、上传下载图片和文件、管理表情回复、发送应用内/短信/电话加急、发送和处理交互卡片（Interactive Card）、监听卡片按钮回调（card.action.trigger）。当用户需要发消息、查看或搜索聊天记录、下载聊天中的文件、查看群成员、搜索群、创建群聊或话题群、管理标记数据、管理 Feed 置顶（添加/移除/查询置顶会话）、管理标签数据、处理卡片回调时使用。"
metadata:
  requires:
    bins: ["lark-cli"]
  cliHelp: "lark-cli im --help"
---

# im (v1)

开始前读取一次 [`lark-shared`](../lark-shared/SKILL.md)，遵守身份、认证、输出与高风险规则；复用本任务已读的同版本指导。已知命令和参数可直接执行，缺少要求时只读下表对应 reference 或该命令 `--help`。

## 身份、资源与安全

- `--as user` 使用用户的 `user_access_token`；`--as bot` 使用应用的 `tenant_access_token`。支持两者不代表权限相同：群成员身份、所有者/管理员、租户和应用可见范围都按当前操作者检查。群权限/所有权问题读 [`chat identity`](references/lark-im-chat-identity.md)，不要盲目切换身份重试。
- `chat_id=oc_xxx` 标识群/P2P；`message_id=om_xxx` 标识消息；`thread_id=om_xxx/omt_xxx` 标识话题。Reaction 是消息表情；Flag 是消息/话题书签；Feed Shortcut 是侧栏置顶聊天；Feed Group (`ofg_xxx`) 是会话标签，其成员由 `feed_id + feed_type` 标识。normal 手动管理，rule 按规则派生。
- 发消息前必须取得用户对收件人、内容和发送身份的明确授权。保留用户给定的目标和内容；不要擅自扩大群成员或接收范围。
- 优先返回 `chat_app_link`（打开已加入会话）、`message_app_link`（消息）、`share_link`（邀请入群）。手拼会话链接仅用 `https://<applink_host>/client/chat/open?openChatId=<oc_xxx>`，不用 `chatId` 或 `lark://...chat_id=`。
- **发送、回复、更新 interactive 卡片前，必须读 [`card create`](references/card/lark-im-card-create.md) 并按工作流产出 JSON，不能手写或复制 payload。** [`card action reply`](references/lark-im-card-action-reply.md) 处理 `card.action.trigger`；订阅中的卡片尚无紧凑转换，返回原始事件并在 stderr 提示。
- `--audio` 仅接受 Opus（`.opus` / Ogg Opus `.ogg`）；其他格式先转换，或用 `--file` 原样发附件。转发文档先以 `--doc-format im-markdown` 获取，再以 `--markdown` 发送；保留原文与 `cite type="user"` 标签。

## 常用路径

已有本版本参数和授权时，直接调用；占位符替换为已确认的 ID：

```bash
lark-cli im +chat-messages-list --chat-id oc_xxx --as user
lark-cli im +messages-search --query '关键词' --as user
lark-cli im +messages-send --user-id ou_xxx --text '已获授权的原文' --as user
```

消息读取已返回 sender `name`（服务端 `sender_name`，缺失时回退 ID）、完整 `sender_i18n_names` 和可选 `open_bot_id`；system 无姓名属正常。**不要为此另查通讯录或机器人资料**，无需额外 contact / `application:bot.basic_info:read` 权限。

四种消息读取（chat/thread list、mget、search）默认补 reactions 和已编辑消息的 update_time，勿另调 reactions batch_query；只需正文可用 `--no-reactions`。chat list/mget 还展开有界的 thread replies。检查分页、截断及逐项失败；字段缺失不等于获取失败，完整契约见 [`message enrichment`](references/lark-im-message-enrichment.md)。chat list/mget/thread list 可选 `--download-resources` 保存到 `./lark-im-resources/`，默认关闭、排除 sticker，单资源失败不终止消息读取。

## 按意图选 Shortcut / Reference

有 shortcut 优先用它；无需一次读完所有参考。

| 意图 | 命令与参考 | 关键分流 |
|---|---|---|
| 创建群/话题群 | [`+chat-create`](references/lark-im-chat-create.md) | user/bot；可指定 owner/manager |
| 已加入群/P2P | [`+chat-list`](references/lark-im-chat-list.md) | 默认群；P2P、exclude-muted 仅 user |
| 搜索可见群 | [`+chat-search`](references/lark-im-chat-search.md) | 关键词/成员筛选；exclude-muted 仅 user |
| 群成员 | [`+chat-members-list`](references/lark-im-chat-members-list.md) | users/bots 分桶；检查 truncations |
| 群名/描述 | [`+chat-update`](references/lark-im-chat-update.md) | 权限取决于当前操作者 |
| 群/P2P 消息 | [`+chat-messages-list`](references/lark-im-chat-messages-list.md) | chat-id 或 user-id；时间、顺序、分页 |
| 话题消息 | [`+threads-messages-list`](references/lark-im-threads-messages-list.md) | om_/omt_；解析 thread_id |
| 已知消息 ID | [`+messages-mget`](references/lark-im-messages-mget.md) | 最多 50 条；展开话题回复 |
| 跨会话搜索 | [`+messages-search`](references/lark-im-messages-search.md) | 关键词/发送者/附件/时间；有界分页 |
| 发送/回复 | [`+messages-send`](references/lark-im-messages-send.md) / [`+messages-reply`](references/lark-im-messages-reply.md) | text 原文；markdown 轻格式；content 精确 JSON；幂等键 |
| 编辑消息 | [`+messages-edit`](references/lark-im-messages-edit.md) | **bot-only**；text/post/附件区 |
| 下载单个附件 | [`+messages-resources-download`](references/lark-im-messages-resources-download.md) | 消息 ID + 资源 key |
| 已读状态/已读用户 | [`+messages-read-status` / `+message-read-users`](references/lark-im-message-read-status.md) | 前者 user-only、1–50 条；后者依身份检查权限 |
| 书签创建/取消/列表 | [`+flag-create`](references/lark-im-flag-create.md) / [`+flag-cancel`](references/lark-im-flag-cancel.md) / [`+flag-list`](references/lark-im-flag-list.md) | user-only；message 层为普通书签，feed 层话题 item_type=4、普通群话题=11；默认取消尽力清两层；has_more 表示未读全 |
| 侧栏置顶 | [`+feed-shortcut-create`](references/lark-im-feed-shortcut-create.md) / [`+feed-shortcut-remove`](references/lark-im-feed-shortcut-remove.md) / [`+feed-shortcut-list`](references/lark-im-feed-shortcut-list.md) | user-only、仅 CHAT (`oc_xxx`)；增删每批 10；list 单页；--no-detail 省额外群查询；检查 ok:false 逐项账本 |
| 标签列表/成员/已知成员 | [`+feed-group-list`](references/lark-im-feed-group-list.md) / [`+feed-group-list-item`](references/lark-im-feed-group-list-item.md) / [`+feed-group-query-item`](references/lark-im-feed-group-query-item.md) | user-only；已知 ID 用 query，避免全量 list |
| 标签创建/更新/删除/增删成员 | [`feed.groups.*`](references/lark-im-feed-groups.md) | 执行前读参考；优先 normal、最少规则 |
| 表情读取/增删 | [`reactions.*`](references/lark-im-reactions.md) | 执行前读参考；只能删自己添加的表情 |

## 原生 API

`lark-cli im <resource> <method>` 用 `--params` / `--data`，不要猜字段。首次构造未知请求先查 `lark-cli schema im.<resource>.<method> --input-only`；需要响应定义时省略 `--input-only`。复用已读的同版本 schema。权限以命令 metadata/schema 为准，不在此重复 scope 表。

以下为影响路由或数据安全的限制；其余输入/输出见所选 method 的 schema：

| 原生资源/方法 | 限制 |
|---|---|
| `chats.create` | bot-only；创建群优先上面的 user/bot shortcut |
| `chats.get/link/update` | 操作者须在群中；内部群须同租户；限制分享时 link 须 owner/admin；update 支持 user/bot |
| `chat.members.create/delete` | create 须在群中，bot 邀请对象须在应用可见范围，内部群须同租户；仅管理者可邀时须 owner/admin 或持 `im:chat:operate_as_owner` 的建群 bot。delete 须 owner/admin/建群 bot，每次最多 50 用户或 5 bot |
| `chat.user_setting.batch_query/batch_update` | user-only、须为各目标群成员，每次最多 10 群 |
| `chat.nickname.get/update/delete` | user-only，只操作自己；update 非空且最多 300 字节，清空用 delete |
| `chat.join_requests.list/handle` | user-only、owner/admin；list page_size 1–100，**按 has_more 停止，末页仍可能有 page_token**；handle 每次 1–50，逐项检查 results（success/failed/already_handled） |
| `chat.managers.add_managers/delete_managers` | 仅 owner；管理员上限 10（超大群 20）；每次最多 5 bot，删除用户每次最多 50 |
| `chat.moderation.get/update` | 操作者须在群；get 须同租户，update 须 owner 或持 `im:chat:operate_as_owner` 的建群 bot |
| `messages.delete` | bot 撤群消息须在群，撤他人消息须 owner/admin/建群 bot；user P2P 撤回对象须在应用可见范围 |
| `messages.read_status/read_users` | 必须先读上面的已读参考；read_status user-only 最多 50；read_users 须仍在群，只查该身份自己发出的近 7 天消息 |
| `messages.patch` | 先读卡片工作流；仅应用近 14 天发出的卡片，content 为 JSON 字符串且最多 30 KB |
| `messages.merge_forward/urgent_app/urgent_phone/urgent_sms` | bot-only；加急须 bot 为发送者且在会话中 |
| `messages.forward` / `threads.forward` / `pins.create/delete/list` | user/bot；转发和置顶遵守用户目标与授权 |
| `images.create/files.create` | user/bot；user 上传须 `im:resource` |
| `reactions.*` | user/bot、须在消息会话；按上面参考执行 |
| `feed.groups.*` | user-only；读上面参考，结构化 params/data；三个读取 shortcut 不另拼 raw 方法 |
