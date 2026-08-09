# 20260613-1 USER_GUIDE

本次迭代为 `integration` 新增了运行时 `standalone` 开关，用于把当前 `--port` 端口上的全部 HTTP 服务限制为仅本机访问。

## HTTP 接口

读取当前状态：

```bash
curl http://127.0.0.1:8080/api/standalone
```

开启本机独占模式：

```bash
curl -X POST http://127.0.0.1:8080/api/standalone=true
```

关闭本机独占模式：

```bash
curl -X POST http://127.0.0.1:8080/api/standalone=false
```

重置为启动默认值：

```bash
curl -X DELETE http://127.0.0.1:8080/api/standalone
```

返回结构示例：

```json
{
  "status": 0,
  "data": {
    "enabled": true,
    "startupEnabled": false,
    "overridden": true,
    "runtimeOnly": true
  }
}
```

## CLI 命令

```bash
./integration standalone get
./integration standalone set --value true
./integration standalone set --value false
./integration standalone reset
```

这些命令会调用本机正在运行的 `integration` 的 `/api/standalone` 接口，只修改当前进程内存，不写入 `config.json`。

## 行为说明

- `POST /api/standalone=true|false` 只接受来自 `localhost` 或 `127.0.0.1` 的请求
- 当 `standalone=true` 时，这个 `--port` 端口上的所有 HTTP 服务都只允许 `localhost`、`127.0.0.1`、`::1` 访问。
- 限制范围不仅包含 `/api/...`，也包含 `/site/...` 等静态页面，以及同端口下其他 integration/proxy 收口接口。
- 非本机请求不会收到任何额外响应，连接会被直接断开。
- `integration` 重启后，`standalone` 会恢复为默认关闭状态。
