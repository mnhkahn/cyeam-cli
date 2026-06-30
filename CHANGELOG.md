
<a name="v0.1.42"></a>
## [v0.1.42](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.41...v0.1.42) (2026-06-30)

### Code Refactoring


- **pinyin:** 切换拼音转换为独立 go-pinyin 包

### Documentation


- **pdf:** 更新PDF技能文档，补充多种输入方式说明

### Features


- **skills:** 新增 git-cmsg 提交信息生成 skill

- **update:** 新增二进制更新时的 checksum 校验

### Miscellaneous Tasks


- **deps:** 切换 go-pinyin 为远程 v0.1.0 正式版


<a name="v0.1.41"></a>
## [v0.1.41](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.40...v0.1.41) (2026-06-29)

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.40"></a>
## [v0.1.40](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.39...v0.1.40) (2026-06-28)

### Bug Fixes


- **news:** 修复 pretty 表格列宽异常问题

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.39"></a>
## [v0.1.39](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.38...v0.1.39) (2026-06-28)

### Features


- **news:** 恢复 AI 新闻数据源，使用 MCP curl 从 RSS 源获取

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.38"></a>
## [v0.1.38](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.37...v0.1.38) (2026-06-28)

### Features


- **news:** 优化图片提取与描述处理

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.37"></a>
## [v0.1.37](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.36...v0.1.37) (2026-06-28)

### Features


- **news:** 改用 MCP 直接获取新闻并提取图片

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.36"></a>
## [v0.1.36](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.35...v0.1.36) (2026-06-28)

### Documentation


- **news:** 增加表格展示建议

- **skills:** 统一添加【重要】提示，要求必须读原文

### Features


- **news:** news get 支持 --pretty 输出表格格式

- **news:** news get 输出结构化 JSON，不再转成 Markdown

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.35"></a>
## [v0.1.35](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.34...v0.1.35) (2026-06-27)

### Documentation


- **news:** 修正数据源名称 ai-news-zh → ai_news

- **news:** 明确两个独立数据源：tech_news(技术动向) + ai-news-zh(AI资讯)

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.34"></a>
## [v0.1.34](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.33...v0.1.34) (2026-06-27)

### Features


- **news:** 更新 news skill，添加完整处理流程

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.33"></a>
## [v0.1.33](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.32...v0.1.33) (2026-06-26)

### Documentation


- **mail:** 更新 skill 文档，新增标记已读/未读命令说明

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.32"></a>
## [v0.1.32](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.31...v0.1.32) (2026-06-26)

### Features


- 优化 PDF 字体加载和 mail 标记功能

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.31"></a>
## [v0.1.31](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.30...v0.1.31) (2026-06-26)

### Features


- **mail:** 支持用户名从环境变量读取

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.30"></a>
## [v0.1.30](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.29...v0.1.30) (2026-06-25)

### Features


- **mail:** add --all flag to list messages from all configured accounts

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.29"></a>
## [v0.1.29](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.28...v0.1.29) (2026-06-25)

### Documentation


- **image:** 补全图片转换 skill 文档

### Features


- **cloudinary:** 新增 Cloudinary 图片上传 skill

- **update:** 添加 brew 风格的下载进度条和状态输出

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.28"></a>
## [v0.1.28](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.27...v0.1.28) (2026-06-25)

### Bug Fixes


- **skills:** 捕获 skill 同步输出并加超时，避免静默失败与卡死

### Features


- **image:** 新增 image convert 图片格式转换命令

- **mail:** 新增多邮箱 IMAP 收取与 SMTP 发送命令

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.27"></a>
## [v0.1.27](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.26...v0.1.27) (2026-06-23)

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.26"></a>
## [v0.1.26](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.25...v0.1.26) (2026-06-23)

### Bug Fixes


- **login:** 登录链接输出到 stdout 并与轮询解耦

### CI/CD


- **release:** 发版时推送飞书群机器人通知

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.25"></a>
## [v0.1.25](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.24...v0.1.25) (2026-06-23)

### Documentation


- **skill:** 统一登录指引到 cyeam-cli skill，移除 Fly.io 字样

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.24"></a>
## [v0.1.24](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.23...v0.1.24) (2026-06-21)

### Bug Fixes


- **login:** 添加调试日志，登录流程输出到 stderr 避免缓冲

- **login:** 登录提示输出到 stderr 避免被 cobra stdout 缓冲吞掉

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.23"></a>
## [v0.1.23](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.22...v0.1.23) (2026-06-21)

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.22"></a>
## [v0.1.22](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.21...v0.1.22) (2026-06-21)

### Features


- **auth:** 新增文件 token 后备存储和 --print-link 登录

- **roadbook:** 新增 csv 命令和完整 AI 执行流程

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.21"></a>
## [v0.1.21](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.20...v0.1.21) (2026-06-21)

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.20"></a>
## [v0.1.20](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.19...v0.1.20) (2026-06-21)

### Bug Fixes


- **pinyin:** 米字格内线改为虚线，实线看不清

### Features


- **tv:** today 命令默认展示已完成比赛

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.19"></a>
## [v0.1.19](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.18...v0.1.19) (2026-06-20)

### CI/CD


- **release:** 修复 changelog 推送时远程 main 已前进的冲突问题

### Code Refactoring


- **cli:** update命令输出改为纯文本

### Features


- **pdf:** 新增 Markdown/HTML 转 PDF 命令和 skill

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.18"></a>
## [v0.1.18](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.17...v0.1.18) (2026-06-19)

### Bug Fixes


