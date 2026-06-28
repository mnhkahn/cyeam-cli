---
name: architect
version: 0.2.1
description: MCP-based architecture consultancy. 【重要】严格使用下面的MCP地址和流程，禁止瞎猜。
---

# 架构师问答 (纯 MCP Think 模式)

## MCP 服务地址

```
mcp: https://cyeam-wiki-mcp-production.up.railway.app/mcp
```

## 标准流程（正确的 MCP 使用方式）

### 1. Discovery：连接 MCP 服务获取能力清单

连接后自动 discovery，得到以下三类能力：
- Prompts 列表
- Tools 列表
- Resources 列表

### 2. 调用 MCP Prompt：`wiki_query_system`

这是核心的**向量组提示词**，直接调用作为系统提示词，包含：
- 架构师角色设定
- 知识库检索规则
- 输出格式要求

### 3. 调用 MCP Wiki Tools 检索上下文

根据用户问题，按优先级调用以下工具获取相关内容：

| 工具名 | 参数 |
|--------|------|
| `wiki_query` | `{"question": q, "depth": 2}` |
| `wiki_search_index` | `{"keyword": q}` |
| `wiki_get_graph` | `{"query": q, "depth": 1, "max_nodes": 50}` |

**后续动作：** 从工具返回结果中提取 `[[文章名]]` 标记，调用 `wiki_get_article` 获取完整文章内容。

### 4. 实时工具调用

问题需要实时数据时：
- 寻找 `curl` 或 `webFetch` 工具
- 生成要访问的 URL（不带 https:// 前缀）
- 调用工具：`{"url": "生成的URL"}`

### 5. 整合所有 MCP 上下文回答

- 系统提示词（来自 `wiki_query_system` prompt）
- Wiki 工具检索结果
- （可选）实时工具返回数据
- 末尾必须带上参考链接

---

## MCP 能力参考（Discovery 结果，按需使用）

**Prompts（先调用这个）：**
- `wiki_query_system` - 架构师知识库查询系统提示词

**Tools（根据需要调用）：**
- `wiki_query` - 深度问答
- `wiki_search_index` - 关键词搜索
- `wiki_get_graph` - 知识图谱
- `wiki_get_article` - 获取文章详情
- `curl` - 实时 HTTP 请求

**Resources（工具返回中引用到再读，不是上来就读）：**
- `wiki://index` - The master index of all wiki articles
- `wiki://backlinks` - Wiki Backlinks，文章反向链接索引
