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

命令只向原卡片上传附件和追加评论，不创建卡片，不移动列表，不修改描述或 dueComplete。媒体附件名为 `homework-review-image-<UUID>.<ext>` 或 `homework-review-audio-<UUID>.<ext>`，上传时提供 `file`、`name` 及对应 MIME。确认响应为原生上传后，再发布评论。评论格式为全角冒号前缀 `作业批改：`、换行和一个 JSON 对象：

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

`~/.cyeam/trello-reviews/<UUID>.json` 以 0600 权限保存正文、内容摘要、上传及发布阶段、附件 ID、action ID，不保存凭据。写请求前先保存意图，更新用临时文件同步并原子替换；同一状态文件用独占锁防止并发发布。同一 UUID 不可改卡片、类型、内容或媒体扩展名。

失败后重试会查询卡片附件和分页评论，复用已成功的原生附件或 action ID；评论恢复忽略提交前已有的同文评论。成功的本地记录直接返回。查询失败会停止。请求结果未知且未找到匹配记录时，也会停止，避免 Trello 无原生幂等键时重复发布。稍后用同一 UUID 再次查询；人工确认写入失败后，可显式添加 `--retry-uncertain` 允许再次 POST，此操作有重复风险。

保留 JSON 状态直到不再需要重试。进程异常退出可能留下 `<UUID>.json.lock`，确认没有进程正在使用该提交后可移除锁，再用原 UUID 重试。状态及锁在当前设备生效；不要跨设备并行提交同一轮批改。新一轮使用新 UUID，即使文字相同也追加新的历史；不要删除历史评论引用的附件。修改旧评论的编辑功能不在本命令范围内。

## 验证

自动测试通过本地模拟 Trello 验证三种协议、MIME、上传先于评论、丢失响应后的恢复、相同文字新历史、未知结果停止、分页与非法输入；测试不写真实卡片。

实机验收需在指定测试卡分别提交文字、图片、音频，检查最新在前、历史保留、原照片/录音及普通评论不成为批改。平板端还需验证无批改、附件被删除、网络中断、切换作业、离开页面及切后台停止播放、刷新恢复。CLI 实现不覆盖平板 UI 或播放生命周期。
