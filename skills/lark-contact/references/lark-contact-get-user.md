# +get-user

按 ID 取用户基本信息(姓名等)。

```bash
# 取自己
lark-cli contact +get-user --as user

# user 按 ID 取他人基本资料
lark-cli contact +get-user --user-id ou_xxx --as user

# bot 按 ID 取他人
lark-cli contact +get-user --user-id ou_xxx --as bot

# 按 union_id / user_id 取(默认 open_id)
lark-cli contact +get-user --user-id <id> --user-id-type union_id --as bot
```

## 注意事项

- **user 身份也支持按 ID 取基本资料**（如姓名）；需要部门 / 邮箱 / 是否激活等更多字段，或批量查询多个 open_id 时，再用 `+search-user --user-ids <id1,id2>`。
- **`--as bot` 必须传 `--user-id`**:不传会直接报错(只有 user 身份能省略 `--user-id` 取自己)。
