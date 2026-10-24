---
name: cnote
version: 0.1.16
description: 云端笔记——在 OneDrive 上管理轻量 HTML 笔记，支持查看、新建、追加、列表。需要 Microsoft 登录。想查电视节目去问 tv skill。【重要】必须先读 skill 原文获取正确命令格式，禁止瞎猜。
---

# 云端笔记

## 概述

在 OneDrive `Notes` 文件夹中管理 HTML 笔记。需要先登录 Microsoft 账号（登录方式见 cyeam-cli skill 的"登录方式"）。

```bash
cyeam cnote list                     # 列出所有笔记
cyeam cnote get "笔记标题"            # 查看（默认 markdown 格式）
cyeam cnote get "笔记标题" --format text
cyeam cnote new "笔记标题" < note.html
cyeam cnote append "笔记标题" < more.html
```

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
- **Microsoft 授权**：仅在需要云端功能时运行 `cyeam login`，通过
  `https://login.microsoftonline.com/consumers` 的设备码流程登录，并尝试打开授权页面。
  请求 `Files.ReadWrite`（读写 OneDrive 文件）、`User.Read`（读取账号信息）、
  `offline_access`（刷新令牌，免于每次登录）。文件权限覆盖授权账号的 OneDrive 文件，
  不局限于本技能的文件夹；令牌优先存系统钥匙串，不可用时回退到 `~/.cyeam/token.json`。
- **云笔记**：通过 `https://graph.microsoft.com/v1.0` 读写 OneDrive 的 `Notes/` 文件夹；
  Microsoft 的文件下载可能重定向到其存储域名。`list/get` 读取文件信息或内容，
  `new` 上传 stdin 中的 HTML（同名文件会被替换），`append` 读取原文再上传追加后的全文。
  这些操作需要用户要求的笔记内容和目标标题，不需要系统管理员权限。
- **使用统计**：默认向 `https://us.i.posthog.com` 发送命令名称、CLI 版本、系统信息和随机安装编号，
  服务端可根据出口 IP 解析地域；不发送命令参数、图片内容或登录凭证。
  随机编号保存在 `~/.cyeam/analytics-id`，用 `--no-analytics` 或 `CYEAM_CLI_NO_ANALYTICS=1` 关闭。
- **更新提示**：默认最多每 24 小时向 GitHub 检查最新版本，并缓存到 `~/.cyeam/update-state.json`，
  提示本身不会替换程序。设置 `CYEAM_CLI_NO_UPDATE_NOTIFIER=1` 关闭。
  `cyeam update` 是显式升级命令，会下载并校验最新 Release、替换当前程序，成功后还会尝试通过
  `npx skills add` 同步 skills；需要保持固定版本时不要运行它，按上述步骤安装选定版本。

关闭 CLI 统计和更新提示（业务本身的网络请求不受影响），可在当前 shell 设置：

```bash
export CYEAM_CLI_NO_ANALYTICS=1
export CYEAM_CLI_NO_UPDATE_NOTIFIER=1
```

## 命令

```
cyeam whoami                         查看登录状态（登录方式见 cyeam-cli skill）

cyeam cnote list                     列出笔记（显示文件名、修改时间、可点击链接）
cyeam cnote get <title>              读取笔记内容（默认 markdown 渲染）
cyeam cnote get <title> --format text
cyeam cnote new <title> < note.html  从 stdin 创建笔记
cyeam cnote append <title> < note.html
```

## 数据存储

笔记存储在 OneDrive 的 `Notes/` 文件夹下，每篇笔记是一个 `.html` 文件。列表时如果有 `webUrl` 会显示可点击的"打开"链接。

## 注意事项

- 需要 Microsoft 账号登录，使用 Device Code 流程（登录方式见 cyeam-cli skill）
- 登录后 token 自动刷新（带 refresh_token）
- 内容是纯 HTML，`get` 命令会解析并转换为 markdown/text 显示
- `new` 和 `append` 从 stdin 读取，适合管道：`echo "<h1>hello</h1>" | cyeam cnote new test`