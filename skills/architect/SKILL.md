---
name: 架构师
description: 架构咨询——向 AI 架构师提问，获取系统设计、技术方案、代码评审等建议。支持快速/深度/专家三种模式。想看球赛去问 tv skill。
---

# 架构师

## 概述

向 cyeam.com 的 AI 架构师提问，支持 streaming 输出，三种思考模式。

```bash
cyeam ask "如何设计一个高并发消息队列？"
cyeam ask "这个系统的限流怎么做？" --mode think
cyeam ask "Review 这段代码" --mode expert
cyeam ask search "golang 性能优化"
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
cyeam ask <问题>              默认 fast 模式
cyeam ask <问题> --mode fast   快速回答（默认）
cyeam ask <问题> --mode think  深度思考
cyeam ask <问题> --mode expert 专家模式
cyeam ask search <关键词>      搜索 cyeam.com 站内内容
```

## 输出

`ask` 实时 streaming 输出到 stdout，适合管道或重定向。`ask search` 输出 JSON。

## 注意事项

- 需要联网，调用 `cyeam.com` 后端
- 不需要登录
- `search` 子命令返回站内搜索 JSON 结果，不是 AI 问答