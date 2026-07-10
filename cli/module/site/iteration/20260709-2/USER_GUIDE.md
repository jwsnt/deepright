# Site 迭代手册（20260709-2）

## 本次更新

- 当前会话沙盒的状态查询、缓存、恢复、切换刷新都改为仅按 `chatId`
- 前端读取状态改为 `/api/sandbox_status?chatId=...`；即使当前没有选中 `agentId`，只要存在 `chatId` 也会恢复当前会话沙盒状态
- 写入接口仍使用 `/api/sandbox=filepick|net|filepick_net|off?agentId=...&chatId=...`；其中 `agentId` 只作为后端日志输入
- 同一 `chatId` 下切换不同 `agentId` 时，沙盒按钮高亮、浮层模式与更新时间都会保持一致
- 前端仍不区分 macOS 与 Windows/WSL；后端会自动选择对应 helper 与目录授权流程

## 用户可见行为

- `目录权限`、`离线执行`、`双重限制`、`关闭沙盒` 四个入口与现有引导、浮层布局保持不变
- 浮层仍会展示当前选中的 `agentId`，但该展示值不参与沙盒状态查询、缓存命中与恢复判断
- 如果后端返回 WSL helper / `bubblewrap` 不可用，页面会展示失败提示，但不会影响当前会话其它交互
