# 会话记录恢复 API

## 接口

`POST /api/restore?agentId=xxx&chat=yyy&timeline=zzz&lastId=n`

## 参数

| 参数 | 类型 | 说明 |
|------|------|------|
| agentId | string | Agent ID（必填） |
| chat | string | 会话 ID（必填） |
| timeline | string | 起始时间，ISO 格式如 2026-04-28T00:00:00.123（必填，不包含该时间，建议使用毫秒精度） |
| lastId | int | 与 timeline 配合使用的最后一条记录 ID，默认 0；当时间相同时只返回更大的记录 ID |

## 响应

```json
{
  "status": 0,
  "data": [
    {
      "id": 1,
      "agentId": "D",
      "chatId": "uuid-xxx",
      "role": "Q",
      "content": "{\"model\":\"deepright\",...}",
      "createdAt": "2026-04-28T21:23:23"
    },
    {
      "id": 2,
      "agentId": "D",
      "chatId": "uuid-xxx",
      "chatType": "user",
      "role": "A",
      "responseType": "normal",
      "content": "data: {\"choices\":[{\"delta\":{\"content\":\"你\"}}]}\n",
      "createdAt": "2026-04-28T21:23:25.101"
    },
    {
      "id": 3,
      "agentId": "D",
      "chatId": "uuid-xxx",
      "chatType": "user",
      "role": "A",
      "responseType": "normal",
      "content": "data: {\"choices\":[{\"delta\":{\"content\":\"好\"}}]}\n",
      "createdAt": "2026-04-28T21:23:25.238"
    }
  ]
}
```

## 字段说明

| 字段 | 说明 |
|------|------|
| chatType | 会话类型：`user`=用户会话，`cron`=定时任务 |
| role | Q=提问, A=回答, X=主动取消 |
| responseType | 响应类型：`normal`=正常，`abnormal`=异常 |
| content | 原始报文（Q 为请求 JSON，A 为 SSE 增量报文） |
| createdAt | ISO 时间戳，支持毫秒精度 |

## 说明

结果按 id 升序（即时间顺序）返回，只包含严格晚于 `timeline` 的记录；当 `createdAt` 与 `timeline` 相同时，仅返回 `id > lastId` 的记录。
同一轮回答可能返回多条连续的 `A` 记录，调用方需按顺序拼接或增量合并。
正常完成时，最后还会返回一条内容为 `data: [DONE]` 的 `A` 记录，用于标识响应终态。
