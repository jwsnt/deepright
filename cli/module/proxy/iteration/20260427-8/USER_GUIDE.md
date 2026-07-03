# 会话记录 SQLite 持久化

## 功能说明

转发 `/v1/chat/completions` 时，将请求和 SSE 响应以原始报文格式保存到共享的 `data` SQLite 数据库的 `chat_log` 表中。

## 存储表结构

```sql
CREATE TABLE chat_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  agent_id TEXT NOT NULL,
  chat_id TEXT NOT NULL,
  chat_type TEXT NOT NULL, -- user=用户会话, cron=定时任务
  role TEXT NOT NULL,      -- Q=提问, A=回答, X=主动取消
  response_type TEXT NOT NULL, -- normal=正常, abnormal=异常
  content TEXT NOT NULL,
  created_at TEXT NOT NULL  -- ISO 时间戳
)
```

## 写入方式

- Q（提问）：请求到达时异步写入完整请求 JSON body
- A（回答）：SSE 响应按增量块异步写入，收到一段保存一段，保持原始 SSE 报文内容
- X（取消）：主动取消时写入空内容标记
- 正常结束：会额外写入一条 `data: [DONE]` 的 A 记录，供恢复方识别终态
- `chat_type` 用于区分记录来源：普通网页会话写 `user`，定时任务链路写 `cron`
- `response_type` 用于区分响应结果：正常 SSE 写 `normal`，异常原因写 `abnormal`
- 每条记录的时间戳在实际写入时生成，精度为毫秒

## 说明

- 与 Cron 模块共享 `data` SQLite 文件
- Integration 使用全局连接池，Proxy 使用每次请求的连接
- 恢复会话时，单轮回答可能对应多条连续的 `A` 增量记录，需按顺序拼接还原
