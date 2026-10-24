# cyeam-cli

cyeam 命令行工具，提供架构咨询、日期查询、路书分享、OneDrive 云笔记、书法字形处理等功能。

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

### Windows

从 [v1.1.4 发布页](https://github.com/mnhkahn/cyeam-cli/releases/tag/v1.1.4) 下载对应架构的 ZIP，
解压前用 PowerShell `Get-FileHash -Algorithm SHA256 <下载文件路径>` 核对摘要：

| 文件 | SHA-256 |
|---|---|
| `cyeam_Windows_x86_64.zip` | `4d7fab8f5ffac5510e18d8eb0d69ae85937e3537c8e8e2f9cf48389296ec9696` |
| `cyeam_Windows_arm64.zip` | `ef958f9733334448a686ce0836bd9641ccc9edb70719024c85d17ee88386305c` |

摘要不一致时停止。校验成功后解压到用户目录，将其加入用户 PATH；无需管理员权限。

## 使用

### 使用统计

CLI 内置 PostHog 项目令牌，安装后无需配置账号或密钥。每次执行有效命令
（包括帮助页面）会尝试直接向 `https://us.i.posthog.com` 发送一条
`cli_command_started` 事件。后台可按以下属性查看统计：

| 属性 | 含义 / 示例 |
|---|---|
| `command_path` | 完整命令路径，如 `cyeam mo char detail` |
| `command` / `subcommand` | 主命令 `mo` / 子命令 `char detail` |
| `cli_version` | CLI 发布版本（本地开发构建为 `dev`） |
| `$os` / `$os_version` | 系统名称及版本 |
| `$os_distro` | Linux 发行版名称（如可获取） |
| `architecture` | 程序运行架构 `arm64` / `amd64` |
| `$go_version` | 构建使用的 Go 版本 |
| `$geoip_country_name` 等 | PostHog 根据网络出口解析的国家、省份、城市 |

每个安装使用随机编号（`~/.cyeam/analytics-id`），用于估算活跃安装数；
它不来自用户名、邮箱或硬件序列号。若无法保存编号，则使用临时随机编号。
不会发送命令参数、参数值、文件内容、环境变量值或登录凭证。
参数校验未通过的调用和 shell 自动补全不计入命令事件。

事件在后台发送，退出前最多等待约 3 秒完成发送；网络失败静默丢弃，
不修改命令输出或退出状态。地域可能不完整，代理 / VPN 会影响地域结果。
使用统计需要网络连接，不保证每次执行都能成功记录。

关闭本次统计：

```bash
cyeam --no-analytics version
```

关闭当前 shell 及其后续子进程的统计（也可将此设置加入 shell 配置）：

```bash
export CYEAM_CLI_NO_ANALYTICS=1
```

设置 `DO_NOT_TRACK=1` 或 `CI=true` 也会关闭统计。新增命令自动继承根命令的
统计钩子；如需自定义 `PersistentPreRunE`，应保留根命令的统计逻辑。

### 比赛与转播预报

```bash
cyeam tv today
cyeam tv list --league nba --days 7
cyeam tv list --league cn-football --from 2026-03-01 --to 2026-03-21 --include-finished
cyeam tv next --league nba --team 湖人 --json
```

支持 NBA、男女足世界杯和中国男女足国家队。默认按北京时间显示；
`--days N` 从当天起查询 N 个自然日，`--from` / `--to` 可指定日期范围。
NBA 官方赛程源不可用时自动尝试 ESPN；ESPN 按单日查询并合并去重。

查询完整成功时，`--json` 的 `data` 保持比赛数组（没有比赛为 `[]`）。
部分请求失败且仍有符合条件的比赛时，`data` 为
`{"matches": [...], "partial": true, "warnings": [...]}`，每条警告包含 `league` 和 `message`。
没有符合条件的比赛且存在请求失败时，命令返回非零退出码和 `ok: false` 错误，
不会把数据源失败报告成“暂无比赛”。转播信息来自内置规则，实际播出以平台为准。

### 账号

```bash
# 登录 Microsoft 账号，用于 OneDrive 路书和云笔记
# 登录后会使用 refresh token 自动刷新访问令牌
cyeam login

# 查看当前登录状态
cyeam whoami

# 退出登录
cyeam logout
```

### 架构咨询

```bash
# 快速模式（默认）
cyeam ask "如何设计微服务架构？"

# 深度思考模式
cyeam ask --mode think "如何设计微服务架构？"

# 专家模式
cyeam ask --mode expert "如何设计微服务架构？"

# 搜索 cyeam.com
cyeam ask search "微服务"
```

### 日期查询

```bash
# 获取今日节日
cyeam date holiday

# 获取指定日期节日
cyeam date holiday 2024-01-01
```

### 路书分享

```bash
# 列出 OneDrive 路书
cyeam roadbook list

# 分享路书
cyeam roadbook share route.json

# 获取路书
cyeam roadbook get <id>
```

### Calligraphy Composition / Seal Script OCR

```bash
# 生成行书古文字形数据
cyeam mo guwen "永"
cyeam mo guwen --ai-compose "永"  # 启用 AI 合成缺失字形

# 获取行书字形候选
cyeam mo char detail "永"

# 获取字形构成
cyeam mo char composition "永"

# 合成行书字符图片
cyeam mo char compose "永" --out yong.png

# OCR 篆字图片
cyeam mo ocr calligraphy.png
```

### CNote 云笔记

```bash
# 列出 OneDrive Notes 目录下的笔记，并显示可点击的打开链接
cyeam cnote list

# 读取笔记详情，默认输出 Markdown 风格文本
cyeam cnote get "日记"

# 读取笔记详情，输出纯文本
cyeam cnote get "日记" --format text

# 新建笔记，内容从 stdin 读取
cyeam cnote new "日记" < note.html

# 追加笔记内容
cyeam cnote append "日记" < more.html
```

### 邮件

```bash
# 将指定邮箱 INBOX 中的全部未读邮件一键标为已读
cyeam mail mark-all-read <account>
```

### 英语音标

```bash
# 查询英语单词的英式/美式音标和简明释义
cyeam phonetic hello

# 人类可读输出
cyeam --pretty phonetic hello
```

### Git 版本标签

在目标 Git 仓库内执行：

```bash
cyeam tag minor --dry-run  # 预览小版本升级
cyeam tag minor            # 小版本升级：v0.2.15 -> v0.3.0
cyeam tag major            # 大版本升级：v0.2.15 -> v1.0.0
```

安装项目 skills 后，可显式调用 `git-tag` 技能：`/git-tag` 默认小版本升级，
`/git-tag major` 大版本升级，`/git-tag minor --dry-run` 预览；使用 `$` 调用技能的客户端可写 `$git-tag`。
显式调用会直接执行，仍保留工作区和标签检查。`git-cmsg` 会将仅打标签的请求交给该技能。

以本地最大的稳定版本标签 `vMAJOR.MINOR.PATCH` 为基准，忽略预发布及其他格式标签；
没有匹配标签时从 `v0.0.0` 开始。小版本增加次版本号，大版本增加主版本号，并清零后续位。
工作区必须干净（包括暂存和未跟踪文件），标签指向当前 HEAD。
命令只创建本地标签，不自动拉取或推送；如需同步远端标签，请先运行 `git fetch --tags`。
确认后使用 `git push origin <新标签>` 发布，在本项目中会触发 GitHub Release 工作流。

### 版本和更新

```bash
# 查看版本
cyeam version

# 更新工具
cyeam update
```

### 作业批改结果提交

```bash
cyeam trello card review <原卡片ID> --text '第3题需要订正：注意进位。'
cyeam trello card review <原卡片ID> --image correction.webp
cyeam trello card review <原卡片ID> --audio correction.m4a
```

支持文字、图片、音频三选一，按协议 v1 在原卡片发布批改评论并保留历史。加 `--has-errors=true` 转入「待改错」，`--has-errors=false` 批改通过并保留「已改错」；省略则只发布内容。作业状态统一为待完成、已完成、待改错、已改错。
失败后使用输出的 `review_id`，携带相同内容和 `--review-id <UUID>` 重试。
完整说明见 [作业批改命令](docs/trello-review.md)。
