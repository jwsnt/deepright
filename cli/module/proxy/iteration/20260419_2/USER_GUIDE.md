# Proxy 迭代 20260419_2 使用手册

## 变更说明

SSE 代理转发的 HTTP 客户端超时策略调整：

- 仅设置连接超时（`--connect_timeout`），连不上时立即报错
- 不设置读取超时和总超时，连上后等待直到成功或失败

## 新增参数

| 参数 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `--connect_timeout` | 否 | `15000` | 上游服务连接超时（毫秒） |

## 使用示例

```bash
# 默认 15 秒连接超时
./proxy --agent-dir ./agents

# 自定义 5 秒连接超时
./proxy --agent-dir ./agents --connect_timeout 5000
```

## 行为说明

- 连接阶段：在 `--connect_timeout` 指定的毫秒内无法建立 TCP 连接，立即返回 502 错误
- 读取阶段：连接建立后，无超时限制，持续等待 SSE 流直到上游返回 `[DONE]` 或连接断开

## 子模块调用

```go
client := NewProxyClient(15 * time.Second)

proxy := &ProxyServer{
    Host:           "http://127.0.0.1:9998",
    AgentDir:       "./agents",
    ConnectTimeout: 15 * time.Second,
    Client:         client,
}
```
