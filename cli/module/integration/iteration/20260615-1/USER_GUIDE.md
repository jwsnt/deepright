# Integration 迭代手册（20260615-1）

## 本次更新

- `cli-get` 默认 HTTP 总超时从 `60000ms` 调整为 `45000ms`
- 主应用 `config/config.json` 新增 `http` 配置块，可集中配置：
  - `http_connect_timeout`
  - `http_socket_timeout`
  - `http_timeout`
  - `debug`
- 所有原先写入/读取 `runtime.json` 的运行态配置，统一收口到主应用 `config/config.json` 的同名字段
- 以上 HTTP 配置只从主应用 `config/config.json.http` 读取，不再兼容旧的平铺写法，也不再从其他文件回退
- `http.debug=true` 时，会把 `cli/get` / `cli/pub` 明细日志写入 `integration` 标准日志

## 配置示例

```json
{
  "http": {
    "http_connect_timeout": 15000,
    "http_socket_timeout": 45000,
    "http_timeout": 45000,
    "debug": true
  }
}
```

## 明细日志口径

- `cli/get` 请求远程主机超时时间
  - 同时附带本次耗时和当前 HTTP 超时配置，便于区分连接超时、响应头等待超时和总超时
- `cli/get` 返回待执行任务时的原始报文、时间
- `cli/pub` 回传执行结果时的状态、结果、时间

## 同步结果

- `integration/main.go` 已支持从 `config/config.json.http` 读取 `cli-get` HTTP 配置
- `integration/main.go` 已支持 `http.debug` 详细日志
- `integration/main_test.go` 已补充嵌套 `http` 配置和详细日志测试
