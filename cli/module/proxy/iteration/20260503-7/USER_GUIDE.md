# 20260503-7 使用手册

## 目标

本次迭代为 proxy 模块补充 Agent 模型与密钥相关的数据库审计日志。

## 新增日志表

- `proxy_agent_provider_log`

## 行为说明

- `/api/token` 对模型密钥的新增与更新，都会写入 `proxy_agent_provider_log`
- 日志表包含 `agent_id`、`chat_id`、`model`、`token`、`action`、`updated_at`
- 当前 `/api/token` 写入时没有显式 Agent/Chat 上下文，因此 `agent_id` 与 `chat_id` 为空字符串
- 日志表按 `agent_id + chat_id + 时间` 建立索引，也按 `model + 时间` 建立索引
- 如果历史数据库里仍存在旧表 `token_store_log`，启动时会自动改名为 `proxy_agent_provider_log`

## 数据表

- 主表：`token_store`
- 日志表：`proxy_agent_provider_log`

## 编译

```bash
cd /path/to/deepright/cli/module/proxy
/opt/homebrew/bin/go build -o proxy ./
```
