# cyeam-cli

cyeam 命令行工具，提供架构咨询、日期查询、路书分享、OneDrive 云笔记、书法字形处理等功能。

## 安装

### 首次安装

从 GitHub Releases 下载最新版本：

```bash
# macOS (Apple Silicon)
curl -L https://github.com/mnhkahn/cyeam-cli/releases/latest/download/cyeam_Darwin_arm64.tar.gz | tar xz
chmod +x cyeam
sudo mv cyeam /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/mnhkahn/cyeam-cli/releases/latest/download/cyeam_Darwin_x86_64.tar.gz | tar xz
chmod +x cyeam
sudo mv cyeam /usr/local/bin/

# Linux (amd64)
curl -L https://github.com/mnhkahn/cyeam-cli/releases/latest/download/cyeam_Linux_x86_64.tar.gz | tar xz
chmod +x cyeam
sudo mv cyeam /usr/local/bin/

# Windows
# 下载 cyeam_Windows_x86_64.zip 并解压，将 cyeam.exe 添加到 PATH
```

### 更新

```bash
cyeam update
```

## 使用

### 账号

```bash
# 登录 Microsoft 账号，用于 OneDrive 路书和云笔记
# 登录后会使用 refresh token 自动刷新访问令牌
cyeam login

# 查看当前登录状态
cyeam whoami

# 退出登录
cyeam logout
```

### 架构咨询

```bash
# 快速模式（默认）
cyeam ask "如何设计微服务架构？"

# 深度思考模式
cyeam ask --mode think "如何设计微服务架构？"

# 专家模式
cyeam ask --mode expert "如何设计微服务架构？"

# 搜索 cyeam.com
cyeam ask search "微服务"
```

### 日期查询

```bash
# 获取今日节日
cyeam date holiday

# 获取指定日期节日
cyeam date holiday 2024-01-01
```

### 路书分享

```bash
# 列出 OneDrive 路书
cyeam roadbook list

# 分享路书
cyeam roadbook share route.json

# 获取路书
cyeam roadbook get <id>
```

### 书法字形（Mo）

```bash
# 生成行书古文字形数据
cyeam mo guwen "永"
cyeam mo guwen --ai-compose "永"  # 启用 AI 合成缺失字形

# 获取行书字形候选
cyeam mo char detail "永"

# 获取字形构成
cyeam mo char composition "永"

# 合成行书字符图片
cyeam mo char compose "永" --out yong.png

# OCR 行书图片
cyeam mo ocr calligraphy.png
```

### CNote 云笔记

```bash
# 列出 OneDrive Notes 目录下的笔记，并显示可点击的打开链接
cyeam cnote list

# 读取笔记详情，默认输出 Markdown 风格文本
cyeam cnote get "日记"

# 读取笔记详情，输出纯文本
cyeam cnote get "日记" --format text

# 新建笔记，内容从 stdin 读取
cyeam cnote new "日记" < note.html

# 追加笔记内容
cyeam cnote append "日记" < more.html
```

### 邮件

```bash
# 将指定邮箱 INBOX 中的全部未读邮件一键标为已读
cyeam mail mark-all-read <account>
```

### 英语音标

```bash
# 查询英语单词的英式/美式音标和简明释义
cyeam phonetic hello

# 人类可读输出
cyeam --pretty phonetic hello
```

### Git 版本标签

在目标 Git 仓库内执行：

```bash
cyeam tag minor --dry-run  # 预览小版本升级
cyeam tag minor            # 小版本升级：v0.2.15 -> v0.3.0
cyeam tag major            # 大版本升级：v0.2.15 -> v1.0.0
```

安装项目 skills 后，可显式调用 `git-tag` 技能：`/git-tag` 默认小版本升级，
`/git-tag major` 大版本升级，`/git-tag minor --dry-run` 预览；使用 `$` 调用技能的客户端可写 `$git-tag`。
显式调用会直接执行，仍保留工作区和标签检查。`git-cmsg` 会将仅打标签的请求交给该技能。

以本地最大的稳定版本标签 `vMAJOR.MINOR.PATCH` 为基准，忽略预发布及其他格式标签；
没有匹配标签时从 `v0.0.0` 开始。小版本增加次版本号，大版本增加主版本号，并清零后续位。
工作区必须干净（包括暂存和未跟踪文件），标签指向当前 HEAD。
命令只创建本地标签，不自动拉取或推送；如需同步远端标签，请先运行 `git fetch --tags`。
确认后使用 `git push origin <新标签>` 发布，在本项目中会触发 GitHub Release 工作流。

### 版本和更新

```bash
# 查看版本
cyeam version

# 更新工具
cyeam update
```
