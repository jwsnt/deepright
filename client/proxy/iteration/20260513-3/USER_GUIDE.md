# Proxy 迭代手册（20260513-3）

## 本次更新

- 新增 `GET /log_skill_status?agentId=xxx&chatId=yyy`
- 接口会直接查询共享日志数据库 `agent_message_log`
- 用于判断指定 `Agent + Chat` 在最近一轮或最近 N 轮范围内是否执行过命令流程
- 返回结果可被 `site` 等模块直接复用，不需要先导出日志文件再做二次解析

## HTTP 用法

```bash
curl 'http://127.0.0.1:9876/log_skill_status?agentId=A&chatId=chat-001'
curl 'http://127.0.0.1:9876/log_skill_status?agentId=A&chatId=chat-001&round=1'
curl 'http://127.0.0.1:9876/log_skill_status?agentId=A&chatId=chat-001&round=3'
```

返回示例：

```json
{
  "status": 0,
  "agentId": "A",
  "chatId": "chat-001",
  "round": 1,
  "hasSkill": true,
  "hasCLIPub": true,
  "source": "db"
}
```

## 查询规则

- `agentId`：必填
- `chatId`：必填
- `round`：可选，默认 `1`
- `round` 的范围定义沿用 `20260513-2` 中 `log_skill` 的最近轮次概念
- 接口会先按最近 `N` 轮 `/v1/chat/completions` 请求收缩范围
- 然后检查该范围内是否出现 `log_type=2`（`cli/get`）或 `log_type=3`（`cli/pub`）
- 只要命中一次，就会返回 `hasSkill=true`

## 说明

- 数据源直接读取统一日志表 `agent_message_log`
- 当前判断口径等同于“指定轮次范围内是否出现过 `cli/get` 或 `cli/pub`”
- 该接口本身不导出文件，也不会改写日志内容
