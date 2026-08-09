# 20260503-2 使用手册

## 简介

本次迭代调整了定时任务的模型 Token 使用方式：

- 创建备忘录（定时任务）时不再保存模型 Token
- 执行任务时再根据模型名称动态从 SQLite `token_store` 读取对应密钥

## 行为说明

- `task_meta` 不保存 token
- `task_detail` 也不保存 token
- `POST /api/cron/create` 只接收任务内容、模型、时间、周期、`chatId` 等元数据
- 后台执行链路会根据 `model` 查询 `/api/token` 持久化到 SQLite 的 `token_store`
- 查询到 token 后，会在请求上游 `/v1/chat/completions` 时写入 `Authorization`

## 影响范围

- proxy 的 cron 创建逻辑
- integration 的 cron 创建逻辑
- integration 的 cron 执行逻辑
- site 创建备忘录时的请求体

## 依赖

- 需要先通过 `POST /api/token` 保存模型与密钥
- 如果某个模型在 `token_store` 中没有对应密钥，执行时不会自动从任务元数据回退，因为任务元数据已不再保存 token
