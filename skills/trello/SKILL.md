---
name: trello
description: Trello 学习任务看板——通过 cyeam CLI 查询看板、状态分组和当天任务；创建、修改、移动任务；上传和查看成果照片；查看任务历史；注册或删除实时 Webhook。用户提到 Trello、学习任务看板、待执行/已提交/已验收状态、任务照片或 Trello 同步时使用。必须先读 skill 原文获取正确命令格式，禁止瞎猜。
---

# Trello 任务看板

使用 `cyeam trello`，不要自行拼装 Trello HTTP 请求。认证只从 `TRELLO_API_KEY` 和 `TRELLO_TOKEN` 环境变量读取，绝不输出其值。

## 规则

- 不知道 ID 时，先 `boards`，再 `lists <board-id>`；不要按名称猜测 ID。
- 查询当天任务使用 `cards --board <board-id> --today`；按本机时区和 `due` 筛选。
- 创建、修改、移动、上传附件及 Webhook 增删会改变外部状态，必须先获得用户明确确认。
- Webhook 命令只管理订阅；公网 HTTPS 接收端、验签、去重和安卓同步由独立服务实现。

## 命令

```text
cyeam trello boards
cyeam trello lists <board-id>
cyeam trello cards --list <list-id> [--today]
cyeam trello cards --board <board-id> [--today]

cyeam trello card create --list <list-id> --name <title>
    [--desc <text>] [--due <RFC3339>] [--labels <label-id,...>]
cyeam trello card update <card-id>
    [--name <title>] [--desc <text>] [--due <RFC3339>] [--complete=true|false]
cyeam trello card move <card-id> --list <target-list-id>
cyeam trello card attach <card-id> --file <local-path> [--name <display-name>]
cyeam trello card attachments <card-id>
cyeam trello card actions <card-id> [--limit 50]

cyeam trello webhook create --callback-url <https-url> --model-id <id>
    [--description <text>]
cyeam trello webhook delete <webhook-id>
```

`card update --due ""` 清除截止时间；未传 `--complete` 时不改变任务完成标记。默认输出是 JSON 信封，`--pretty` 输出响应 JSON。
