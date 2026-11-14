---
name: asr
metadata:
  version: "0.2.0"
description: 语音转文字与英语阅读批改——把本地音频识别成文本，并通过 cyeam.com 服务端完成阅读转写、对齐、评分、重点词和反馈。用户要识别音频、转写录音或批改 Trello 朗读作业时使用。【重要】必须先读 skill 原文获取正确命令格式，禁止瞎猜。
---

# 语音转文字（ASR）

通过 `cyeam asr` 调用 cyeam.com 语音识别接口，把本地音频转成文字。无需登录、无需配置。

## 单纯转写

```bash
cyeam asr <audio-file>
cyeam --pretty asr <audio-file>
```

默认输出 JSON 信封，`data` 含 `file` 和 `text`；`--pretty` 只输出识别文本。此命令调用 `/api/speech-to-text`，适合没有阅读原文的通用转写。

## 英语阅读批改

```bash
# 已有转写：仍调用服务端重新对齐和评分，不是离线计算
cyeam asr review --reference passage.txt --transcript transcript.txt > reading-report.json

# 直接上传录音：服务端识别后生成完整报告
cyeam asr review --reference passage.txt --audio reading.m4a > reading-report.json

# 输出服务端固定反馈，并另存文本
cyeam asr review --reference passage.txt --transcript transcript.txt \
  --feedback-out feedback.txt --pretty
```

- `--audio` 调用 `POST /api/homework/reading`；`--transcript` 调用 `POST /api/homework/reading/regrade`，不重复调用 ASR。
- 转写、规范化、对齐、功能词过滤、替换统计、重点词、等级和反馈全部由服务端完成。CLI 和模型都不得重新计算、筛词、覆盖等级或改写文案。
- 默认 JSON 的 `data.input` 含 `kind=reading`、`reference` 和 `transcript`；`data.report` 含原始文本、SHA-256、规范化记录、匹配/替换统计、完整差异、重点词及固定反馈。服务端以后新增字段也应原样保留。
- 页面和服务端当前直接给出等级，不默认标“暂评”，也不自动给 100。阅读参考分为匹配实词数÷（匹配＋替换实词数）×100，≥80 优、超过60且低于80 良、40至60 中、低于40 差。
- 同一原文词的不同转写合并累计，替换达到3次时进入 `focus` 和练习链接；按次数降序、原文首次出现顺序排列，保留词尾并过滤功能词。
- 少于3个可比较词（短文按实际词数）或可比较量不足原文有效词的四分之一时不自动评级。阅读参考分是文本对比分数，不是声学发音评分或孩子真实正确率。
- 不再支持 `--confirmed-grade`。需要人工改等级时，不能通过编辑报告字段实现；服务端重评会忽略这些字段。

用户要求回写时，执行：

```bash
cyeam trello card review <card-id> --reading-report reading-report.json
```

不要另外拼接 `--text`、`--grade` 或 `--provisional`。回写端提取保存的 `input`，再次调用服务端 `/api/homework/reading/regrade`，忽略中间文件里被改写的反馈、得分和等级；重试仍使用同一 `--review-id`。

## 约束

- 阅读录音支持 m4a、mp3、wav、webm、ogg、aac、flac、amr、3gp、opus；单个文件不能超过 15MB。
- 原文最多 2000 词、30000 字节，转写最多 60000 字节。
- 转写和重评都需要联网；接口不可用时命令返回错误，不要自行拼装 HTTP 请求或恢复本地评分。
- 识别可能需要排队，CLI 默认等待服务端响应；失败不生成评级，不自动切换其它收费接口。

## 识别质量说明

- 对儿童语音、有背景人声或噪声的录音，识别文本可能混入杂音内容或错字。明显是背景人声的片段不要算成孩子的跟读错误。
- 只有一段录音、没有原文可对照时，不能据此判定内容对错，只能确认是否完成朗读。
- 保留原始转写，再使用服务端返回的规范化和对齐结果。不能直接把逐词 diff 或 WER 当作孩子的朗读错误，也不能把 `1-WER` 称为孩子准确率。
- 大小写、标点、上下文缩写及支持的同音转写等价由服务端处理；未覆盖或歧义形式保留原始材料供核听，不能临时添加特例去提高等级。
- 缺失和多出保留为差异证据，不计入阅读参考分；替换是复读线索，不直接认定为发音错误。内容已经确定漏读、错读前，应核听对应录音。

## 作业场景

提取 Trello 作业时使用 `cyeam trello homework --board <board-id> --asr`，可直接获得音频附件的 `transcript`，失败原因在 `transcript_error`。已有识别文本时保存为 UTF-8 文件并调用 `asr review --transcript`；没有识别文本或识别失败时，使用 `--audio` 重新完成阅读批改。详细流程见 homework skill。
