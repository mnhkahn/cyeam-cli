---
name: mail
version: 0.1.18
description: 多邮箱收发——通过 IMAP/SMTP 读取和发送邮件，支持 Zoho/cyeam、Gmail、iCloud 等多个账户。用户要看邮件、读某封邮件、发邮件、标记已读/未读时使用。【重要】必须先读 skill 原文获取正确命令格式，禁止瞎猜。
---

# 多邮箱收发

## 规则

**用户要看/读/发/标记邮件 → 跑 cyeam mail 命令 → 返回结果。**

- **看邮件：默认用 `--all` 列出所有邮箱**，用户指定具体邮箱才用 `<账户>`
- **读邮件：必须指定 `<账户> <uid>`，可加 `--mark-read` 同时标记已读**
- **发邮件：必须指定 `<账户>`**
- **标记已读/未读：必须指定 `<账户>` 和 UID，或用 `--uids` 批量**

```
用户: "看看我有什么新邮件"                  →  cyeam mail list --all
用户: "看看我 cyeam 邮箱有什么邮件"         →  cyeam mail list cyeam
用户: "读一下 UID 123 那封"                →  cyeam mail read cyeam 123
用户: "读一下 UID 123 并标记已读"           →  cyeam mail read cyeam 123 --mark-read
用户: "把 UID 123 标记为已读"              →  cyeam mail mark-read cyeam 123
用户: "把 UID 123,456,789 都标记为已读"    →  cyeam mail mark-read cyeam --uids 123,456,789
用户: "把 UID 123 标记为未读"              →  cyeam mail mark-unread cyeam 123
用户: "给 x@y.com 发封邮件"                →  cyeam mail send cyeam --to x@y.com --subject "..." --body "..."
```

读邮件分两步：先 `list --all` 拿到 UID 和对应账户，再 `read <账户> <uid>` 读正文。

## 命令

```
cyeam mail list --all [--limit 20]                列所有邮箱的最近邮件：账户/UID/发件人/主题/时间/未读
cyeam mail list <账户> [--limit 20]               列指定邮箱的最近邮件
cyeam mail read <账户> <uid> [--mark-read]        读单封：发件人/收件人/主题/日期/正文，可选标记已读
cyeam mail mark-read <账户> <uid>                 标记单封为已读
cyeam mail mark-read <账户> --uids <uid1,uid2>    批量标记已读，逗号分隔
cyeam mail mark-unread <账户> <uid>               标记单封为未读
cyeam mail mark-unread <账户> --uids <uid1,uid2>  批量标记未读
cyeam mail send <账户> --to <地址> --subject <主题> --body <正文>
    [--cc <地址>] [--body-file <文件>]             发送邮件（--to/--cc 可重复）
```

## 配置

账户列表写在 `~/.cyeam/mail.json`，**推荐用户名和密码都从环境变量读取**，避免明文写入配置文件：

```json
{
  "accounts": [
    {
      "name": "cyeam",
      "imap_host": "imap.zoho.com",
      "imap_port": 993,
      "username_env": "CYEAM_MAIL_USERNAME",
      "password_env": "CYEAM_MAIL_PASS",
      "smtp_host": "smtp.zoho.com",
      "smtp_port": 465
    }
  ]
}
```

- `username_env`：用户名所在的环境变量名，运行前需 `export CYEAM_MAIL_USERNAME=you@cyeam.com`
- `username`：用户名明文写在配置中（不推荐），如设置则优先级高于 username_env
- `password_env`：应用专用密码所在的环境变量名，运行前需 `export CYEAM_MAIL_PASS=xxx`
- `smtp_host`/`smtp_port` 可省略：默认把 `imap.` 换成 `smtp.`，端口默认 465
- 多账户就在 `accounts` 里多加几项，命令里用 `name` 指定

常见邮箱 IMAP/SMTP：

| 邮箱   | IMAP            | SMTP            |
|--------|-----------------|-----------------|
| Zoho   | imap.zoho.com   | smtp.zoho.com   |
| Gmail  | imap.gmail.com  | smtp.gmail.com  |
| iCloud | imap.mail.me.com| smtp.mail.me.com|

## 前置条件（用户侧）

- 邮箱后台开启 IMAP 访问
- 生成「应用专用密码」（不是登录密码），设到对应环境变量
- 端口 465 用 TLS，587 用 STARTTLS

## 输出

所有命令输出 JSON 信封，加 `--pretty` 去掉信封。

## 不做的事

- 删除、移动邮件
- 附件下载、全文搜索
