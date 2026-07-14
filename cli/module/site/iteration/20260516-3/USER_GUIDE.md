# 20260516-3 使用说明

- 右下角 `知识库WIKI` 标题栏新增一个小灯泡按钮，位置在展开图标右侧。
- 小灯泡默认显示为灰色；页面会通过 `/knowledge_lastUpdate` 判断知识库最后更新时间，超过主应用 `config/config.json.knowledge.interval` 指定的分钟阈值时切换为金色闪动。
- `knowledge.interval` 未配置、非法或小于等于 `0` 时，默认按 `60` 分钟处理。
- 如果当前 `Agent + Chat` 正在发送请求、等待 SSE 响应或恢复未完成请求，小灯泡会临时禁用，且不会展开确认浮层。
- 点击后会在中间会话区弹出确认浮层，提示 `检测到知识库，是否重新整理？`
- 浮层中会通过 `/knowledge_path` 读取真实知识库根目录，并预览即将发送的提示词：`按知识库WIKI要求重新整理 [FILE:/.../knowledge]`
- 其中知识库目录的绝对路径会以小气泡形式展示，方便确认目标目录；该路径与 `../../../knowledge/REQUIREMENT.md` 对应的知识库目录规则保持一致。
- 点击确认后，页面会在当前 `Agent + Chat` 中自动发送这条提示词。
- 仅通过该小灯泡触发的 SSE 请求，会显式在 `/v1/chat/completions` 请求体的 `metadata` 中附带 `knowledge_commit: true`。
- 当这次知识库整理请求的 SSE 完整结束后，右侧知识库内容区会自动刷新，并同步更新 `最近刷新` 时间。
