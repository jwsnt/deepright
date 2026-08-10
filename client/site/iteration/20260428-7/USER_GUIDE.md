# 迭代 20260428-7：备忘录视图统一刷新

## 功能说明

以下时机会立即刷新备忘录明细列表，如果元数据列表视图也处于激活状态则同时刷新：

1. 首次打开页面
2. 切换会话
3. 设置中切换 Agent 并保存
4. 保存备忘录成功后
5. 删除备忘录后

## 输入框默认值刷新

切换会话或 Agent 时，如果备忘录输入框无内容（仅 placeholder），立即以当前 AgentId 刷新 placeholder（`xxx的备忘录...`）。

## 技术实现

- `refreshCronViews()` 统一入口，同时刷新 timeline 和元数据列表（如果激活）
- `rsInitNote()` 在切换会话、保存设置、首次加载时调用，更新 placeholder
- `renderTimeline()` 带 200ms 防抖，避免高频重复请求