- **tv:** 修复赛程查询的日期范围、NBA 403 及新增 tomorrow 命令

### Documentation


- skill 描述加不可直接作为工具名调用提示

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.17"></a>
## [v0.1.17](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.16...v0.1.17) (2026-06-19)

### Code Refactoring


- **live-broadcast:** 精简 skill，禁止预加载/缓存/分析

### Features


- **ai:** 新增 ai-models skill 和 cyeam ai models 命令

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.16"></a>
## [v0.1.16](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.15...v0.1.16) (2026-06-17)

### Features


- --help 纯文本输出，新增 --pretty 人类可读模式

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.15"></a>
## [v0.1.15](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.14...v0.1.15) (2026-06-17)

### Bug Fixes


- npx skills add 静默安装，版本号 v 前缀比对修复

### Miscellaneous Tasks


- update changelog [skip ci]

### Tests


- **update:** 使用真实版本号替换虚构的 v1.1.0


<a name="v0.1.14"></a>
## [v0.1.14](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.13...v0.1.14) (2026-06-17)

### Bug Fixes


- **skills:** 移除 -g 参数兼容 PromptScript

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.13"></a>
## [v0.1.13](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.12...v0.1.13) (2026-06-17)

### Features


- **pinyin:** 新增拼音查询和看拼音写字练习纸 PDF 功能

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.12"></a>
## [v0.1.12](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.11...v0.1.12) (2026-06-17)

### Code Refactoring


- **cli:** 清理 ask 遗留代码和 SKILL 文档

- **cli:** 移除 ask 命令，architect skill 改为 MCP 地址

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.11"></a>
## [v0.1.11](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.10...v0.1.11) (2026-06-17)

### Documentation


- **skills:** 强调模型必须翻译球队名并加国旗 emoji

### Features


- **skills:** 新增 news skill 查询 AI 新闻与科技资讯

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.10"></a>
## [v0.1.10](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.9...v0.1.10) (2026-06-17)

### Bug Fixes


- **skills:** 为所有 SKILL.md 添加 version 字段修复全局安装

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.9"></a>
## [v0.1.9](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.8...v0.1.9) (2026-06-16)

### Features


- **cli:** 统一所有输出为 JSON 信封格式

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.8"></a>
## [v0.1.8](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.7...v0.1.8) (2026-06-16)

### Code Refactoring


- **tv:** 移除队伍名中英映射，交由模型翻译

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.7"></a>
## [v0.1.7](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.6...v0.1.7) (2026-06-16)

### Features


- **cli:** JSON 信封输出、更新通知与技能同步

### Miscellaneous Tasks


- update changelog [skip ci]


<a name="v0.1.6"></a>
## [v0.1.6](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.5...v0.1.6) (2026-06-15)

### CI/CD


- 跑 GoReleaser 前彻底删除旧 Release 和 tag

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
## [v0.1.3](https://github.com/mnhkahn/cyeam-cli/compare/skill-6...v0.1.3) (2026-06-12)

### Documentation


- add cnote get implementation plan

- add cnote get design

- **cyeam-cli:** 更新SKILL.md中aria2c下载的默认输出路径

### Features


- **cnote:** add cnote get command with markdown/text format support

- **tv:** add tv schedule command for NBA / World Cup / China national football


<a name="skill-6"></a>
## [skill-6](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.2...skill-6) (2026-06-12)

### Documentation


- **skills:** add cyeam-cli installation steps for different platforms


<a name="v0.1.2"></a>
## [v0.1.2](https://github.com/mnhkahn/cyeam-cli/compare/skill-5...v0.1.2) (2026-06-12)


<a name="skill-5"></a>
## [skill-5](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.1...skill-5) (2026-06-12)

### Documentation


- **cli:** 设计共享终端表格样式

### Features


- **cli:** 支持微软令牌刷新和笔记链接

- **cli:** 统一列表表格输出样式

### Miscellaneous Tasks


- 更新gitignore并添加终端表格设计 brainstorm 笔记


<a name="v0.1.1"></a>
## [v0.1.1](https://github.com/mnhkahn/cyeam-cli/compare/skill-4...v0.1.1) (2026-06-11)


<a name="skill-4"></a>
## [skill-4](https://github.com/mnhkahn/cyeam-cli/compare/skill-3...skill-4) (2026-06-11)

### Bug Fixes


- skill release 标记为预发布，latest 指向正式版本

### Code Refactoring


- 移除slogan相关功能，重构日期节日查询逻辑

### Documentation


- **cli:** 同步最新命令说明

- **release:** 移除 Homebrew 后续支持说明

### Features


- 新增微软登录、OneDrive 集成及笔记功能


<a name="skill-3"></a>
## [skill-3](https://github.com/mnhkahn/cyeam-cli/compare/skill-2...skill-3) (2026-06-10)

### Bug Fixes


- 修复 README 下载链接，使用具体版本号


<a name="skill-2"></a>
## [skill-2](https://github.com/mnhkahn/cyeam-cli/compare/v0.1.0...skill-2) (2026-06-10)

### Bug Fixes


- 简化 skill 发布工作流，移除跨仓库写入

### CI/CD


- 添加 skill 发布工作流，自动提交到各平台

### Documentation


- 新增 README 文档，包含安装说明和命令使用指南

### Miscellaneous Tasks


- 添加 .gitignore，忽略压缩文件和构建产物


<a name="v0.1.0"></a>
## v0.1.0 (2026-06-10)

### CI/CD


- **github workflow:** 优化飞书通知的环境变量配置

### Features


- 初始化cyeam-cli命令行工具项目

