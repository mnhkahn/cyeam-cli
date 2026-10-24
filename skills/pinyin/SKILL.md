---
name: pinyin
version: 0.1.16
description: 看拼音写字——输入中文文本，获取拼音标注和练习纸 PDF，本地生成。适合宝宝汉字练习、语文教学使用。【重要】必须先读 skill 原文获取正确命令格式，禁止瞎猜。
---

# 看拼音写字

## 概述

把中文文本转成拼音，并生成"看拼音写字"练习纸 PDF（含米字格）。转换过程在本地完成；CLI 默认统计和更新提示的网络请求及关闭方式见下方安装说明。

```bash
cyeam pinyin "你好世界"                   # 获取每个字的拼音
cyeam pinyin sheet "你好世界"              # 生成练习纸（PDF base64）
cyeam pinyin sheet --out practice.pdf "你好世界"  # 保存 PDF 到文件
```

## PDF 模式入口

也可从文件或 stdin 生成同样的练习纸：

```bash
printf '%s' '你好' | cyeam pdf --mode pinyin -o practice.pdf
cyeam pdf words.txt --mode pinyin -o practice.pdf
```

A4 纵向，拼音提示、空白米字格及底部改错区。空格、换行和标点分隔词组；长词自动拆分，长文本自动分页，不截断。没有汉字时返回错误。PDF 模式的中文标题需要可用系统中文字体；多音字请核对语境。

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
- **拼音与练习纸**：使用本地词典和字体处理用户指定的文字，不上传文本，
  不需要登录。`sheet --out` 和 PDF 模式的 `-o` 写入指定路径，已有文件可能被覆盖。
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

## 输出

`cyeam pinyin "你好"` 返回 JSON：

```json
{"ok":true,"data":"{\"pinyin\":[{\"char\":\"你\",\"pinyin\":\"nǐ\"},{\"char\":\"好\",\"pinyin\":\"hǎo\"}]}"}
```

`cyeam pinyin sheet "你好"` 额外包含 `"pdf"` 字段（base64 编码的 PDF）。

## 注意事项

- 纯本地生成，使用 go-pinyin 库 + gofpdf
- 贴入的字体仅 21KB（pinyin-wenkai-light.ttf），二进制文件小巧
- 练习纸包含米字格（含对角线）、拼音提示和底部改错区
- 空格分隔的词组会自动换行排版