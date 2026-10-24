---
name: roadbook
version: 0.1.16
description: 路书分享——需要先 cyeam login（OneDrive），支持 CSV 文本生成路书列表、查看详情。不想坐过山车看球赛的话去问 tv skill。【重要】必须先读 skill 原文获取正确命令格式，禁止瞎猜。
---

# 路书分享

## 用户意图 → 执行路径速查

| 用户说 | 路径 | 是否需要登录 |
|---|---|---|
| "生成路书链接" / "帮我规划路线" / 分享旅行计划 | → 走 **生成路书流程**（步骤A） | 需要 `cyeam login` |
| "查看路书列表" | → 走 **列出路书流程**（步骤B） | 需要 `cyeam login` |
| "登录" / "登录路书" | → 走 **登录流程**（步骤C） | — |
| "查看某个路书详情" | → 走 **查看路书流程**（步骤D） | 不需要 |

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
- **OneDrive 路书**：`list` 通过 `https://graph.microsoft.com/v1.0` 读取 `路书/`
  文件夹及文件内容以获得标题；Microsoft 的文件下载可能重定向到其存储域名。
  `csv` 从 stdin 读取行程，把生成的 JSON 写入该文件夹，然后向 cyeam.com 分享。
- **路书分享和读取**：`share` 读取用户指定的本地 JSON，`csv` 使用生成的 JSON，向
  `https://www.cyeam.com/api/roadbook/share` 上传行程以生成链接；`get` 向
  `https://www.cyeam.com/api/roadbook/get?id=<ID>` 查询已分享内容。
  `share/get` 无需 Microsoft 登录；生成的链接用于让持有链接的人查看行程，
  仅在用户要求分享时执行上传。
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
cyeam roadbook list                   从 OneDrive 列出所有路书
cyeam roadbook share <file.json>      分享本地路书 JSON，返回 id 和可分享 URL
cyeam roadbook get <id>               查看已分享路书详情
cyeam roadbook csv                   从 stdin 读取 CSV 文本生成路书（需要 cyeam login）
```

## 输出

`list` 以表格显示：

```
标题            修改时间        链接
──────────────  ────────────  ─────────
京都三日游        2026-03-15     链接
```

`share` 和 `csv` 返回 JSON：`{"id":"xxx","url":"https://www.cyeam.com/tool/roadbook?id=xxx"}`

---
## ⚠️ 执行流程（模型请严格按步骤执行）

### 步骤 A：生成路书链接（用户说"生成路书" / 分享旅行计划时使用）

直接按以下步骤执行：

**A1. 检查登录**
```bash
cyeam whoami
```
如果输出包含 "Not logged in"，先走步骤 C 登录，再继续。

**A2. 整理行程信息**
如果用户已提供完整行程（地点、天数、顺序），直接跳到 A3 整理 CSV。信息不足时只问缺失项。**不要问多余问题。**

**A3. 整理 CSV**
每行一个地点，5列逗号分隔：`名称,地址,类型,日期,备注`
- 类型：景点/餐饮/住宿/起点/其他
- 日期：Day1 / Day2... 或 5.1 / 5.2...

**A4. 执行命令**
用 `printf` 管道传给 `cyeam roadbook csv`，**不要创建临时文件**：

```bash
printf "%s\n" \
  "故宫博物院,北京市东城区景山前街4号,景点,Day1,上午" \
  "国家博物馆,北京市东城区东长安街16号,景点,Day1,下午" \
  | cyeam roadbook csv --title "北京行" --pretty
```

命令会依次做三件事：
1. 把路书 JSON 保存到 OneDrive `路书` 文件夹
2. 调 `/api/roadbook/share` 接口生成短分享链接
3. 返回 JSON 结果

**A5. 返回结果**
输出包含 `url` 字段，直接回复用户：
> 路书已生成，点击链接查看完整行程地图：https://www.cyeam.com/tool/roadbook?id=xxx

**A6. 检查输出**
- 如果报 "Not logged in" 错误，走步骤 C 登录
- 如果报其他错误，把错误信息告诉用户

---

### 步骤 B：列出路书（用户说"列出路书" / "我的路书"）

**B1. 检查登录状态**
```bash
cyeam whoami
```
如果输出包含 "Not logged in"，走步骤 C 让用户先登录。

**B2. 列出路书**
```bash
cyeam roadbook list --pretty
```

**B3. 返回给用户**
以表格形式展示标题、修改时间。

---

### 步骤 C：登录（用户说"登录"或在 B1 中检测到未登录）

**C1. 执行登录**

```bash
cyeam login
```

命令会打印授权链接和验证码：有桌面环境时自动尝试打开浏览器；在远程服务器（SSH 等无桌面环境）浏览器会静默失败，按打印的链接和验证码在任意浏览器完成授权即可，CLI 自动获取 token。

**C2. 告知用户**
告诉用户：
> 请在浏览器中访问显示链接（https://microsoft.com/devicelogin），输入验证码完成登录。登录后回到这里继续。

**C3. 运行命令并等待**
运行命令后，CLI 会等待用户在浏览器完成授权。等待命令输出结果。

**C4. 验证登录**
```bash
cyeam whoami
```
确认显示 "logged in" 状态。

---

### 步骤 D：查看路书详情（用户说"查看某个路书"）

直接执行：
```bash
cyeam roadbook get <id> --pretty
```
返回 JSON 格式的路书数据。把内容展示给用户。

---

## 注意事项

- `csv`、`list` 需要 `cyeam login`（Microsoft OneDrive 登录）
- `get`、`share` 不需要登录，通过 cyeam.com API 操作
- `csv` 命令流程：登录 → 解析 CSV → 存 OneDrive `路书` 文件夹 → 调 share API → 返回短链接
- 路书数据存储在 OneDrive `路书` 文件夹 + Redis 分享快照（30天有效）