# Site 迭代手册（20260522-2）

## 本次更新

- 由左侧 `Skill` 小图标触发的自动提炼请求，不再附带旧字段 `metadata.extract`
- 同一条请求改为附带 `metadata.skill_extract = true`
- `metadata.requestSource = "skill_extract"` 的来源标记与原有阻断自触发点亮逻辑保持不变

## 当前行为

1. 当前会话最新一轮 SSE 完成后，页面会调用 `/log_skill_status`
2. 如果最近一轮日志命中 `cli/get` 或 `cli/pub`，左侧 `Skill` 小图标会点亮
3. 点击该图标并完成日志导出、填写提炼目标后，页面会自动向当前会话发起提炼请求
4. 这条请求会附带：

```text
metadata.skill_extract = true
metadata.requestSource = "skill_extract"
```

5. 旧字段 `metadata.extract` 不会再由这条自动提炼请求写入
6. 提炼请求结束后，页面不会再针对这一轮重新调用 `/log_skill_status` 检查点亮状态
7. 即使这次提炼链路里又真实产生了 `cli/get` 或 `cli/pub`，左侧 `Skill` 小图标也不会再次闪动
8. 后续只有新的普通请求完成后，再按常规日志结果重新判断是否点亮

## 说明

- `metadata.skill_extract = true` 用于标记这是由 `Skill` 小图标发起的提炼请求
- `metadata.requestSource = "skill_extract"` 继续用于标记该轮请求来源，并阻断本轮自触发点亮
- 这次更新不改变原有提炼浮层、日志导出流程和自动发送提炼提示词的内容
