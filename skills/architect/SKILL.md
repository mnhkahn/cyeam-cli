---
name: architect
version: 0.1.13
description: Architecture consultancy — MCP-based knowledge base for system design, technical proposals, code review, and more. Use the MCP server directly instead of CLI.
---

# 架构师知识库 (MCP)

## MCP 服务

知识库通过 MCP 协议暴露，AI Agent 直接连接即可使用，无需经过 CLI。

```
mcp: https://www.cyeam.com/arch/mcp
```

## 功能

Agent 连上后通过 MCP discovery 自动获取可用的 prompts、tools、resources，包括但不限于：

- 搜索架构知识库
- 获取系统设计方案
- 技术选型建议
- 代码审查

## 说明

- 不需要登录
- 不需要安装 cyeam CLI
- Agent 直接调 MCP 协议，比走 CLI 更直接
- 此 Skill 仅用于描述 MCP 服务地址，不涉及 CLI 命令