# Integration：特殊 SSE 页面打开

在主运行配置 `config/config.json` 中设置：

```json
{
  "page": {
    "new_tab": 931,
    "iframe": 932
  }
}
```

`GET /api/runtime_config` 会只读返回完整 `page` 对象，供 Site 将 `931` 识别为“打开系统浏览器”、将 `932` 识别为“页内 iframe”的 SSE 业务码。

当 Site 收到 `page.new_tab` 包后，会请求 `POST /api/browser/open`；Integration 仅接受本机请求和带 host 的 HTTP(S) URL，并交给操作系统默认浏览器处理。接口不会写入配置；非法 URL、远程请求或浏览器启动失败都会返回错误。`page.iframe` 仅由 Site 在页面内展示，不调用该接口。

模型配置测试也使用相同的页面业务码。Integration 只会把内容确实符合 `{"url":"...","message":"..."}` 的匹配特殊包以最小字段转给 Site，普通测试 SSE 分包不会暴露给浏览器。Site 因而可在结果区只显示 `message`，并执行相同的浏览器或页内 iframe 打开行为。

`cli/get` 心跳若返回的 HTTP 状态码或 JSON 业务 `code` 等于配置的 `new_tab` 或 `iframe`，Integration 将其视为一次成功但没有任务的轮询；它不会进入 heartbeat 失败计数，也不会导致 Site 出现远程服务报警。

对应需求见 [REQUIREMENT.md](REQUIREMENT.md)。
