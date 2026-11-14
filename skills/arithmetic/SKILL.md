---
name: arithmetic
description: 北师大版小学数学口算题生成与图片批改。通过 cyeam arithmetic 调用服务端接口出题、识别、验算和评级，支持答案、练习单、逐行平板作业及识别文字重评。
---

# 北师大版小学口算

出题和批改逻辑统一位于 cyeam.com 服务端。必须调用 `cyeam arithmetic`，不再由模型随机编题、识别图片、计算答案或重新评分。无需 Microsoft 登录、AI 接口或 OCR 密钥，但需要能够访问 cyeam.com。

## 批改时的答案展示

- 批改、判分、查错和订正时，默认只提供题号、对错或待核对、评分、错误类型及不泄露答案的重做提示，不直接给出正确答案、完整解法或能反推出答案的差值。
- 识别数字与计算值不符，只能说明“按识别值比较不符”；没有清晰的计算步骤证据时，不得断言孩子照抄、漏乘、丢进位或定位到某一位的具体错误。不得根据差值编造错因，也不能把不同照片或不同轮次识别的差异当作孩子改过答案的证据。
- 直接采用服务端的识别、验算和评级，不让 AI 再读图识别、修正识别数字或重复验算。答案缺失、无法解析或题型不支持时保留待核对，不猜测补齐。
- 只有用户明确要求提供答案，或明确允许某些题目/场景“先参考答案”时，才在授权范围内展示答案；单纯要求检查、打分、订正或说明错因不视为允许给答案。一次例外不延伸到其它题目或以后批改。
- 此规则适用于聊天回复、Trello 批改正文、图片/音频反馈、复习提醒及分享的报告/附件。

## 出题命令

```bash
# 查询当前支持的题型 ID、中文名和年级
cyeam arithmetic types
cyeam arithmetic types --grade 4

# 默认 20 题、不含答案；指定年级时均衡混合该年级全部题型
cyeam arithmetic --grade 2

# 指定一个或多个题型；也可传 types 返回的完整中文名
cyeam arithmetic --type multiply,divide --count 20
cyeam arithmetic --type 表内乘法 --count 20 --answers

# 固定随机种子，复现同一份练习
cyeam arithmetic --grade 4 --seed 42 --answers

# 练习单文本：标题、姓名日期栏、编号、多栏排版，竖式题纵向排列
cyeam arithmetic --grade 2 --format worksheet --pretty

# 每行一道题：用于平板作业或 PDF 输入
cyeam arithmetic --grade 2 --format text --pretty

# 直接生成 PDF：题目文本交给现有 PDF 命令
cyeam arithmetic --type multiply-laws --format text --pretty | cyeam pdf --mode arithmetic --layout grade4 -o practice.pdf
```

- `--grade` 的可用范围以 `cyeam arithmetic types` 实时返回为准。用户未指定年级时，可根据明确题型选择 `--type`，CLI 由服务端自动推断年级；无法确定时询问用户，不能自行使用其它年级。
- `--type` 接受英文 ID 或完整中文名，多个题型用逗号分隔，也可重复传参；不能跨年级混合。不指定时混合该年级所有题型。先用 `types` 确认 ID，不猜参数。
- `--count` 默认 20，范围 1–1000；一批题目不重复，混合题型数量差不超过 1。有限题型可能达不到请求题量，CLI 会报错；应减少题量或增加题型，不重复填充。
- `--seed` 可选。省略时自动生成种子，JSON 中返回实际种子供复现。
- 默认 JSON 输出遵循 CLI 的 `ok/data` 信封，`data.questions` 包含 `type` 和 `text`，仅传 `--answers` 时包含 `answer`。
- `--format text --pretty` 输出无标题、无编号、每行一道题的原始文本。`--format worksheet --pretty` 输出可读练习单；不加 `--pretty` 时，文本仍包装在 JSON 信封内。
- 默认不附答案。用户明确要求答案或允许先参考答案时，在指定范围内使用 `--answers`；文本格式在题目之后的独立“答案”区列出编号答案，JSON 则在每题上返回答案字段。解析仅在用户明确要求时另行解释。

