---
name: cyeam-cli
version: 0.1.16
description: cyeam.com CLI — 安装、更新、输出格式说明、命令速查。各领域详情见独立 skill（live-broadcast、pinyin 等）。架构咨询走 MCP。【重要】必须先读 skill 原文获取正确命令格式，禁止瞎猜。
allowed-tools: Bash(cyeam:*)
---

# Cyeam CLI

## 规则

这个 skill 只描述 CLI 本身的安装、更新、输出格式和完整命令列表。**每个领域的具体用法和注意事项在对应的独立 skill 中（live-broadcast、pinyin、calligraphy-composition、seal-script-ocr 等），AI 应先查那些 skill 再执行命令。**

---

## 安装

安装前先阅读下面的命令。示例固定为 **v1.1.4**，不跟随 latest；下载的是可执行程序归档。
仅在需要安装 CLI 时执行，不要在每次调用命令时重复安装。

### macOS / Linux：校验后安装到用户目录

在 Bash 或 Zsh 中执行；需要 `curl`、`tar`、`install`，以及 `shasum` 或 `sha256sum`。
脚本自动选择 macOS / Linux 的 arm64 或 x86_64，其他平台会停止。

```bash
(
  set -eu
  version='v1.1.4'
  case "$(uname -s)/$(uname -m)" in
    Darwin/arm64)
      asset='cyeam_Darwin_arm64.tar.gz'
      sha256='06b5af004ab4f0f05d5829ea6bd18b97c4449c262af7df14c67175f7d3cb29db' ;;
    Darwin/x86_64)
      asset='cyeam_Darwin_x86_64.tar.gz'
      sha256='107342bb9d2fa444b96155c468e71b8cc63e0f5ad0acc784698afe221548efa6' ;;
    Linux/aarch64|Linux/arm64)
      asset='cyeam_Linux_arm64.tar.gz'
      sha256='ed87697f3948bead67b585e9881c3bef20e8d2d903beb5b125587281a5091c5b' ;;
    Linux/x86_64)
      asset='cyeam_Linux_x86_64.tar.gz'
      sha256='873d6805fab8ab56540bb034aaf2f57f12eb672fcabd322ebe401e8250a5966d' ;;
    *) echo '不支持的平台，请查看固定版本的 Release。' >&2; exit 1 ;;
  esac
  workdir=$(mktemp -d)
  trap 'rm -rf "$workdir"' EXIT
  cd "$workdir"
  curl --fail --show-error --silent --location --proto '=https' --proto-redir '=https' \
    --output "$asset" "https://github.com/mnhkahn/cyeam-cli/releases/download/$version/$asset"
  printf '%s  %s\n' "$sha256" "$asset" > expected.sha256
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -c expected.sha256
  else
    shasum -a 256 -c expected.sha256
  fi
  # set -e 确保下载或校验失败时，不会解压或安装。
  tar -xzf "$asset" cyeam
  mkdir -p "$HOME/.local/bin"
  install -m 0755 cyeam "$HOME/.local/bin/cyeam"
  CYEAM_CLI_NO_UPDATE_NOTIFIER=1 "$HOME/.local/bin/cyeam" --no-analytics version
)
```

只写临时目录和 `~/.local/bin/cyeam`（已有同名程序会被替换），无需管理员权限，
不修改系统目录、服务或 shell 配置。当前 shell 使用：

```bash
export PATH="$HOME/.local/bin:$PATH"
```

