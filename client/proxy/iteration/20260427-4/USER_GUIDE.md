# 备忘录元数据查询 API

## 接口

`POST /api/cron/detail/metadata?agentId=xxx`

## 参数

| 参数 | 类型 | 说明 |
|------|------|------|
| agentId | string | Agent ID（URL query 参数，必填） |

## 响应

```json
{
  "status": 0,
  "data": [
    {
      "id": 1,
      "cycle": 0,
      "rawTime": "2026-04-30 12:10",
      "agentId": "A",
      "model": "OpenAI",
      "thinking": true,
      "cron": "0 10 12 30 4 ? 2026",
      "content": "查看天气"
    }
  ]
}
```

## 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | 元数据 ID |
| cycle | int | 0=仅一次, 1=工作日, 2=自然日, -1=自定义 Cron |
| rawTime | string | 原始时间（自定义 Cron 时为空） |
| agentId | string | 绑定的 Agent |
| model | string | 选择的模型 |
| thinking | boolean | 是否深度思考 |
| cron | string | Cron 表达式 |
| content | string | 任务内容 |

## 错误

- 405: 非 POST 请求
- status=1: agentId 缺失或数据库错误
