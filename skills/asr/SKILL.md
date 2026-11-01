---
name: asr
version: 0.1.0
description: 语音转文字——把本地音频文件（朗读录音、背诵录音等）识别成文本。用户要识别音频、转写录音，或批改 Trello 朗读/背诵类作业录音时使用。【重要】必须先读 skill 原文获取正确命令格式，禁止瞎猜。
---

# 语音转文字（ASR）

通过 `cyeam asr` 调用 cyeam.com 语音识别接口（服务端代理到 ASR 模型），把本地音频转成文字。无需登录、无需配置。

## 用法

```bash
cyeam asr <audio-file>
cyeam --pretty asr <audio-file>
```

默认输出 JSON 信封，`data` 含 `file` 和 `text`；`--pretty` 只输出识别文本。

## 约束

- 支持常见音频格式：m4a、mp3、wav、webm、ogg、aac、flac、amr、3gp、opus，按扩展名设置上传 Content-Type。
- 单个文件不能超过 15MB（服务端限制），超出会直接报错。
- 识别需要联网；接口不可用时命令返回错误，不要自行拼装 HTTP 请求代替。

## 识别质量说明

- 对儿童语音、有背景人声或噪声的录音，识别文本可能混入杂音内容或错字。判读作业时应以识别文本为参考证据，结合原题要求判断；明显是背景人声的片段（如家长说话）不要算成孩子的跟读错误。
- 只有一段录音、没有原文可对照时（如听力题），不能据此判定内容对错，只能确认是否完成朗读。

## 作业场景

Trello 作业录音（`homework-audio-*.m4a`）在 `cyeam trello homework` 报告里会自动带 `transcript` 字段（仅英语卡片），一般直接读取即可；报告里没有识别文本或识别失败时，用本命令对附件的 `saved_to` 路径手动转写。详细流程见 homework skill。
