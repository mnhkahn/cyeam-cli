---
name: news
version: 0.2.2
description: AI 新闻与科技资讯——获取最新科技新闻，支持二次爬取详情、AI 自动总结。【重要】必须先读 skill 原文获取正确命令格式，禁止瞎猜。
---

# AI 新闻与科技资讯

## 使用方式

```bash
# 获取某天新闻（默认 JSON 格式）
cyeam news get --date 2026-06-28

# 人类可读表格格式
cyeam news get --date 2026-06-28 --pretty
```

## 数据来源

CLI 内置两个独立数据源：

| 模块 | 说明 |
|------|------|
| 技术动向 (`news`) | 综合技术新闻 |
| AI 资讯 (`ai_news`) | AI 专题新闻 |

## CLI 输出

CLI 返回结构化数据，每条新闻包含：

| 字段 | 说明 |
|------|------|
| `title` | 新闻标题 |
| `link` | 原文链接 |
| `image` | 首图 URL |
| `create_time` | 发布时间戳 |

## 大模型后续处理指南

拿到 CLI 返回的数据后，可按以下流程二次加工：

1. **二次爬取详情**：对重要链接执行深度爬取，获取完整正文
2. **AI 总结提炼**：每条新闻提炼 100-200 字中文要点
3. **打分排序**：按新闻重要性 0-100 分打分，降序排列
4. **去重筛选**：过滤重复、低价值内容，保留 8-15 条核心新闻

## 输出格式

### CLI 返回结构

`cyeam news get` 返回标准 JSON 信封：

```json
{
  "ok": true,
  "data": "{...}",    // data 是字符串化的 JSON 对象，见下文结构
  "_notice": {}
}
```

### data 字段的 JSON 结构

```json
{
  "date": "2026-06-27",
  "news": {
    "create_time": 1719331200,
    "summary": "今日科技要闻总结...",
    "news": [
      {
        "title": "新闻标题",
        "link": "https://...",
        "description": "AI提炼的完整要点",
        "image": "https://.../img.jpg",
        "create_time": 1719331200
      }
    ]
  },
  "ai_news": {
    "create_time": 1719331200,
    "news": [
      {
        "title": "AI新闻标题",
        "link": "https://...",
        "description": "要点内容",
        "image": "",
        "create_time": 1719331200
      }
    ]
  }
}
```

### 字段说明

| 字段 | 说明 |
|------|------|
| `date` | 新闻日期 YYYY-MM-DD |
| `news.summary` | 全局总结，一段话概括最重要的3-5个趋势 |
| `news[].title` | 新闻标题 |
| `news[].link` | 原文链接 |
| `news[].description` | AI 提炼的完整要点内容 |
| `news[].image` | 首图 URL（如有，空字符串代表无） |
| `news[].create_time` | 新闻发布时间戳（Unix 秒） |

### 输出格式参数

使用全局 `--pretty` 参数控制输出格式：

```bash
cyeam news get --date 2026-06-27           # 默认，结构化 JSON（在 envelope 中）
cyeam news get --date 2026-06-27 --pretty  # 人类可读的表格形式
```

## 适合的场景

- "今天有什么科技新闻？"
- "最近 AI 圈有什么动态？"
- "帮我总结下最新的 tech news"
- "最近有什么值得关注的开源项目发布？"

## 注意事项

- 微信公众号链接不二次爬取，直接展示原始摘要
- 二次爬取失败的链接也用原始摘要兜底
- 最终输出控制在 8-15 条新闻，过多时只保留最重要的
