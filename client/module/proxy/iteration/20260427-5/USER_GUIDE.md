# 备忘录任务明细查询 API

## 接口

`POST /api/cron/detail/list?agentId=xxx&date=yyyy-MM-dd`

## 参数

| 参数 | 类型 | 说明 |
|------|------|------|
| agentId | string | Agent ID（必填） |
| date | string | 日期，格式 yyyy-MM-dd（必填） |

## 响应

```json
{
  "status": 0,
  "data": [
    {
      "id": 1,
      "metaId": 1,
      "execTime": 1714200600,
      "agentId": "A",
      "model": "OpenAI",
      "thinking": true,
      "content": "查看天气",
      "started": 0
    }
  ]
}
```

## 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | 明细 ID |
| metaId | int | 关联的元数据 ID |
| execTime | int64 | 执行时间（Unix 时间戳） |
| agentId | string | 绑定的 Agent |
| model | string | 选择的模型 |
| thinking | boolean | 是否深度思考 |
| content | string | 任务内容 |
| started | int | 0=未启动, 1=已启动, 2=无需启动 |

## 查询范围

返回指定日期 00:00:00 至 23:59:59 内的所有任务明细，包括一次性和周期性任务。
