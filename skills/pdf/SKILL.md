---
name: pdf
version: 0.1.17
description: Markdown/HTML/Typst 转 PDF——输入 Markdown、HTML 或 Typst 文件，本地生成 PDF 文档。支持标题/列表/代码块/多列/自由排版及口算练习单。【重要】必须先读 skill 原文获取正确命令格式，禁止瞎猜。
---

# Markdown/HTML/Typst 转 PDF

## 概述

把 Markdown、HTML 或 Typst 文件转成 PDF 文档。转换过程在本地完成；CLI 默认统计和更新提示的网络请求及关闭方式见下方安装说明。

支持三种模式：`--mode auto`（默认，普通文档）、`--mode arithmetic`（口算练习单）、`--mode pinyin`（看拼音写字）。后两种模式使用纯文本输入并直接由 CLI 排版。

```bash
cyeam pdf README.md                      # 从 Markdown 文件生成 PDF（base64 JSON）
cyeam pdf README.md -o out.pdf           # 从 Markdown 文件生成并保存 PDF
cyeam pdf layout.typ -o out.pdf          # 从 Typst 文件生成并保存 PDF（需要本机 typst）
cat index.html | cyeam pdf               # 从 stdin 读 HTML，输出 base64 JSON
cat README.md | cyeam pdf -o out.pdf     # 从 stdin 读 Markdown 并保存 PDF
```

## 输入方式选择（重要）

`cyeam pdf` 支持多种输入方式。根据当前执行环境选择最稳的方式，不要瞎猜命令格式。

## 排版格式选择（重要）

- 默认优先生成 Markdown：适合普通文档、说明、列表、代码块等线性内容。
- 当用户明确要求紧凑、节省空间、排满页面、多列展示、自由排版、版面更好看时，不要只生成普通 Markdown 单栏内容。
- 轻量多列仍可用 Markdown，但要使用 `cyeam pdf` 支持的 columns 扩展语法。
- 需要更自由的布局（多列、网格、明确换列、页面级排版）时，优先生成 Typst 文件（`.typ`），再执行 `cyeam pdf <file.typ> -o <file.pdf>`。
- 不要用 HTML/CSS 实现紧凑或多列排版；当前 HTML 输入只适合简单结构转换，CSS 不作为 PDF 布局引擎执行。

### Markdown 多列扩展

适合仍想保留 Markdown 书写体验、但需要让短内容更紧凑的场景。语法：

```markdown
::: columns 3
::: column
第一列内容
:::

::: column
第二列内容
:::

::: column
第三列内容
:::
:::
```

规则：

- `columns` 后面的数字表示建议列数，但实际列数以 `column` 块数量为准。
- 每个 `column` 内仍写普通 Markdown。
- 只有用户要求紧凑、节省空间、多列、排满页面等版面目标时才使用；普通文档不要滥用。

### Typst 自由排版

适合用户要求自由排版、明确多列、网格、卡片式布局或更高页面利用率时。示例：

```typst
#set page(paper: "a4", margin: 18mm)
#set text(size: 11pt)

#columns(3, gutter: 12pt)[
  第一列内容

  #colbreak()

  第二列内容

  #colbreak()

  第三列内容
]
```

Typst 文件必须以 `.typ` 保存后执行：

```bash
cyeam pdf /tmp/layout.typ -o /tmp/layout.pdf --pretty
```

如果本机没有安装 `typst`，命令会报错。此时改用 Markdown columns 扩展，或提示用户安装 Typst。

### 口算练习格式

CLI 原生支持 `--mode arithmetic`，无需 Typst。输入为 UTF-8 纯文本，每个非空行一道题（可带编号），空行忽略；不要混入 Markdown 标题、元信息、表格或答案。CLI 只排版，不自动出题、编号或补答案。

```bash
cyeam pdf questions.txt --mode arithmetic -o practice.pdf
cyeam pdf questions.txt --mode arithmetic --layout grade4 --title "四年级口算" -o practice.pdf
cyeam pdf questions.txt --mode arithmetic --layout vertical -o practice.pdf
```

