# Agent 复制入口

本次迭代在设置里的 Agent 选择区增加了一个复制入口。

## 入口位置

- 只要当前 Agent 列表里存在可用 Agent，下拉框小三角左侧就会显示复制小图标
- 没有选中 Agent 时，图标会保持禁用态

## 交互流程

1. 在设置里先选中 source Agent
2. 点击下拉框内的复制小图标
3. 使用和“新增 Agent”相同的输入行填写新的 target Agent 名称
4. 点击 `复制`
5. 页面会先创建 target Agent，再调用 `/api/copy`

## 行为说明

- 复制期间会锁定中间界面，完成后自动解锁
- 成功后会刷新 Agent 下拉框，并自动切换到新的 target Agent
- `/api/copy` 只同步 `app/`、`data/`、`skills/`、`SOUL.md`、`USER.md` 和知识库目录，不会覆盖 target 自己的 `config.json`
