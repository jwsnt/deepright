# 备忘录 API

## 接口

`POST /api/cron/create?agentId=xxx`

## 请求体

```json
{
  "content": "任务内容",
  "model": "OpenAI",
  "thinking": true,
  "rawTime": "2026-04-30 12:10",
  "cycle": 0
}
```

自定义 Cron 表达式：

```json
{
  "content": "任务内容",
  "model": "OpenAI",
  "thinking": true,
  "cycle": -1,
  "cron": "10 12 * * 1-5"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| content | string | 备忘录内容 |
| model | string | 选择的模型 |
| thinking | boolean | 是否深度思考 |
| rawTime | string | 日期+时间, 格式 yyyy-MM-dd hh:mm（cycle≥0 时必填） |
| cycle | int | 0=仅一次, 1=工作日, 2=自然日, -1=自定义 Cron |
| cron | string | 自定义 Cron 表达式（cycle=-1 时必填，5字段格式） |

## 响应

```json
{
  "status": 0,
  "id": 1,
  "cron": "0 10 12 30 4 ? 2026",
  "agentId": "A"
}
```

## 存储

数据写入 `data` SQLite 文件, 与 cron 模块共享。
