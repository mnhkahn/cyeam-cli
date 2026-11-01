---
name: arithmetic
description: 北师大版小学数学口算题生成。通过 cyeam arithmetic 本地出题，支持一年级、二年级及 2026 新版四年级上册、答案、练习单和逐行平板作业；也支持 PaddleOCR 图片检查。
---

# 北师大版小学口算

出题逻辑已实现于 CLI 的 `internal/arithmetic`。必须调用 `cyeam arithmetic` 生成题目，不再由模型随机编题或计算答案。无需 Microsoft 登录或 AI 接口。

## 批改时的答案展示

- 批改、判分、查错和订正时，默认只提供题号、对错或待核对、评分、错误类型及不泄露答案的重做提示，不直接给出正确答案、完整解法或能反推出答案的差值。例如口算提示“第7题请重新计算，检查乘法和进位”，不告诉孩子应该填几。
- OCR 数字与计算值不符，只能说明“按识别值比较不符”；没有清晰的计算步骤证据时，不得断言孩子照抄、漏乘、丢进位或定位到某一位的具体错误。不得根据差值编造错因，也不能把不同照片或不同轮次 OCR 的差异当作孩子改过答案的证据。用户质疑识别时，应保留原始读数和输入来源，先排查识别链路，不能用标准答案改写 OCR。
- 必须内部验算；口算图片检查由 CLI 依据飞桨 OCR 结果完成计算和判分，直接采用 CLI 的结果，不让 AI 再读图识别或重复验算。答案缺失、无法解析或题型不支持时才标为无法判定，不猜测补齐。
- 只有用户明确要求提供答案，或明确允许某些题目/场景“先参考答案”时，才在授权范围内展示答案；单纯要求检查、打分、订正或说明错因不视为允许给答案。一次例外不延伸到其它题目或以后批改。
- 此规则适用于聊天回复、Trello 批改正文、图片/音频反馈、复习提醒及分享的报告/附件。内部工具返回的正确答案、计算过程和差值不得默认原样转发。

## 命令

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

- `--grade` 支持 `1`、`2`、`4`，四年级仅对应 2026 新版上册。用户未指定年级时，可根据明确题型选择 `--type`，CLI 自动推断年级；无法确定时询问用户，不能自行使用其它年级。
- `--type` 接受英文 ID 或完整中文名，多个题型用逗号分隔，也可重复传参；不能跨年级混合。不指定时混合该年级所有题型。先用 `types` 确认 ID，不猜参数。
- `--count` 默认 20，范围 1–1000；一批题目不重复，混合题型数量差不超过 1。有限题型（如表内乘法只有 81 个有序算式）可能达不到请求题量，CLI 会报错；应减少题量或增加题型，不重复填充。
- `--seed` 可选。省略时自动生成种子，JSON 中返回实际种子供复现。
- 默认 JSON 输出遵循 CLI 的 `ok/data` 信封，`data.questions` 包含 `type` 和 `text`，仅传 `--answers` 时包含 `answer`。
- `--format text --pretty` 输出无标题、无编号、每行一道题的原始文本。`--format worksheet --pretty` 输出可读练习单；不加 `--pretty` 时，文本仍包装在 JSON 信封内。
- 默认不附答案。用户明确要求答案或允许先参考答案时，在指定范围内使用 `--answers`；文本格式在题目之后的独立“答案”区列出编号答案，JSON 则在每题上返回答案字段。解析仅在用户明确要求时另行解释。

## 支持范围

- 一年级：100 以内加减、连加、连减、加减混合、人民币计算。
- 二年级：100 以内竖式加减及连加减、表内乘法与除法、乘加乘减、表内乘除混合、有余数除法、三位数加减及连加减、米与厘米换算。
- 四年级上册：整十整百整千乘法、积的变化规律、两位数乘法拆分、三位数乘两位数（含整十整百变形）、加法与乘法交换律结合律、乘法分配律、大数读写与填空、数量关系与相遇问题、明确近似方式的乘法估算。

数字范围、中间结果、整除与余数、凑整结构、单位和估算方式由代码控制。三年级、其它学期、分数、小数及旧版四年级多位数除法不在范围内；用户提出时说明范围，等待后续扩展，不改用模型绕过限制。

