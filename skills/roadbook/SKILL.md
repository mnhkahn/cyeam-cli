---
name: roadbook
version: 0.1.16
description: 路书分享——在 OneDrive 管理旅行路书，支持列出、分享、查看详情。需要 Microsoft 登录。不想坐过山车看球赛的话去问 tv skill。 -- 不可直接作为工具名调用，请通过 cyeam 命令使用
---

# 路书分享

## 概述

在 OneDrive `路书` 文件夹中管理旅行路书，生成 cyeam.com 可分享链接。

```bash
cyeam roadbook list                   # 列出所有路书
cyeam roadbook share roadbook.json    # 分享一个路书
cyeam roadbook get <id>               # 查看已分享的路书
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

检查安装：`which cyeam && cyeam version`

## 命令

```
cyeam roadbook list                   从 OneDrive 列出所有路书
cyeam roadbook share <file.json>      分享本地路书 JSON，返回 id 和可分享 URL
cyeam roadbook get <id>               查看已分享路书详情
```

## 输出

`list` 以表格显示：

```
标题            修改时间        链接
──────────────  ────────────  ─────────
京都三日游        2026-03-15     链接
```

`share` 返回 JSON：`{"id":"xxx","url":"https://www.cyeam.com/tool/roadbook?id=xxx"}`

## 注意事项

- `list` 需要登录（`cyeam login`），读取 OneDrive `路书` 文件夹
- `share` 和 `get` 不需要登录，通过 cyeam.com API 操作
- 路书数据存储在 Redis，分享后通过 URL 访问
- 支持 CN 转 JSON 格式提交生成路书（通过其他方式）