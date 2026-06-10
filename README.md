# cyeam-cli

cyeam 命令行工具，提供架构咨询、日期查询、路书分享、书法字形处理等功能。

## 安装

### 首次安装

从 GitHub Releases 下载最新版本：

```bash
# macOS (Apple Silicon)
curl -L https://github.com/mnhkahn/cyeam-cli/releases/download/v0.1.0/cyeam_Darwin_arm64.tar.gz | tar xz
chmod +x cyeam
sudo mv cyeam /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/mnhkahn/cyeam-cli/releases/download/v0.1.0/cyeam_Darwin_x86_64.tar.gz | tar xz
chmod +x cyeam
sudo mv cyeam /usr/local/bin/

# Linux (amd64)
curl -L https://github.com/mnhkahn/cyeam-cli/releases/download/v0.1.0/cyeam_Linux_x86_64.tar.gz | tar xz
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
# 获取今日标语
cyeam date slogan

# 获取指定日期标语
cyeam date slogan 2024-01-01

# 获取今日节日
cyeam date holiday

# 获取指定日期节日
cyeam date holiday 2024-01-01
```

### 路书分享

```bash
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

### 其他

```bash
# 查看版本
cyeam version

# 更新工具
cyeam update
```
