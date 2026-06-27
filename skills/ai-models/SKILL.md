---
name: ai-models
version: 0.1.16
description: AI免费模型查询——从 cyeam.com 获取各平台免费模型排行榜，支持按平台/类型/名称筛选。【重要】必须先读 skill 原文获取正确命令格式，禁止瞎猜。
---

# AI免费模型查询

## 规则

用户问免费模型 → 跑 `cyeam ai models` → 返回结果。可按 `--platform`、`--type`、`--search` 筛选。

```
用户: "有哪些免费的模型？"                  →  cyeam ai models
用户: "OpenRouter 上有哪些免费模型？"       →  cyeam ai models --platform openrouter
用户: "免费的推理模型有哪些？"              →  cyeam ai models --type reasoning
用户: "搜索 Qwen 相关的免费模型"            →  cyeam ai models --search qwen
```

## 筛选参数

| 参数 | 说明 | 值 |
|------|------|----|
| `--platform` | 按平台筛选 | openrouter, siliconflow, nvidia, huggingface |
| `--type` | 按类型筛选 | text, multimodal, reasoning |
| `--search` | 按名称/提供商搜索 | 任意关键词 |

## 数据

- 数据源: https://www.cyeam.com/ai/models
- 每日更新，聚合 5 个平台 (OpenRouter, SiliconFlow, NVIDIA, 百度飞桨, Hugging Face) 的免费模型
- 包含文本/代码 ELO 排名、上下文窗口、特性标签等

## 输出

默认 JSON 信封，data 中包含 date、count、models 数组。`--pretty` 去掉信封。直接展示即可。不总结、不分析。