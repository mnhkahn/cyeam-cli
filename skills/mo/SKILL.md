---
name: mo
version: 0.1.16
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
cyeam mo guwen <text>                 行书古文生成（JSON）
cyeam mo guwen <text> --ai-compose    AI 补全缺失的字形
cyeam mo char detail <char>           单字行书写法候选（JSON）
cyeam mo char composition <char>      偏旁部首（JSON）
cyeam mo char compose <char> --out png  合成书法图片（需要 --out）
cyeam mo ocr <image>                  书法图片 OCR（JSON）
```

## 输出

- `guwen`、`char detail`、`char composition`、`ocr` 输出 JSON 到 stdout
- `char compose` 输出 PNG 文件（需 `--out` 指定路径）

## 注意事项

- 仅支持行书，不支持楷书、隶书等其他字体
- 不需要登录
- `--ai-compose` 会对库中没有的字用 AI 生成行书写法
- `ocr` 上传图片到 cyeam.com 识别，图片格式不限但建议 PNG/JPG