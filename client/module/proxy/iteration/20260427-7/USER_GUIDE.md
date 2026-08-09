# 客户端断开自动取消代理请求

## 功能说明

转发 `/v1/chat/completions` 时，使用独立的 `context.WithCancel` 管理上游连接。非主动断开（如刷新页面）不会取消上游连接，SSE 流继续直到完成。仅通过 `/api/cancel` 主动取消。

## 技术实现

`context.Background()` + `context.WithCancel()` 替代 `r.Context()`，上游连接生命周期与客户端连接解耦。
