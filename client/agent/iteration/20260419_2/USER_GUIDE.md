# Agent 迭代 20260419_2 使用手册

## 变更说明

新增 `GetSkillNames` 子模块 API 和 `--skills` CLI 参数，从 Agents 数组中获取指定 AgentId 的所有 Skills 名称列表。

## CLI 使用

```bash
./agent-scanner --skills <agentId> <directory>
```

### 示例

```bash
./agent-scanner --skills a ./test-case
# 输出: ["__internal_F"]

./agent-scanner --skills b ./test-case
# 输出: ["__internal_A", "__internal_c", "__internal_F"]
```

## 子模块调用

```go
names, err := GetSkillNames(root, deviceID, ttl, "a")
// names: ["__internal_F"]
```

## 参数说明

| 参数 | 说明 |
|------|------|
| `--skills <id>` | 指定 agentId，返回其 Skills 名称的 JSON 数组 |

## 错误处理

- agentId 不存在时返回错误信息并退出
