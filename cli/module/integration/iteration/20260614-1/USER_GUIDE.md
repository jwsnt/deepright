# 20260614-1 USER_GUIDE

本次迭代为 `integration` 新增了 Agent 导入与导出能力，便于备份单个 Agent 或在机器间迁移。

## HTTP 接口

- `GET /api/agent/export?agent_id=xxx`
- 返回一个 zip 文件
- zip 内会保留顶层 Agent 目录
- 导出时会过滤该 Agent 一级目录中的 `chrome*`、`data`、`tmp`

- `POST /api/agent/import`
- 支持导入 export 生成的 zip，或导入一个完整 Agent 目录结构
- 如果待导入 Agent 与现有目录重名，会直接拒绝导入，并提示先删除同名 Agent
- zip 导入会先解压，再完成导入，并清理临时 zip 文件

## CLI

- `integration agent export --agent DEF_AGENT --output ./DEF_AGENT.zip`
- `integration agent import --input ./DEF_AGENT.zip`
- `integration agent import --input /path/to/DEF_AGENT`

## 说明

- 导入成功后会刷新 Agent 元数据缓存
- 这次变更不修改现有 Agent 初始化、删除、配置保存等行为
