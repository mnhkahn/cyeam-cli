---
name: pdf
version: 0.1.17
description: Markdown/HTML/Typst 转 PDF——输入 Markdown、HTML 或 Typst 文件，本地生成 PDF 文档。支持标题/列表/代码块/多列/自由排版。【重要】必须先读 skill 原文获取正确命令格式，禁止瞎猜。
---

# Markdown/HTML/Typst 转 PDF

## 概述

把 Markdown、HTML 或 Typst 文件转成 PDF 文档。全部本地生成，无需网络。

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

```bash
# macOS Apple Silicon
curl -L https://github.com/mnhkahn/cyeam-cli/releases/latest/download/cyeam_Darwin_arm64.tar.gz | tar xz && chmod +x cyeam && sudo mv cyeam /usr/local/bin/

# macOS Intel
curl -L https://github.com/mnhkahn/cyeam-cli/releases/latest/download/cyeam_Darwin_x86_64.tar.gz | tar xz && chmod +x cyeam && sudo mv cyeam /usr/local/bin/

# Linux amd64
curl -L https://github.com/mnhkahn/cyeam-cli/releases/latest/download/cyeam_Linux_x86_64.tar.gz | tar xz && chmod +x cyeam && sudo mv cyeam /usr/local/bin/
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