## 平板口算作业

用户要求把题目布置到 Trello 或作业平板时：

1. 用 `--format text --pretty` 出题，**不传 `--answers`**。保留每行一道题、完整单位及答题空格，不加标题、姓名栏、Markdown 表格、编号或多栏排版。
2. 交给 `homework` skill 排期，并遵循 `trello` skill 的命令格式。每天全部题目作为一张卡片，用 `cyeam trello card create --type arithmetic --task <多行题目>` 写入。
3. `--task` 使用真实换行，预计用时、执行顺序等说明放在 `--desc`；不生成 PDF 或附件。
4. 用户另要答案时，用相同 `--seed` 和参数加 `--answers` 单独获取，不能混进作业内容。

## PDF 与打印版

- 原始题目文本可直接传给 `cyeam pdf --mode arithmetic`；按 PDF skill 选择 `standard`、`grade4`、`large-number`、`quantity`、`vertical` 等布局。
- `worksheet` 文本包含多栏及元信息，不能作为 PDF arithmetic 模式的输入；该模式只接受逐行题目。需要把标题、姓名栏、纵向算式和答案页完整保留到 PDF 时，按 PDF skill 使用自定义 Typst。
- 题目与答案分离，不把 `--answers` 的文本输出直接传给 PDF arithmetic 模式。
- 只有实际生成文件后才说明已生成 PDF；用户要求打印时转交 print-file skill。

## 图片检查

CLI 图片检查仅使用百度飞桨 PaddleOCR 服务。使用者必须配置自己的接口地址和密钥；未配置时明确返回功能不可用，不上传图片、不调用其它 OCR 或下载本地模型。口算出题无需密钥，继续正常使用。

先让使用者提供自己注册服务后的接口地址和密钥，通过下列命令保存，不能内置开发者密钥：

```bash
cyeam ocr config set --url "https://使用者自己的OCR服务地址"
cyeam ocr config status
cyeam ocr config clear
```

也可先用 `cyeam ocr config set --url "接口地址" --url-only` 单独保存地址，之后运行 `cyeam ocr config set` 隐藏输入密钥。单独保存地址不会启用 OCR，也不会修改已有的地址/密钥配对；`clear` 同时移除这个预设地址。


`set` 隐藏输入密钥；脚本可用 `--token-stdin`。优先存系统钥匙串，不可用时存 `~/.cyeam/paddleocr.json`（0600）。状态和成功输出不含密钥；保存不验证服务，也不上传图片。之后图片检查自动使用已保存配置；环境变量优先，默认必须同时提供 URL/token，不能把已保存 token 配到其它地址。显式 `--ocr-env` 仍可用于临时配置。


用户提交口算照片要求检查时，调用 `cyeam arithmetic check <image>`。检查不受出题年级限制，目前支持横式四则运算、括号、精确小数/分数及商余数答案；单位换算、文字应用题、估算等不能自动判定的内容仍需人工核对。

```bash
# 使用进程环境中的 OCR_API_URL / OCR_API_TOKEN
cyeam arithmetic check homework.jpg --expected-count 18

# 显式读取已有的 PaddleOCR 接入配置（环境变量优先）
cyeam arithmetic check homework.jpg --ocr-env ~/code/wangxizhi/.env --expected-count 18 --timeout 10s --retries 2

# 保存 OCR 原始结果，供不重复上传的离线回放
cyeam arithmetic check homework.jpg --save-ocr ocr.json
cyeam arithmetic check homework.jpg --ocr-result ocr.json
```