如需永久生效，可自行把上面这一行加入 shell 配置。
可在 [v1.1.4 发布页](https://github.com/mnhkahn/cyeam-cli/releases/tag/v1.1.4) 和
[checksums.txt](https://github.com/mnhkahn/cyeam-cli/releases/download/v1.1.4/checksums.txt)
核对摘要。升级固定版本时，必须同时核对并替换版本号和对应 SHA-256。
摘要用于核对发布文件一致性，不等同于独立签名或安全审计。

### 外部请求和权限用途

- **安装下载**：只从 GitHub 的固定 Release 下载选定归档；GitHub 可能重定向到其资产 CDN。
  下载后先验证文档中固定的 SHA-256，再解压单个 `cyeam` 文件。安装检查关闭统计与更新提示。
- **书法查询**：`mo guwen`、`mo char detail/composition/compose` 向
  `https://www.cyeam.com/mo/api/*` 发送用户指定的文字、字体及合成选项；合成图写到 `--out` 指定路径，
  已有文件可能被覆盖。返回的原帖图片可能来自 Cloudinary 等图片 CDN，展示图片时客户端会请求相应 URL。
- **OCR**：`mo ocr <image>` 读取指定的本地文件，向 `https://www.cyeam.com/mo/api/ocr`
  上传文件名和图片内容以完成识别。仅在用户要求识别该图片时运行；不需要 Microsoft 登录。
- **使用统计**：默认向 `https://us.i.posthog.com` 发送命令名称、CLI 版本、系统信息和随机安装编号，
  服务端可根据出口 IP 解析地域；不发送命令参数、图片内容或登录凭证。
  随机编号保存在 `~/.cyeam/analytics-id`，用 `--no-analytics` 或 `CYEAM_CLI_NO_ANALYTICS=1` 关闭。
- **更新提示**：默认最多每 24 小时向 GitHub 检查最新版本，并缓存到 `~/.cyeam/update-state.json`，
  提示本身不会替换程序。设置 `CYEAM_CLI_NO_UPDATE_NOTIFIER=1` 关闭。
  `cyeam update` 是显式升级命令，会下载并校验最新 Release、替换当前程序，成功后还会尝试通过
  `npx skills add` 同步 skills；需要保持固定版本时不要运行它，按上述步骤安装选定版本。

仅保留 `mo` 业务必需的网络请求，可在当前 shell 设置：

```bash
export CYEAM_CLI_NO_ANALYTICS=1
export CYEAM_CLI_NO_UPDATE_NOTIFIER=1
```

## 登录方式

```bash
cyeam login                           # 登录 Microsoft 账号
cyeam logout                          # 退出登录
cyeam whoami                          # 查看当前登录状态
```

- 登录采用设备码流程：命令会打印授权链接和验证码，并在有桌面环境的机器上自动尝试打开浏览器
- 无桌面环境的服务器上浏览器打开会静默失败，按打印出的链接和验证码在任意浏览器完成授权即可，CLI 会自动获取 token
- token 优先存系统钥匙串（macOS Keychain / Linux D-Bus），不可用时自动回退到 `~/.cyeam/token.json`

## 完整命令列表

```bash
cyeam version                          # 版本
cyeam update                           # 更新
cyeam tag minor|major [--dry-run]       # Git 版本标签 → 见 git-tag skill
cyeam login / logout / whoami          # Microsoft 登录
cyeam date holiday [YYYY-MM-DD]        # 节假日
cyeam tv ...                           # 直播 → 见 live-broadcast skill
cyeam news ...                         # 新闻 → 见 news skill
cyeam pinyin ...                       # 拼音 → 见 pinyin skill
cyeam phonetic <word>                  # 英语音标 → 见 phonetic skill
cyeam mo guwen/char ...                # 书法集字 → 见 calligraphy-composition skill
cyeam mo ocr <image>                  # 篆字识别 → 见 seal-script-ocr skill
cyeam roadbook ...                     # 路书 → 见 roadbook skill
cyeam cnote ...                        # 云笔记 → 见 cnote skill
cyeam mail ...                         # 多邮箱收发 → 见 mail skill
cyeam image ...                        # 图片转换 → 见 image skill
cyeam ai models [...]                  # AI免费模型 → 见 ai-models skill
cyeam trello ...                       # Trello任务看板 → 见 trello skill
```

所有命令输出 JSON 信封 `{"ok":true,"data":...,"_notice":{...}}`：命令输出是合法 JSON 时 `data` 为嵌套的 JSON 对象/数组，否则为字符串。加 `--pretty` 去掉信封。
