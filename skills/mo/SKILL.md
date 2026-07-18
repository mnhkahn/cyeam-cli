---
name: mo
version: 0.1.17
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
cyeam mo guwen <text> --ai-compose    仅为缺失字形启用合成补全
cyeam mo char detail <char>           查询原帖单字候选（JSON，默认首选）
cyeam mo char composition <char>      偏旁部首（JSON）
cyeam mo char compose <char> --out png  强制拆字合成书法图片（需要 --out）
cyeam mo ocr <image>                  书法图片 OCR（JSON）
```

## 输出

- `guwen`、`char detail`、`char composition`、`ocr` 输出 JSON 到 stdout
- `char detail` 返回的每个候选包含原帖字形与出处；用户要单字时优先使用它
- `char compose` 输出 PNG 文件（需 `--out` 指定路径），且始终是拆字合成，不代表原帖原字

## 注意事项

- 仅支持行书，不支持楷书、隶书等其他字体
- 不需要登录
- `--ai-compose` 只会对库中没有的字启用合成补全；已有原字仍应保留原字
- `ocr` 上传图片到 cyeam.com 识别，图片格式不限但建议 PNG/JPG