- 默认使用普通文字识别任务 `promptLabel: ocr`，不使用整页 `spotting` 定位模式。上传前在内存中将最长边限制为 1600px（不放大较小图片；缩小时用高质量 JPEG），原图文件不改动；服务端 `maxPixels: 1003520`、`maxNewTokens: 2048`，不请求可视化图片。文本块按行解析，空文字响应直接报识别失败，不能当作“识别到 0 题”的正常结果。
- 图像会发送到配置的 PaddleOCR HTTP 服务，上传方式与 wangxizhi 的客户端相同（Base64、`fileType: 1`、`Authorization: token ...`）。CLI 直接调用，不依赖 wangxizhi 的 Python 环境；不把凭证写进代码或输出。
- 接受 PNG、JPEG、GIF、WebP，最大 20 MiB；当前按单张静态图检查。`--ocr-env` 只读指定文件的 `OCR_API_URL` 和 `OCR_API_TOKEN`，不执行文件内容；省略时优先使用完整进程环境配置，否则读取本机已保存配置，不自动寻找其它仓库的配置。
- 默认返回 JSON 信封，`data.questions` 列出题号、表达式、识别答案、计算结果、差值（计算结果减识别答案）、比较结果、状态、原因及 OCR 证据索引。商余数答案分别核对商和余数，不提供数值差值。这些是内部验算数据；默认向用户反馈时省略 `expected_answer`、`difference` 和完整解法，不分享带答案的原始报告。批改优先读取 JSON 后整理不含答案的反馈；`--pretty` 的原始输出含计算结果，只供内部核对，除非用户明确允许，不直接展示或转发。
- `status` 为 CLI 的判分结果：`correct` / `incorrect` 直接作为对错反馈，`needs_review` 表示答案缺失、无法解析、多个候选答案、题号重复或题型不支持等无法判定的情况。直接采用飞桨识别内容及 CLI 本地验算，不让 AI 再次识别原图、修正 OCR 数字或重复计算。
- 默认不按置信度拦截判分；飞桨没有返回置信度时，仍直接验算和判对错。仅用户明确要求时传 `--min-confidence`（0–1），只过滤服务实际返回且低于阈值的分数；缺失分数不阻断判分。
- 通过坐标关联横式题干与同一行右侧答案，**不会把下方竖式草稿当作最终答案**，也不会凭计算结果猜测或修正 OCR 数字。支持 PaddleOCR 的普通文本及常见 LaTeX 算式格式。
- `--expected-count` 用于检查题数是否一致；编号题单还会报告识别到的编号范围内的缺号。总数一致不能证明 OCR 没有漏题或误识别；零题结果不能视为全部正确。
- `--timeout` 为每次请求超时，默认 10s，最大 5m；超时后立即顺序重试，`--retries` 默认 2（共最多 3 次请求，约 30s 加本地处理），范围 0–5，传 0 禁用重试。只有超时自动重试，认证/HTTP 错误不重试，用户取消立即结束。每次重试会在 stderr 提示，不重复做本地图像处理；客户端取消不能保证服务端停止上一轮推理。`--save-ocr` 保存服务原始响应（可能含图片链接），文件应留在本地；测试夹具只能保留去除凭证、接口地址和图片链接后的文字/坐标，不能把原图当作识别正确的证据。`--ocr-result` 回放不上传图片，图片参数用于核对缓存的输入指纹；没有指纹的旧缓存会提示来源无法核验。
- `--pretty` 输出中文判分清单。面向用户按 `correct`、`incorrect`、`needs_review` 汇报对/错/无法判定，不因为服务未返回置信度而再次要求 AI 看图复核。默认不展示标准答案和差值。

### 图片方向、原图与缓存来源

- OCR 请求默认开启飞桨文档方向矫正。横转照片可显式传 `--rotate 90|180|270`（顺时针），仅在内存中旋转，不改动附件或原文件；该参数不用于 OCR 缓存回放。
- Trello 口算照片送识别前优先通过 `card attachment get ... --max-width 0` 下载同一附件原图；不要把 1200px 缩略预览当成原图。此要求仅针对 OCR 输入，附件的现有展示方式保持不变。
- 每份识别结果的 `input` 记录原图名、原图及实际上传数据的 SHA-256、上传尺寸、识别模式、手动旋转角度及来源是否已核验。`--save-ocr` 同时在原始响应的 `_cyeam_input` 保存输入指纹。必须确认当前文件与本次提交对应，不把旧照片、旧提交的结果用于订正照片。
- `--ocr-result` 会核对当前图片与缓存的 SHA-256，不一致时报错。老缓存没有指纹，或原图文件不在本机时，只能作为未核验来源的回放，不能宣称是当前照片的新识别结果；不要因方便而替换为另一个样例缓存。
