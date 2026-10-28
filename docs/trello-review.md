# 提交作业批改

`cyeam trello card review` 实现「批改结果提交协议 v1」，沿用 `cyeam trello login` 或 `TRELLO_API_KEY` / `TRELLO_TOKEN` 授权。命令发布人工准备好的结果，不执行识别、批改或评分。

```bash
# 文字：换行和引号由 JSON 序列化器处理
cyeam trello card review <原卡片ID> --text '第3题需要订正：注意进位。
其余题目正确。'

# 较长正文读取 UTF-8 文件
cyeam trello card review <原卡片ID> --text-file review.txt

# 一张图片或一段音频，二选一
cyeam trello card review <原卡片ID> --image correction.webp
cyeam trello card review <原卡片ID> --audio correction.m4a
```

`--text`、`--text-file`、`--image`、`--audio` 必须且只能选一个。正文不能全为空白；媒体必须为非空本地文件，上限 50 MiB。图片支持 jpg/jpeg/png/webp/gif；音频支持 m4a/mp3/wav/aac/ogg/opus/amr/3gp。扩展名需符合真实文件格式，命令据此发送明确 MIME；设备实际音频解码能力仍需实机验证。

命令始终使用原卡片，不修改原题描述。省略 `--has-errors` 时只上传附件和追加评论；明确传入时同时更新作业状态。媒体附件名为 `homework-review-image-<UUID>.<ext>` 或 `homework-review-audio-<UUID>.<ext>`，上传时提供 `file`、`name` 及对应 MIME。确认响应为原生上传后，再发布评论。评论格式为全角冒号前缀 `作业批改：`、换行和一个 JSON 对象：

```json
{"version":1,"type":"text","text":"第3题需要订正：注意进位。\n其余题目正确。"}
```

图片和语音将 `text` 替换为 `attachment_id`，`type` 分别为 `image`、`audio`。评论不包含 URL、Base64 或 Markdown。只有上传、尚未发布评论的附件不构成批改结果。

## 重试和历史

首次执行可传 `--review-id <UUID>`；默认自动生成。stderr 在请求前输出 UUID 和本地状态路径，便于失败后恢复；成功 JSON 返回 `review_id`、`card_id`、`action_id`、`state_file`，媒体还返回 `attachment_id`。

```bash
# UUID 仅为示例；每轮批改生成自己的新 UUID
cyeam trello card review <原卡片ID> --image correction.webp \
  --review-id 3b754652-a17b-4924-96e7-e4f878d0cf8f

# 网络失败后，用原命令、原 UUID、原文件内容重试
```

`~/.cyeam/trello-reviews/<UUID>.json` 以 0600 权限保存正文、内容摘要、上传及发布阶段、附件 ID、action ID，不保存凭据。写请求前先保存意图，更新用临时文件同步并原子替换；同一状态文件用独占锁防止并发发布。同一 UUID 不可改卡片、类型、内容、媒体扩展名或 `--has-errors`（包括是否省略）。

失败后重试会查询卡片附件和分页评论，复用已成功的原生附件或 action ID；评论恢复忽略提交前已有的同文评论。批改评论已发布但状态更新失败时，使用原 UUID 和相同参数重试，只补状态更新，不重复评论。整个流程成功的本地记录直接返回，不重新移动后续已流转的卡片。查询失败会停止。请求结果未知且未找到匹配记录时，也会停止，避免 Trello 无原生幂等键时重复发布。稍后用同一 UUID 再次查询；人工确认写入失败后，可显式添加 `--retry-uncertain` 允许再次 POST，此操作有重复风险。

保留 JSON 状态直到不再需要重试。进程异常退出可能留下 `<UUID>.json.lock`，确认没有进程正在使用该提交后可移除锁，再用原 UUID 重试。状态及锁在当前设备生效；不要跨设备并行提交同一轮批改。新一轮使用新 UUID，即使文字相同也追加新的历史；不要删除历史评论引用的附件。修改旧评论的编辑功能不在本命令范围内。

## 验证

自动测试通过本地模拟 Trello 验证三种协议、MIME、上传先于评论、丢失响应后的恢复、相同文字新历史、未知结果停止、分页与非法输入；测试不写真实卡片。

实机验收需在指定测试卡分别提交文字、图片、音频，检查最新在前、历史保留、原照片/录音及普通评论不成为批改。平板端还需验证无批改、附件被删除、网络中断、切换作业、离开页面及切后台停止播放、刷新恢复。CLI 实现不覆盖平板 UI 或播放生命周期。

## 四个作业状态

Trello 列表是状态来源：待完成 → 已完成 → 待改错 → 已改错。首次提交进入「已完成」，孩子提交订正进入「已改错」；已改错表示已交订正，是否正确由最新批改说明。

```bash
# 批改有错：发布带 has_errors:true 的评论，并移动到「待改错」、dueComplete=false
cyeam trello card review <原卡片ID> --text '第3题需要订正。' --has-errors=true

# 批改通过：发布 has_errors:false；初次提交保留「已完成」，复批保留「已改错」
cyeam trello card review <原卡片ID> --text '订正全部正确。' --has-errors=false
```

图片、语音和文字文件同样支持该参数。未完整判定时省略参数，不根据评语中的关键词推断状态。目标列表按当前卡片的看板查询实际 ID，缺少时自动创建。报告 `trello homework` 的 `status` 为 `todo`、`completed`、`needs_correction`、`corrected`，其他列表为 `unknown`；`list_name` 保留实际列表名，不能只用 `due_complete` 区分状态。

状态更新失败会返回错误，不能将评论发布成功当作整轮操作完成。重试须保持原 UUID、正文和 `--has-errors`。同一 UUID 完成后再次运行不重放状态更新，新一轮批改使用新 UUID。


## 评分字段与平板

发布：`cyeam trello card review <card-id> --text-file review.txt --grade 良 --has-errors=true`。`--grade` 接受 100、优、良、中、差；`--provisional` 需同时传等级。100 表示整份全对，自动按 `--has-errors=false` 批改通过，不可与 `--provisional` 或 `--has-errors=true` 并用。评分保存于原卡片描述第一行「打分结果：良」（或「打分结果：暂评·良」），替换首行已有同名字段，否则新增一行，保留全部原题和其他说明。兼容免费版，无需原生自定义字段，不创建额外卡片。不要同时编辑同一张卡片描述。

协议 v1 评论可选增加 `grade` 和 `provisional`，旧客户端仍能展示正文。没有评分时保持旧行为。平板读取描述第一行和每轮评论的评分快照，以文字、图标、不同色彩显示，本地缓存保留最近评分。订正提交保留最近评分，复批后再更新。

评分写入在评论发布后、列表状态更新前执行，单独保存成功检查点。描述更新失败用相同 UUID 和完整参数重试，只补评分；状态更新失败重试跳过已成功的评分。完成后重复调用不重放旧评分。同一 UUID 改等级或暂评标志会拒绝。新一轮复批必须使用新 UUID。
