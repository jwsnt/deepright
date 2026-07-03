# 20260503-2 使用手册

## 简介

本次迭代为 Integration 模块补充了两点：

- 以 HTTP 服务模式启动时，把启动参数写入启动目录下的 `runtime.json`
- `integration cron create` 与 `integration cron create-cron` 在未启动 HTTP 服务时，也会先初始化共享 sqlite，再执行统一校验逻辑

## runtime.json

- 只有启动 HTTP 服务时才会写入 `runtime.json`
- CLI 创建任务不会写这个文件
- 每次启动都会覆盖更新

示例：

```bash
./integration --agent-dir /agent/ --site ../site
```

写入结果：

```json
{
  "port": 8080,
  "host": "http://127.0.0.1:9998",
  "agent-dir": "/agent/",
  "device": "",
  "agent-cache": 120000,
  "site": "../site",
  "connect_timeout": 15000,
  "sleep": 3000,
  "thread": 3,
  "http_timeout": 60000,
  "http_connect_timeout": 15000,
  "http_socket_timeout": 45000,
  "idle_timeout": 90
}
```

## CLI 说明

- `integration cron create`
- `integration cron create-cron`

以上两条命令在未启动 HTTP 服务时，也会先初始化共享 sqlite，再复用与 HTTP 一致的 Agent 校验、模型校验和任务创建逻辑。
