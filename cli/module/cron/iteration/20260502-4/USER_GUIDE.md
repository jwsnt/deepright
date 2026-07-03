# 20260502-4 使用手册

## 简介

本迭代收敛了 Cron 模块的存储边界：

- `task_meta` 不再存储模型 token（密钥）
- `task_detail` 也不存储模型 token（密钥）
- Cron 只保留任务调度执行所需的元数据，如 `agentId`、`chatId`、`model`、`thinking`、`content`

如果调用方需要真正执行任务，应在执行时根据 `model` 动态提供或查询 token。

## 存储约束

Cron 侧任务元数据结构：

- `cycle`
- `rawTime`
- `agentId`
- `chatId`
- `model`
- `thinking`
- `cron`
- `content`

Cron 侧任务明细结构：

- `metaId`
- `execTime`
- `agentId`
- `chatId`
- `model`
- `thinking`
- `content`
- `started`

以上两张表都不包含 token 字段。

## 行为说明

### 创建任务

- 无论通过子模块调用还是 CLI 创建任务，Cron 都只保存模型名称，不保存模型密钥
- 创建时仍会校验模型是否已在共享 `token_store` 中注册，避免生成无法执行的任务

### 执行任务

- Cron 模块本身只负责任务元数据与任务明细管理
- 实际执行方应在执行时按 `model` 动态查询 token
- 这样可以避免 token 在任务元数据、任务明细中重复落库

## 集成对齐

与 integration 的对齐行为：

- `/api/cron/create` 创建任务时不会把 token 写入 `task_meta`
- 周期执行链路会在真正请求上游前，按 `model` 从共享 `token_store` 动态读取 token

## 验收要点

- 查看 `task_meta` 表：没有 token 字段
- 查看 `task_detail` 表：没有 token 字段
- 创建任务后，数据库里只保留 `model`，不保留对应密钥
- 集成执行任务时，仍可按 `model` 正常获取 token 并转发请求
