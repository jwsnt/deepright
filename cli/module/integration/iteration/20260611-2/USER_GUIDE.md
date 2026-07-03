# 运行时 Host 覆盖

## 简介

本迭代为 `integration` 增加了运行时 Host 覆盖能力：

- HTTP 接口：`/api/host`
- CLI 命令：`integration host`

该能力只影响当前正在运行的 `integration` 进程内存，不会写入 `config.json`，也不会覆盖 `--host` 的启动参数。服务重启后，会自动恢复为启动时生效的 Host。

## HTTP 接口

### 读取当前运行时 Host

```bash
curl http://127.0.0.1:8080/api/host
```

返回示例：

```json
{
  "status": 0,
  "data": {
    "host": "https://deepright.cn",
    "startupHost": "https://deepright.cn",
    "overridden": false,
    "runtimeOnly": true
  }
}
```

### 设置当前运行时 Host

```bash
curl -X POST http://127.0.0.1:8080/api/host \
  -H 'Content-Type: application/json' \
  -d '{"host":"https://staging.deepright.cn"}'
```

### 恢复为启动 Host

```bash
curl -X DELETE http://127.0.0.1:8080/api/host
```

## CLI 用法

### 查询当前值

```bash
./integration host get
```

### 设置运行时值

```bash
./integration host set --value https://staging.deepright.cn
```

### 恢复启动值

```bash
./integration host reset
```

### 指定端口

当 `integration` 不是运行在 `8080` 时，可以显式传入端口：

```bash
./integration host get --port 9090
```

## 说明

- 仅允许本机请求修改运行时 Host。
- `cli-get` 心跳、`/v1/chat/completions` 转发、Cron 执行等链路都会实时使用当前运行时 Host。
- `runtime.json` 仍记录启动时 Host，不会因为本次运行时覆盖而被改写。
