# 定时任务明细 CHAT_ID

## 功能说明

为定时任务明细新增 `chatId` 字段，类型为字符串，可为空。

## 执行规则

- 当任务明细显式指定了 `chatId` 时，定时任务执行请求会直接使用该值作为会话 ID
- 当任务明细未指定 `chatId` 或为空时，仍沿用原有默认逻辑，使用 `metaId@detailId`

## 影响范围

- Proxy 侧共享的 `task_detail` 表结构
- `/api/cron/detail/list` 返回的任务明细字段
- 与 integration / cron 共用的任务明细读取语义
