# 备忘录任务明细状态更新 API

## 接口

`POST /api/cron/detail/status?agentId=xxx&detailId=yyy&status=zzz`

## 参数

| 参数 | 类型 | 说明 |
|------|------|------|
| agentId | string | Agent ID（必填） |
| detailId | string | 任务明细 ID（必填） |
| status | string | 新状态值（必填）：0=未启动, 1=已启动, 2=无需启动 |

## 响应

```json
{"status": 0, "affected": 1}
```

## 说明

- Integration 使用全局 cronDB 连接，不单独打开数据库
- 更新条件同时匹配 detailId 和 agentId，防止跨 Agent 操作
