---
name: trello
description: Trello 学习任务看板——通过 cyeam CLI 查询看板、状态分组和当天任务；创建、修改、移动任务；上传和查看成果照片；查看任务历史；注册或删除实时 Webhook。用户提到 Trello、学习任务看板、待执行/已提交/已验收状态、任务照片或 Trello 同步时使用。必须先读 skill 原文获取正确命令格式，禁止瞎猜。
---

# Trello 任务看板

使用 `cyeam trello`，不要自行拼装 Trello HTTP 请求。首次使用先运行 `cyeam trello login --key <api-key>`，在浏览器授权后把页面显示的 Token 粘贴回 CLI；CLI 验证并存入系统钥匙串，钥匙串不可用时回退到权限为 `0600` 的 `~/.cyeam/trello.json`。绝不输出凭据。

## 规则

- 不知道 ID 时，先 `boards`，再 `lists <board-id>`；不要按名称猜测 ID。
- 查询当天任务使用 `cards --board <board-id> --today`；按本机时区和 `due` 筛选。
- 创建、修改、移动、上传附件及 Webhook 增删会改变外部状态，必须先获得用户明确确认。
- Webhook 命令只管理订阅；公网 HTTPS 接收端、验签、去重和安卓同步由独立服务实现。
- 环境变量 `TRELLO_API_KEY` 和 `TRELLO_TOKEN` 仅用于 CI 或临时覆盖；设置其中一个时必须同时设置另一个。

## 命令

```text
cyeam trello login --key <api-key>
cyeam trello status
cyeam trello logout

cyeam trello boards
cyeam trello lists <board-id>
cyeam trello cards --list <list-id> [--today]
cyeam trello cards --board <board-id> [--today]
cyeam trello status-changes --board <board-id>
    [--date <YYYY-MM-DD>] [--to-list <list-id>] [--limit 1000]

cyeam trello card create --list <list-id> --name <title>
    [--desc <text>] [--due <RFC3339>] [--labels <label-id,...>]
cyeam trello card update <card-id>
    [--name <title>] [--desc <text>] [--due <RFC3339>] [--complete=true|false]
cyeam trello card move <card-id> --list <target-list-id>
cyeam trello card attach <card-id> --file <local-path> [--name <display-name>]
cyeam trello card attachments <card-id>
cyeam trello card attachment get <card-id> <attachment-id>
    [--out <local-path>] [--max-bytes <bytes>]
cyeam trello card actions <card-id> [--limit 50]

cyeam trello webhook create --callback-url <https-url> --model-id <id>
    [--description <text>]
cyeam trello webhook delete <webhook-id>
```

`card update --due ""` 清除截止时间；未传 `--complete` 时不改变任务完成标记。默认输出是 JSON 信封，`--pretty` 输出响应 JSON。

获取图片时先用 `card attachments <card-id>` 找到 `attachment-id`、文件名和 MIME 类型，再调用 `card attachment get`。需要在飞书展示时优先传 `--out <local-path>`：CLI 会使用已保存的 Trello 凭据下载受保护附件，并把原始图片字节直接写入文件，返回 `saved_to`、`name`、`mime_type` 和 `size_bytes`；Agent 随后上传该本地文件。不要自行解析 JSON、解码 Base64，也不要把受保护 URL 直接发给飞书。仅在调用方明确需要内联数据时省略 `--out`，此时响应才包含 `base64`。附件默认上限为 25 MiB。

`status-changes` 按本机时区查询某一天的卡片列表流转记录，默认查询今天，返回卡片、流转前后列表、操作者和 `changed_at`。整理每日完成任务时，必须先用 `lists <board-id>` 找到完成列表 ID，再传 `--to-list <list-id>`；不要按列表名称猜测。