## 支持范围

- 一年级：100 以内加减、连加、连减、加减混合、人民币计算。
- 二年级：100 以内竖式加减及连加减、表内乘法与除法、乘加乘减、表内乘除混合、有余数除法、三位数加减及连加减、米与厘米换算。
- 四年级上册：整十整百整千乘法、积的变化规律、两位数乘法拆分、三位数乘两位数（含整十整百变形）、加法与乘法交换律结合律、乘法分配律、大数读写与填空、数量关系与相遇问题、明确近似方式的乘法估算。

数字范围、中间结果、整除与余数、凑整结构、单位和估算方式由服务端代码控制。未列出的范围以 `types` 实时返回为准，不改用模型绕过限制。

## 平板口算作业

1. 用 `--format text --pretty` 出题，**不传 `--answers`**。保留每行一道题、完整单位及答题空格，不加标题、姓名栏、Markdown 表格、编号或多栏排版。
2. 交给 `homework` skill 排期，并遵循 `trello` skill 的命令格式。每天全部题目作为一张卡片，用 `cyeam trello card create --type arithmetic --task <多行题目>` 写入。
3. `--task` 使用真实换行，预计用时、执行顺序等说明放在 `--desc`；不生成 PDF 或附件。
4. 用户另要答案时，用相同 `--seed` 和参数加 `--answers` 单独获取，不能混进作业内容。

## PDF 与打印版

- 原始题目文本可直接传给 `cyeam pdf --mode arithmetic`；按 PDF skill 选择合适布局。
- `worksheet` 文本包含多栏及元信息，不能作为 PDF arithmetic 模式的输入；该模式只接受逐行题目。
- 题目与答案分离，不把 `--answers` 的文本输出直接传给 PDF arithmetic 模式。
- 只有实际生成文件后才说明已生成 PDF；用户要求打印时转交 print-file skill。

## 图片检查与文字重评

```bash
# 题数必填，范围 1–100
cyeam arithmetic check homework.jpg --expected-count 18
cyeam arithmetic check homework.jpg --expected-count 18 --pretty > arithmetic-review.txt

# 保存完整 JSON，人工核对并修改 data.input.items 后由服务端重新评分
cyeam arithmetic check homework.jpg --expected-count 18 > arithmetic-report.json
cyeam arithmetic regrade arithmetic-report.json > arithmetic-regraded.json
```

- 图片上传至 `POST /api/homework/review`，重评调用 `POST /api/homework/regrade`。识别、验算、等级和提示全部来自服务端，CLI 不在本地计算。
- 接受 JPG、PNG、WebP，单张最大 10 MiB、4000 万像素。当前按一张静态图检查，不支持 GIF。
- 服务端支持横式四则运算、括号、精确小数和分数；单位换算、余数、估算、文字应用题等不能自动判定的内容保留 `pending`。
- 默认 JSON 的 `data.input` 是可用于重评的规范化输入，`data.report` 含 `grade`、`provisional`、`correct`、`incorrect`、`pending`、`total`、`accuracy` 和逐题 `items`。
- 逐题 `status` 为 `correct`、`incorrect` 或 `pending`。直接采用这些状态，不改用旧版的 `needs_review`，也不要求 AI 再看原图。
- 服务端不返回标准答案、完整解法或差值。`--pretty` 只展示成绩、识别答案、判定和服务端提示。
- 重评接受完整 CLI 报告、完整服务端响应或单独的 `input` 对象；只发送其中的 `input`，忽略报告里可能被编辑的成绩和反馈。
- 不再使用 `--ocr-env`、`--save-ocr`、`--ocr-result`、`--min-confidence`、`--rotate`、`--retries`，也不需要 `cyeam ocr config`。
- Trello 图片送识别前优先通过 `card attachment get ... --max-width 0` 下载同一附件原图；不要把缩略预览当成原图。必须确认当前文件与本次提交对应，订正照片应重新调用 `check`，不能套用旧报告。
