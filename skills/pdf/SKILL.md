---
name: pdf
version: 0.1.16
description: Markdown/HTML 转 PDF——输入 Markdown 或 HTML 文件，本地生成 PDF 文档。支持标题/列表/代码块/粗斜体等排版。 -- 不可直接作为工具名调用，请通过 cyeam 命令使用
---

# Markdown/HTML 转 PDF

## 概述

把 Markdown 或 HTML 文件转成 PDF 文档。全部本地生成，无需网络。

```bash
cyeam pdf README.md                      # 生成 PDF（base64 JSON）
cyeam pdf README.md -o out.pdf           # 保存 PDF 到文件
cat index.html | cyeam pdf               # 从 stdin 读 HTML
cat README.md | cyeam pdf -o out.pdf     # 从 stdin 读并保存
```

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

- 格式自动检测：`.html`/`.htm` 后缀或内容以 `<!DOCTYPE`/`<html` 开头视为 HTML，其余视为 Markdown
- 纯本地生成，使用 goldmark + gofpdf
- 支持排版要素：H1-H4 标题、段落、粗体/斜体、行内代码、代码块、无序/有序列表、链接、水平线
- 自动分页，A4 纸张
- 不需要登录