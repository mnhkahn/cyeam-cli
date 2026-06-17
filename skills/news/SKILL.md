---
name: news
version: 0.1.13
description: AI 新闻与科技资讯——从 cyeam.com 获取最新 AI 资讯和技术动向，支持按日期范围筛选。使用 cyeam news 命令。
---

# AI 新闻与科技资讯

## 概述

查询 cyeam.com 聚合的最新 AI 资讯和技术动向新闻。数据来源包括 TechCrunch、Wired、The Verge 等英文科技媒体，已翻译整理为中文。

## 使用方式

```bash
# 最近一周的新闻（默认）
cyeam news list

# 指定日期范围
cyeam news list --from 2026-06-10 --to 2026-06-16

# 获取某天完整内容
cyeam news get --date 2026-06-16
```

## 输出说明

`news list` 输出分为两部分：
- **技术动向** — 综合技术新闻（编程语言、开源项目、架构设计等）
- **AI 资讯** — AI 专题新闻（大模型、Agent、融资、安全等），来自 TechCrunch 等英文源

## 行为要点

- `--from`/`--to` 不指定时默认为最近 7 天到当天
- `news get` 需要 `--date` 参数，输出某天的完整新闻详情
- 所有输出为 JSON 信封格式 `{"ok":true,"data":"...","_notice":{...}}`
- 新闻内容已由上游翻译为中文，直接展示即可，无需额外翻译

## 适合的场景

- "今天有什么 AI 新闻？"
- "最近一周的科技资讯"
- "6月10号到16号的 AI 融资消息"
- "帮我总结一下最近的 AI 安全新闻"