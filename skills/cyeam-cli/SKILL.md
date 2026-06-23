---
name: cyeam-cli
version: 0.1.16
description: cyeam.com CLI — 安装、更新、输出格式说明、命令速查。各领域详情见独立 skill（live-broadcast、pinyin 等）。架构咨询走 MCP。 -- 不可直接作为工具名调用，请通过 cyeam 命令使用
allowed-tools: Bash(cyeam:*)
---

# Cyeam CLI

## 规则

这个 skill 只描述 CLI 本身的安装、更新、输出格式和完整命令列表。**每个领域的具体用法和注意事项在对应的独立 skill 中（live-broadcast、pinyin、mo 等），AI 应先查那些 skill 再执行命令。**

---

## 安装

```bash
# macOS Apple Silicon
curl -L https://github.com/mnhkahn/cyeam-cli/releases/latest/download/cyeam_Darwin_arm64.tar.gz | tar xz && chmod +x cyeam && sudo mv cyeam /usr/local/bin/
# macOS Intel
curl -L https://github.com/mnhkahn/cyeam-cli/releases/latest/download/cyeam_Darwin_x86_64.tar.gz | tar xz && chmod +x cyeam && sudo mv cyeam /usr/local/bin/
# Linux amd64
curl -L https://github.com/mnhkahn/cyeam-cli/releases/latest/download/cyeam_Linux_x86_64.tar.gz | tar xz && chmod +x cyeam && sudo mv cyeam /usr/local/bin/
```

安装检查：`cyeam version`

---

## 登录方式

```bash
cyeam login                           # 默认：打开浏览器完成 Microsoft 登录
cyeam login --print-link              # 仅打印链接和验证码，适合远程/服务器环境
cyeam logout                          # 退出登录
cyeam whoami                          # 查看当前登录状态
```

- token 优先存系统钥匙串（macOS Keychain / Linux D-Bus），不可用时自动回退到 `~/.cyeam/token.json`
- `--print-link` 不尝试打开浏览器，只输出 URL + 验证码，用户在本地浏览器打开授权后 CLI 自动获取 token
- 无 D-Bus 钥匙串的服务器（无桌面环境）必须用 `--print-link`

## 完整命令列表

```bash
cyeam version                          # 版本
cyeam update                           # 更新
cyeam login / logout / whoami          # Microsoft 登录
cyeam date holiday [YYYY-MM-DD]        # 节假日
cyeam tv ...                           # 直播 → 见 live-broadcast skill
cyeam news ...                         # 新闻 → 见 news skill
cyeam pinyin ...                       # 拼音 → 见 pinyin skill
cyeam mo ...                           # 书法 → 见 mo skill
cyeam roadbook ...                     # 路书 → 见 roadbook skill
cyeam cnote ...                        # 云笔记 → 见 cnote skill
cyeam ai models [...]                  # AI免费模型 → 见 ai-models skill
```

所有命令输出 JSON 信封 `{"ok":true,"data":"...","_notice":{...}}`。加 `--pretty` 去掉信封。

---

## 架构咨询

不走 CLI，直接 MCP:

```
mcp: https://www.cyeam.com/arch/mcp
```

---

## 不做的事

- 直播流抓取、m3u8 解码、会员/地区绕过
- 开发者转换工具（json2go、curl2go、XML、SQL 等）
- 翻译、搜索建议、QR 码
- Mo AI 存库 API
- Admin/缓存/上传/推送管理