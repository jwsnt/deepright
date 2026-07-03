# 20260503-5 User Guide

## 目标

本次迭代为 `integration` 的 HTTP 服务启动流程补充了更友好的默认行为：

- `--agent-dir` 默认使用当前应用目录下的 `./agent`
- 如果默认或显式指定的 Agent 目录不存在，启动时会自动创建
- `--site` 默认使用当前应用目录下的 `./site`
- 服务启动成功后会自动打开浏览器进入前端页面

## 启动示例

沿用默认目录启动：

```bash
./integration
```

按本次需求显式指定目录：

```bash
./integration --agent-dir agent --site site --host http://xxx.com
```

指定端口启动：

```bash
./integration --port 9090 --host http://127.0.0.1:9998
```

## 默认行为

- `--agent-dir` 未传时，等价于当前目录下的 `./agent`
- `--site` 未传时，等价于当前目录下的 `./site`
- `agent` 目录不存在时会自动创建
- `site` 目录只会解析为绝对路径，不会自动创建
- 启动参数写入 `runtime.json` 时，保存的是解析后的实际路径

## 自动打开浏览器

- 服务启动成功后会自动打开系统默认浏览器
- 默认访问地址为：

```text
http://127.0.0.1:8080/site/#app
```

- 如果通过 `--port` 指定了其他端口，则浏览器地址会变为：

```text
http://127.0.0.1:<port>/site/#app
```

## 说明

- 这次迭代只补充启动默认值和浏览器自动打开逻辑，不影响现有 proxy、cli-get、cron、static 的接口与行为
- 更完整的启动参数、HTTP API 和 CLI 子命令说明请查看上级手册 `../../USER_GUIDE.md`
