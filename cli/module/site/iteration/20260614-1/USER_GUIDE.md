# 20260614-1 USER_GUIDE

本次迭代在设置中的 Agent 选择区新增了导入和导出入口。

## 导出

- 先在设置里选择当前 Agent
- 点击新增 Agent 小图标右侧的导出图标
- 页面会调用 `/api/agent/export?agent_id=当前Agent`
- 浏览器会自动下载 zip，并提示 `Agent 配置已开始下载`

## 导入

- 点击新增 Agent 小图标右侧的导入图标
- 浮层中可选择 `上传压缩包` 或 `上传目录`
- 选择后页面会调用 `/api/agent/import`
- 导入成功后，Agent 下拉框会立即刷新并自动切换到新导入的 Agent

## 限制

- 同名 Agent 不允许覆盖导入
- 如果需要替换，请先删除原 Agent，再重新导入
