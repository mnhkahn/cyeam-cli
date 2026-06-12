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

### 其他

```bash
# 查看版本
cyeam version

# 更新工具
cyeam update
```
