# Agent Scanner 迭代 20260419 — AgentId 查询

## 新增功能

在原有完整输出基础上，新增两个查询能力：

1. 获取所有 AgentId 列表
2. 获取指定 AgentId 的元数据

## CLI 用法

```bash
# 列出所有 AgentId
./agent-scanner --list <目录>

# 获取指定 AgentId 的元数据
./agent-scanner --get <agentId> <目录>

# 原有功能不变：输出完整元数据
./agent-scanner <目录>
```

### 示例

```bash
# 列出所有 AgentId
./agent-scanner --list ./agents
# 输出: ["a", "b"]

# 获取 agent a 的元数据
./agent-scanner --get a ./agents
# 输出: {"workspace": "...", "agentId": "a", "soul": "...", ...}
```

## 子模块 API

```go
// 获取所有 AgentId 列表
ids, err := GetAgentIDs(root, deviceID, ttl)
// 返回 []string{"a", "b"}

// 获取指定 AgentId 的元数据
agent, err := GetAgentByID(root, deviceID, ttl, "a")
// 返回 *Agent 或 nil（未找到）
```

## 注意事项

- 新增功能共享原有的 Agent 元数据缓存
- 原有 `GetAgentOutput` 和无参数 CLI 行为完全不变
- 指定的 agentId 不存在时，CLI 返回错误退出，API 返回 nil
