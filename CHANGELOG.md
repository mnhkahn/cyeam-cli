
<a name="v0.1.6"></a>
## [v0.1.6](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.5...v0.1.6) (2026-06-15)

### CI/CD


- 修复 webhook changelog 模板路径和继续执行

- GoReleaser mode replace 覆盖已有 asset 而非报错

- 跑 GoReleaser 前先删除已有 Release

- 修复 webhook changelog 模板文件路径

- 更新 CHANGELOG 后重新打 tag 保证 GoReleaser 一致性

- 新增 GoReleaser 构建发布和 webhook 通知

### Features


- **tv:** 从 ESPN/NBA API 解析并展示比赛比分

### Miscellaneous Tasks


- update changelog [skip ci]

- update changelog [skip ci]

- update changelog [skip ci]

- update changelog [skip ci]

- update changelog [skip ci]

- 接入 git-chglog，打 tag 时自动更新 CHANGELOG.md


<a name="v0.1.5"></a>
## [v0.1.5](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.4...v0.1.5) (2026-06-15)

### Bug Fixes


- **tv:** ESPN按美东时区分组，查询时扩一天避免遗漏

### Code Refactoring


- **update:** 用 GitHub 重定向替代 API 检查版本，绕过 IP 限流

### Features


- **update:** 通过 cyeam.com 代理 GitHub 版本检查，绕过 IP 限流


<a name="v0.1.4"></a>
## [v0.1.4](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.3...v0.1.4) (2026-06-14)

### Code Refactoring


- **skill:** 将架构师 skill 的 name 和 description 改为英文

### Features


- **news:** 新增 news 命令，支持日期范围和单日查询

- **skills:** 新增独立子命令 skill（架构师/假期/笔记/直播预报/书法/路书）


<a name="v0.1.3"></a>
## [v0.1.3](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.2...v0.1.3) (2026-06-12)

### Documentation


- add cnote get implementation plan

- add cnote get design

- **cyeam-cli:** 更新SKILL.md中aria2c下载的默认输出路径

- **skills:** add cyeam-cli installation steps for different platforms

### Features


- **cnote:** add cnote get command with markdown/text format support

- **tv:** add tv schedule command for NBA / World Cup / China national football


<a name="v0.1.2"></a>
## [v0.1.2](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.1...v0.1.2) (2026-06-12)

### Documentation


- **cli:** 设计共享终端表格样式

### Features


- **cli:** 支持微软令牌刷新和笔记链接

- **cli:** 统一列表表格输出样式

### Miscellaneous Tasks


- 更新gitignore并添加终端表格设计 brainstorm 笔记


<a name="v0.1.1"></a>
## [v0.1.1](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.0...v0.1.1) (2026-06-11)

### Bug Fixes


- skill release 标记为预发布，latest 指向正式版本

- 修复 README 下载链接，使用具体版本号

- 简化 skill 发布工作流，移除跨仓库写入

### CI/CD


- 添加 skill 发布工作流，自动提交到各平台

### Code Refactoring


- 移除slogan相关功能，重构日期节日查询逻辑

### Documentation


- 新增 README 文档，包含安装说明和命令使用指南

- **cli:** 同步最新命令说明

- **release:** 移除 Homebrew 后续支持说明

### Features


- 新增微软登录、OneDrive 集成及笔记功能

### Miscellaneous Tasks


- 添加 .gitignore，忽略压缩文件和构建产物


<a name="v0.1.0"></a>
## v0.1.0 (2026-06-10)

### CI/CD


- **github workflow:** 优化飞书通知的环境变量配置

### Features


- 初始化cyeam-cli命令行工具项目

