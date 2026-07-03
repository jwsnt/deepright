# Site 迭代手册（20260515-1）

## 本次更新

- 由左侧 `Skill` 小图标触发的自动提炼请求，在 SSE 结束后不会再根据本轮日志重新点亮自己
- 即使这一轮日志里真实出现了 `cli/get` 或 `cli/pub`，左侧 `Skill` 小图标也不会再次闪动
- `Skill` 小图标发起的请求会在结束后保持熄灭，避免自触发回流

## 当前行为

1. 当前会话最新一轮 SSE 完成后，页面会调用 `/log_skill_status`
2. 如果最近一轮日志命中 `cli/get` 或 `cli/pub`，左侧 `Skill` 小图标会点亮
3. 点击该图标并完成日志导出、填写提炼目标后，页面会自动向当前会话发起提炼请求
4. 这条请求会附带：

```text
metadata.extract = "true"
metadata.requestSource = "skill_extract"
```

5. 提炼请求结束后，页面不会再针对这一轮重新调用 `/log_skill_status` 检查点亮状态
6. 即使这次提炼链路里又真实产生了 `cli/get` 或 `cli/pub`，左侧 `Skill` 小图标也不会再次闪动
7. 后续只有新的普通请求完成后，再按常规日志结果重新判断是否点亮

## 说明

- `metadata.requestSource = "skill_extract"` 用于标记该轮请求来源，并阻断本轮自触发点亮
- 这条阻断只作用于 `skill_extract` 当前这一轮，不影响后续普通请求的正常判断
- 这次更新不改变原有提炼浮层、日志导出流程和 `metadata.extract` 的转发规则
