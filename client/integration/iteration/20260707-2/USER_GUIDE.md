# Integration 迭代手册（20260707-2）

## 本次更新

- `integration` 在转发 `/v1/chat/completions`、上报 `/cli/get`，以及内部 `memo`、`email`、`feishu` 等最终转发到上游 `/v1/chat/completions` 的请求前，会统一补充顶层 `metadata.sandbox_path`
- `sandbox_path` 与 `knowledge` 同层，只表示当前会话目录白名单路径；不放进 `metadata.agent`、`metadata.agents[]` 或其他 Agent 维度字段
- 字段值固定来自共享 sqlite 中当前会话 `chatId` 对应的 `allowed_dir`；读取后会先做 `trim`
- 外部请求体如果手工传入 `metadata.sandbox_path`，最终也会被 integration 按当前会话真实状态重新计算并覆盖

## 注入规则

- 只有当前会话沙盒模式为 `filepick` 或 `filepick_net`，且 `allowed_dir` 在 `trim` 后非空时，才会带上顶层 `metadata.sandbox_path`
- 当前会话为 `net`、`off`、无记录，或 `allowed_dir` 为空字符串时，最终请求不会携带 `metadata.sandbox_path`
- 如果报文里原本残留旧的 `metadata.sandbox_path`，但当前会话已经没有有效目录，integration 会在转发前显式删除该字段
- `sandbox_path` 是会话维度字段，不与 `agentId` 绑定；同一 `chatId` 下切换不同 Agent 时，只要会话目录未变，最终值保持一致

## 示例

目标上游请求中的 `metadata` 片段示例：

```json
{
  "metadata": {
    "knowledge": {
      "path": "/tmp/knowledge"
    },
    "sandbox_path": "/Users/demo/Desktop"
  }
}
```

## 兼容性说明

- 现有 `metadata.agent.sandbox`、`metadata.agents[].sandbox` 继续保留，仍用于表达沙盒模式
- 本次新增的 `metadata.sandbox_path` 只表达“当前会话可访问的目录白名单路径”，不替代既有 `sandbox` 模式字段
- 主手册 `../../USER_GUIDE.md` 已同步补充这次顶层 `metadata.sandbox_path` 的统一说明
