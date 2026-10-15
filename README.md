# cyeam-cli

cyeam 命令行工具，提供架构咨询、日期查询、路书分享、OneDrive 云笔记、书法字形处理等功能。

## 安装

### 首次安装

从 GitHub Releases 下载最新版本：

```bash
# macOS (Apple Silicon)
curl -L https://github.com/mnhkahn/cyeam-cli/releases/latest/download/cyeam_Darwin_arm64.tar.gz | tar xz
chmod +x cyeam
sudo mv cyeam /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/mnhkahn/cyeam-cli/releases/latest/download/cyeam_Darwin_x86_64.tar.gz | tar xz
chmod +x cyeam
sudo mv cyeam /usr/local/bin/

# Linux (amd64)
curl -L https://github.com/mnhkahn/cyeam-cli/releases/latest/download/cyeam_Linux_x86_64.tar.gz | tar xz
chmod +x cyeam
sudo mv cyeam /usr/local/bin/

# Windows
# 下载 cyeam_Windows_x86_64.zip 并解压，将 cyeam.exe 添加到 PATH
```

### 更新

```bash
cyeam update
```

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

### 书法字形（Mo）

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

# OCR 行书图片
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
