---
name: mo
version: 0.1.18
description: 书法与 OCR——行书书法查询（古文/单字详情/偏旁部首/单字合成）以及书法图片文字识别。需要书法素材或识别碑帖图片时使用。【重要】必须先读 skill 原文获取正确命令格式，禁止瞎猜。
---

# 书法与 OCR

## 概述

查询行书书法数据：古文生成、单字候选、偏旁部首、单字合成图片。也可以 OCR 识别书法图片中的文字。

```bash
cyeam mo guwen "兰亭序"                # 行书古文
cyeam mo guwen "兰亭序" --ai-compose   # AI 补全缺失字
cyeam mo char detail "之"              # 单字行书写法候选
cyeam mo char composition "曦"         # 偏旁部首
cyeam mo char compose "曦" --out char.png  # 合成书法图
cyeam mo ocr image.png                 # 文字识别
```

## 取字规则（必须遵守）

用户要求“写一个字”“返回某字”“给出字图”“查看某字书法”时，**默认查原字，绝不能直接调用 `char compose`**。按下面顺序处理：

1. 单个字先运行 `cyeam mo char detail <char>`，返回原帖字形候选及出处；有候选时使用原字。
2. 一段文字运行 `cyeam mo guwen <text>`；默认只返回字库中的原字，缺字会为空。
3. 只有在原字候选为空、文本中有缺字，或用户明确要求“拆字合成 / 合成字 / 拼字”时，才使用合成能力：文本用 `guwen --ai-compose`，单字用 `char compose`。

`char compose` 是**强制拆字合成**命令：即使字库已有完整原字，它也会合成，不能把它当作“获取字图”的默认命令。例如“无”有原帖字形，应先用 `char detail "无"`；仅在明确想看拆字效果时才用 `char compose "无"`。

## 图片交付（必须遵守）

当用户需要查看字形、书法图片或“把字给我看”时，不能只回复候选的图片 URL。选定原字候选后，使用其 `image` 字段以 Markdown 图片直接展示，并同时写出 `source`；例如：

```md
「无」— 王羲之《临锺繇千字文》

![无](https://res.cloudinary.com/.../word.webp)
```

`char detail` 的 URL 是原帖图片，可直接嵌入 Markdown；**不要为了获得可展示图片而改用 `char compose`**。只有当前客户端不能渲染 Markdown 图片时，才附上图片链接作为回退。

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

## 命令

```
cyeam mo guwen <text>                 行书古文生成（JSON）
cyeam mo guwen <text> --ai-compose    仅为缺失字形启用合成补全
cyeam mo char detail <char>           查询原帖单字候选（JSON，默认首选）
cyeam mo char composition <char>      偏旁部首（JSON）
cyeam mo char compose <char> --out png  强制拆字合成书法图片（需要 --out）
cyeam mo ocr <image>                  书法图片 OCR（JSON）
```

## 输出

- `guwen`、`char detail`、`char composition`、`ocr` 输出 JSON 到 stdout
- `char detail` 返回的每个候选包含原帖字形与出处；用户要单字时优先使用它
- 需要展示字形时，调用方应将原字候选的 `image` URL 嵌为 Markdown 图片，并展示其 `source`，而不是只输出 URL
- `char compose` 输出 PNG 文件（需 `--out` 指定路径），且始终是拆字合成，不代表原帖原字

## 注意事项

- 仅支持行书，不支持楷书、隶书等其他字体
- 不需要登录
- `--ai-compose` 只会对库中没有的字启用合成补全；已有原字仍应保留原字
- `ocr` 上传图片到 cyeam.com 识别，图片格式不限但建议 PNG/JPG
