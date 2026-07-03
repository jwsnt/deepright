本次迭代在 `browser_playwright` 的默认 Chrome UA 覆盖基础上，继续为受管 CDP attach 会话自动补齐页面内 `navigator.platform`。

关键行为：

- 默认 UA 仍固定为 `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36`
- `attach --cdp=...`、`create` 以及受管 `Agent@Chat` 会话，会继续通过 `SetExtraHTTPHeaders({"User-Agent": ...})` 覆盖请求头
- 同时继续通过 `AddInitScript` 与 `page.Evaluate` 覆盖页面内属性
- 新增默认覆盖 `navigator.platform === "MacIntel"`
- 不使用 `NewCDPSession(page) + Emulation.setUserAgentOverride`，避免 Obscura attach 后再次触发 driver 崩溃

验收建议：

```bash
./browser_playwright --session agent-a@chat-001 attach --cdp=chrome
./browser_playwright --session agent-a@chat-001 eval 'navigator.userAgent'
./browser_playwright --session agent-a@chat-001 eval 'navigator.platform'
```

预期结果：

- `navigator.userAgent` 返回默认 Chrome UA
- `navigator.platform` 返回 `MacIntel`
- attach 后会话保持稳定，不出现崩溃或 `EOF`
