# Site 迭代 20260425_4 使用手册

## 变更说明

在设置面板的 Agent 下拉框左侧新增 Thinking/Auto 切换 Checkbox。

## 功能说明

- 选择 Agent 后，下拉框左侧出现 Checkbox，显示当前 Thinking 状态
- 勾选为 Thinking 模式（config.json 中 thinking=true），取消勾选为 Auto 模式（thinking=false）
- 切换后立即保存到 config.json，无需额外点击保存按钮
- 每个 Agent 的 Thinking 配置独立
- 切换 Agent 时自动读取对应 Agent 的实时配置
- Agent 绑定变更时（新建会话、切换会话、浮层选择 Agent）异步检查 config.json 更新状态

## 技术说明

- `checkSwarmStatus` 合并为 `checkAgentConfig`，一次请求同时获取 swarm 和 thinking 状态
- 使用 `_configCheckSeq` 序列号防止异步竞态
- fetch 请求带时间戳参数防缓存
