# 服务地址管理

## 简介

`integration` 提供服务地址读取和修改能力：

- HTTP 接口：`/api/host`
- CLI 命令：`integration host`

修改会立即生效，并写入应用资源目录中的 `config/config.json.host`。重启后会继续使用已保存的服务地址；恢复默认值会保存 `https://www.deepright.cn`。

## HTTP 接口

### 读取当前服务地址

```bash
curl http://127.0.0.1:8080/api/host
```

返回示例：

```json
{
  "status": 0,
  "data": {
    "host": "https://deepright.cn"
  }
}
```

### 设置服务地址

```bash
curl -X POST http://127.0.0.1:8080/api/host \
  -H 'Content-Type: application/json' \
  -d '{"host":"https://staging.deepright.cn"}'
```

### 恢复默认服务地址

```bash
curl -X DELETE http://127.0.0.1:8080/api/host
```

## CLI 用法

### 查询当前值

```bash
./integration host get
```

### 设置服务地址

```bash
./integration host set --value https://staging.deepright.cn
```

### 恢复默认值

```bash
./integration host reset
```

### 指定端口

当 `integration` 不是运行在 `8080` 时，可以显式传入端口：

```bash
./integration host get --port 9090
```

## 说明

- 仅允许本机请求修改服务地址。
- `cli-get` 心跳、`/v1/chat/completions` 转发、Cron 执行等链路都会立即使用新地址。
- 配置写入失败时，接口不会切换当前地址，并会返回可展示的错误。
