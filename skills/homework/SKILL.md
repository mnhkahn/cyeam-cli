---
name: homework
version: 0.3.0
description: 家庭作业批量排期——支持普通作业、背单词和阅读英语类型；用户提供作业内容和完成时间周期，为周期内每一天的每项作业在 Trello 学习任务看板上各创建一张任务卡片。布置作业、安排每日作业计划时使用。依赖 trello skill。【重要】必须先读 skill 原文获取正确命令格式，禁止瞎猜。
---

# 家庭作业批量排期

用户提供**作业内容清单**和**完成周期**（起止日期），为周期内**每一天 × 每项作业**各创建一张 Trello 卡片。

## 作业类型

- 每项作业可带 `type`。省略时为**普通作业**，保持原有标题、截止时间和描述规则不变。
- 支持的类型：`普通作业`、`背单词`、`阅读英语`。
- 一项作业只对应一个具体任务和一张 Trello 卡片；不要为了单词或阅读链接额外创建卡片。
- `背单词`的具体任务必须给出待背单词。接受英文逗号、中文逗号或空白分隔；去除首尾空白后按原顺序以英文逗号连接。
- `背单词`卡片的描述必须记录单词和翻译链接。链接格式固定为 `https://www.cyeam.com/ai/translate?words=<英文逗号分隔的单词>`，例如单词 `bedroom, armchair, cushions` 对应 `https://www.cyeam.com/ai/translate?words=bedroom,armchair,cushions`。不要将逗号替换为 `%2C`，以保持链接可读。
- `阅读英语`的具体任务必须由用户提供阅读链接，例如 `/ai/translate?textbook=0&article=4`。将该链接原样写进卡片描述；不要改写、补全、解析或生成链接。

## 重要：这是流程 Skill，不是 CLI 命令

- **禁止执行 `cyeam homework`**：这个命令不存在。不要将 Skill 名 `homework` 当成 CLI 子命令或可执行文件。
- 本 Skill 只负责解析、排期、去重和确认的工作流；所有实际查询和创建都必须加载并遵循 `trello` Skill，且命令必须以 **`cyeam trello`** 开头。
- `cyeam trello homework --board <board-id>` 是查看当天已有作业和下载附件的只读报告命令，**不能**用于新增作业。新增作业只能使用 `cyeam trello card create`。

## 前置

- Trello 操作全部走 trello skill（`.agents/skills/trello/SKILL.md`），命令格式以它为准，禁止瞎猜。
- 未登录先 `cyeam trello login`，用 `cyeam trello status` 确认凭据有效。
- 不知道看板/列表 ID 时：先 `cyeam trello boards` 找到学习任务看板，再 `cyeam trello lists <board-id>` 找到「待执行」列表。不要按名称猜 ID。

## 流程

1. 解析用户输入：作业项目列表（如 口算、练字、钢琴、背单词或阅读英语）、每项的可选类型和具体任务、起止日期（含首尾）、可选的起始天数序号。未给类型按普通作业处理。
2. 展开成创建计划，周期内每一天、每项作业一张卡片：
   - 标题：`<作业名>（第N天）`，N 逐天递增。用户给了起始序号就用它；没给则先 `cyeam trello cards --board <board-id>` 看已有卡片的编号规律，从最大序号 +1 继续；完全没有规律可遵循时用 `<作业名>（YYYY-MM-DD）`。
   - `--due`：当天 20:00 本机时区的 RFC3339（如 `2026-09-08T20:00:00+08:00`），用户另有指定从其指定。
   - 普通作业有具体要求（如"口算 100 题"）时写进 `--desc`。
   - 背单词作业把单词和翻译链接写进 `--desc`，例如：

     ```text
     背单词：bedroom, armchair, cushions
     翻译链接：https://www.cyeam.com/ai/translate?words=bedroom,armchair,cushions
     ```
   - 阅读英语作业把用户提供的链接原样写进 `--desc`，例如：

     ```text
     阅读英语链接：/ai/translate?textbook=0&article=4
     ```
3. 去重：`cyeam trello cards --list <list-id>` 检查，同名且同 due 的卡片已存在则跳过。
4. 把完整计划清单（日期 × 作业 × 标题）列给用户确认一次，确认后逐条执行：

   ```bash
   cyeam trello card create --list <list-id> --name "<作业名>（第N天）" --due <RFC3339>
   ```

5. 汇报结果：创建成功 / 跳过（已存在）/ 失败各多少张，失败的附原因。

## 规则

- 每天每项作业一张卡片，不要把多天或多项作业合并成一张。
- 背单词和阅读英语链接写入同一张作业卡的描述；不要作为 Trello 附件或评论，也不要另建卡片。
- 批量创建是对外部状态的批量变更，必须先列出计划清单获得用户一次确认再执行。
- 周期跨月/跨年时逐日展开，不要漏掉首尾日期。
- 日期、due 一律用本机时区。
