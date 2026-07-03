# 20260503-18 使用手册

## 目标

本次迭代为 `proxy` 补齐了“备忘录任务明细执行时自动附带任务类型 metadata”的能力：

- 备忘录任务明细在请求 `/v1/chat/completions` 时，metadata 会新增 `cron_type`
- 普通周期任务统一写入 `cron_type=cron`
- 插件桥接生成的任务会写入对应插件 `key`，例如 `cron_type=feishu`
- 原有 metadata 字段保持不变，`META_ID` 等已有逻辑继续生效

## 生效范围

- 后台 cron 执行器扫描到的普通周期任务
- `connect add-request` 聚合后生成并立即执行的插件桥接任务
- `connect add-request` 超过 20 分钟后生成的“无需启动”备忘录明细，在后续真正执行时也会携带对应 `cron_type`

## 请求示例

普通周期任务执行时，请求体中的 metadata 会包含：

```json
{
  "metadata": {
    "agentId": "A",
    "chat": "chat-001",
    "type": "scheduled_task",
    "cron_type": "cron"
  }
}
```

插件桥接任务执行时，请求体中的 metadata 会包含：

```json
{
  "metadata": {
    "agentId": "A",
    "chat": "chat-feishu",
    "type": "scheduled_task",
    "cron_type": "feishu",
    "META_ID": "123"
  }
}
```

## 行为说明

- `cron_type` 的值来源于任务明细的 `task_type`
- 当 `task_type` 为空时，会自动回落为默认值 `cron`
- 当 `meta_ref` 中存在关联 request ID 时，仍会继续写入 `META_ID`
- 该能力只补充 metadata，不改变原有任务调度、执行和回推逻辑

## 验证

本次迭代已补充自动化测试，覆盖：

- 插件桥接任务执行时 metadata 正确写入 `cron_type=feishu`
- 普通周期任务执行时 metadata 正确回落为 `cron_type=cron`
- `meta_ref` 为空时不会额外写入 `META_ID`
