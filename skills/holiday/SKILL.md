---
name: holiday
version: 0.1.16
description: 假期查询——查某一天是上班还是放假、调休安排、节假日名称。想知道是不是工作日、什么时候放假补班的时候用。【重要】必须先读 skill 原文获取正确命令格式，禁止瞎猜。
---

# 假期查询

## 概述

查任意日期的节假日信息：是否放假、节日名称、调休补班、薪资倍数等。数据来自 timor.tech 公共 API。

```bash
cyeam date holiday                  # 今天
cyeam date holiday 2026-10-01       # 指定日期
cyeam date holiday next-monday       # 不支持，只接受 YYYY-MM-DD
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
- **假期查询**：向 `https://timor.tech/api/holiday/year/<年份>?type=Y&weekday=Y`
  请求指定年份的节假日，再在本地计算目标日期的状态。不上传本地文件，不需要登录。
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

```
日期: 2026-10-01
星期: 周四
状态: 休息日
名称: 国庆节
薪资倍数: 3
```

## 状态说明

| 状态 | 含义 |
|------|------|
| 工作日 | 普通上班日 |
| 休息日 | 法定假日或周末 |
| 周末休息 | 周末 |
| 调休补班 | 调休安排的上班日（如国庆前的周末） |

## 注意事项

- 日期格式必须为 `YYYY-MM-DD`，省略参数默认为今天
- 不需要登录
- 数据档期：当前仅支持已公布的年度节假日安排，未公布的年份会返回空