`--layout` 可选 `standard`（默认）、`grade4`、`large-number`、`negative`、`quantity`、`vertical`，对应下表；`--title` 默认“宝宝口算”。这两个参数仅用于 arithmetic 模式。默认 `--mode auto` 保留 Markdown/HTML/Typst 自动检测；arithmetic 模式优先于扩展名，始终按逐行题目解析，支持文件或 stdin 输入及原有 base64 JSON 输出。

用户要求“口算 PDF”“口算练习单”或“按宝宝口算格式排版”时使用。参考 `cyeam_web/controllers/baobao_controller.go` 的 `ArithmeticExec()`，以 PDF 后端布局为准，不采用前端 HTML 打印逻辑写死的 100 题。

- 使用 A4 纵向（210 × 297 mm），黑白、无边框题目网格；题目区左侧 10 mm，总宽 190 mm，从页面顶部 32 mm 开始。页首分隔线位于 30 mm，横跨 5–205 mm。
- 页首默认标题“宝宝口算”，20 pt 加粗；四年级使用具体题型标题，13 pt 加粗。标题左上位置参考 x=10 mm、y=7 mm。网站普通版标题右侧放“检查题目、检查十位、检查个位、检查答案”四项简短提示（8–10 pt），四年级版不放这组提示。CLI 原生模式使用普通字重标题，不附检查提示、二维码或姓名日期栏；需要完整复刻这些装饰时使用自定义 Typst。
- 复刻网站版时，右上角 x=170 mm、y=7 mm 放 20 × 20 mm 二维码，指向 `https://www.cyeam.com/tool/arithmetic`。自定义练习单可省略二维码；来自 arithmetic skill 的标题、姓名、日期等元信息应保留，并相应调整题目起始位置与每页行数，避免重叠。

| 题型 | 每页行 × 列 | 每页容量 | 行高 | 列宽 | 题目字号 |
| --- | --- | --- | --- | --- | --- |
| 普通横式口算 | 20 × 5 | 100 题 | 12 mm | 38 mm | 11.5 pt |
| 四年级一般题型 | 25 × 4 | 100 题 | 10 mm | 47.5 mm | 9 pt |
| 四年级大数 `g4-large-number`、负数 `g4-negative` | 25 × 3 | 75 题 | 10 mm | 190/3 mm | 10 pt |
| 四年级数量关系 `g4-quantity` | 25 × 2 | 50 题 | 10 mm | 95 mm | 10 pt |
| 非四年级竖式（网站题型标识以 `-2` 结尾） | 5 × 5 | 25 题 | 48 mm | 38 mm | 11.5 pt |

- 按从左到右、从上到下的顺序填充网格。横式左对齐、垂直居中；竖式题干置于格子顶部，下面保留演算空间。字体参考黑体，实际使用可用且覆盖中文与运算符的字体。
- 四年级普通填空留 `（　　）`，数量关系留更宽的 `（　　　　）`；保留单位与完整题干。长题放不下时优先增加行高或减少列数，并重新计算每页容量，不截断题干。
- 表中题数是参考版式的每页容量。用户指定题数或已提供题目时，按实际数量分页，末页保留空白，不为填满页面擅自补题。只有用户要求满页且未指定题数时，按对应容量出题；网站生成器也可能因题库或去重限制返回少于容量的题目。
- 题目内容与答案分离，默认不附答案；需要答案时另起答案页。需要出题时遵循 arithmetic skill 的年级、题型和数量约束；本格式只规定 PDF 排版，不扩大支持的题型范围。
- 默认直接使用 CLI 口算模式，按对应容量分页，末页不补题。长题在格内自动换行，超出行高时报错并指出题号；此时缩短题干或选用更宽的布局。竖式模式在顶部放横式题干，下面留演算空白，不自动生成纵向算式。需要额外元信息、自定义行高或纵向算式时改用 Typst；本节毫米尺寸覆盖上方通用示例的 18 mm 页边距。

### 看拼音写字格式

用户要求“看拼音写字 PDF”“拼音米字格练习纸”时，使用 `--mode pinyin`。输入是要练习的汉字原文（UTF-8），不是拼音；文件或 stdin 均可。无需 Typst、网络或登录。

```bash
printf '%s' '你好' | cyeam pdf --mode pinyin -o practice.pdf
cyeam pdf words.txt --mode pinyin -o practice.pdf
```

