# 20260503-2 使用手册

## 简介

本次迭代为 Proxy 模块补充了 HTTP 服务启动参数落盘能力。

## 行为说明

- 当 `proxy` 以 HTTP 服务模式启动时，会在当前启动目录生成或覆盖 `runtime.json`
- 文件内容为本次启动参数的实际取值
- 每次重新启动都会覆盖旧文件

## 示例

启动命令：

```bash
./proxy --agent-dir /agent/ --site ../site
```

写入结果：

```json
{
  "port": 9876,
  "host": "http://127.0.0.1:9998",
  "agent-dir": "/agent/",
  "device": "",
  "agent-cache": 10000,
  "site": "../site",
  "connect_timeout": 15000
}
```
