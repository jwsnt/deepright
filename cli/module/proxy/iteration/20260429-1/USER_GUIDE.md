# 备忘录定时任务自动执行

## 功能说明

每分钟扫描到期的备忘录任务明细，自动转为 `/v1/chat/completions` 请求发送到上游，并将请求和响应记录到会话存储。

## 召回条件

- `started = 0`（未启动）
- `exec_time <= 当前时间`（已到期）
- `exec_time >= 当前时间 - 1小时`（不超过1小时，超过则放弃）

## 状态流转

| started | 含义 | 时机 |
|---------|------|------|
| 0 | 待启动 | 初始状态 / 失败回滚 |
| 1 | 已启动 | 开始执行时立即更新 |
| 2 | 无需启动 | 用户手动忽略 |
| 3 | 已完成 | SSE 响应完成后更新 |

- 开始执行 → `started=1`
- SSE 完成 → `started=3`
- 中途失败（metadata获取失败、请求失败）→ 回滚 `started=0`，下次定时器重试

## 执行流程

1. 扫描到期明细，标记 `started=1`
2. 构造请求：content=任务内容，model/thinking=明细对应值
3. 会话ID = `{metaId}@{detailId}`
4. 注入 metadata（含 Agent/Skills）
5. Q 写入 `chat_log`（chat_type=cron）
6. 发送到上游，读取 SSE 响应
7. A 写入 `chat_log`，更新 `started=3`