- 参考网站 `/tool/pinyin?xiezi=你好`：A4 纵向，左上标题“看拼音写字”，拼音提示下面是空白米字格，不显示汉字答案。
- 题区从 x=13 mm、y=20 mm 开始，米字格 11 × 11 mm，拼音区高 7 mm，词组间隔 5 mm，每行最多 17 格；空格、换行、标点和其他非汉字分隔词组，不占格。超过 17 字的词组自动拆分。
- 底部保留“改错：”及 255、265、275、285 mm 处的四条横线。长文本自动分页，每页重复标题和改错区，不静默截断；分组间距可能让每页容纳字数少于 204 字。
- 拼音由本地词典生成，多音字读音应按语境核对。不含汉字时返回错误。
- 拼音字体内置；标题与改错标签沿用 PDF 中文字体检测，需要系统提供可用中文字体。`--layout`、`--title` 仅用于口算模式，不能与 pinyin 模式混用。
- 与其它 PDF 模式相同，`-o` 保存文件，省略 `-o` 返回 base64 JSON。位置参数始终是文件路径；直接传文字请使用 stdin，或已有的 `cyeam pinyin sheet "你好" -o practice.pdf`。

### 方式一：文件输入（推荐给 Agent）

适合已经有 Markdown/HTML 文件，或 Agent 可以先把内容写入临时文件的场景。结构化 `argv` 工具、普通 shell、Xiaoli 的 `channel_send` 后续发送都适合这种方式。

```bash
# 1. 先准备真实源文件，例如 /tmp/openclaw-install-guide.md
# 2. 再把源文件转成 PDF
cyeam pdf /tmp/openclaw-install-guide.md -o /tmp/openclaw-install-guide.pdf --pretty
```

### 方式二：stdin 输入

适合执行环境能向命令 stdin 写入正文的场景。正文可以是 Markdown 或 HTML。

```bash
cyeam pdf -o /tmp/openclaw-install-guide.pdf --pretty
```

注意：上面这种写法只有在 stdin 已经提供内容时才正确；如果没有 stdin 内容，会失败并提示 `no content provided (stdin is empty)`。

### 方式三：shell 管道输入

适合人类 shell，或明确支持 shell 管道的 Agent/bash 工具。

```bash
cat /tmp/openclaw-install-guide.md | cyeam pdf -o /tmp/openclaw-install-guide.pdf --pretty
printf %s "# 标题\n\n正文" | cyeam pdf -o /tmp/out.pdf --pretty
```

如果当前工具只接受结构化 `argv`、不提供 stdin，也不支持 shell 管道，请使用方式一：先创建源文件，再执行 `cyeam pdf <source> -o <pdf> --pretty`。

生成给用户的 PDF 后，继续调用当前渠道的发送工具（如 `channel_send`）发送生成的 `.pdf` 文件。

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
- **PDF 生成**：读取用户指定的本地文件或 stdin，转换过程不上传文档；
  `--out` / `-o` 写入指定路径，已有文件可能被覆盖，不需要登录。
  Typst 输入会调用已安装的 `typst compile`，并读写临时目录；
  文档引用的 Typst 包可能触发 Typst 自身的包下载，需要离线时使用本地依赖。
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

`cyeam pdf README.md` 返回 JSON：

```json
{"ok":true,"data":"{\"pdf\":\"<base64>\"}"}
```

加 `--out` 直接保存文件：

```bash
cyeam pdf README.md -o out.pdf --pretty
# saved: out.pdf
```

不带 `--pretty` 时输出 JSON 信封：`{"ok":true,"data":"saved: out.pdf\n"}`

## 注意事项

- 格式自动检测：`.typ` 后缀视为 Typst；`.html`/`.htm` 后缀或内容以 `<!DOCTYPE`/`<html` 开头视为 HTML；其余视为 Markdown
- 纯本地生成，使用 goldmark + gofpdf
- 支持排版要素：H1-H4 标题、段落、粗体/斜体、行内代码、代码块、无序/有序列表、链接、水平线
- Markdown 额外支持 `::: columns` / `::: column` 多列扩展
- Typst 输入需要本机已安装 `typst`
- 自动分页，A4 纸张
- 不需要登录
