# browser_playwright 迭代 20260511-1 使用手册

## 变更说明

本次迭代修复受管 `browser_instance` 场景下的 `create/attach` 崩溃问题。

旧行为里：

- `browser_instance create/get` 实际已经成功
- `browser_playwright attach` 后还会对每个 page 执行 `NewCDPSession(page)`
- 再发送 `Emulation.setUserAgentOverride`
- 在 Obscura 提供的 CDP 场景下，这条路径可能触发 Playwright driver 崩溃
- 最终对外表现为 `target closed: could not read protocol padding: EOF`

现在的行为：

- 保留原有 `browser_instance` 创建/复用逻辑
- 保留默认 Chrome UA 语义
- 不再使用 `NewCDPSession(page) + Emulation.setUserAgentOverride` 做逐页 UA 覆盖
- 改为使用 `SetExtraHTTPHeaders({"User-Agent": ...})`
- 同时通过 `AddInitScript` / `page.Evaluate` 覆盖页面内 `navigator.userAgent`

## 受影响命令

以下命令会直接受益于这次修复：

- `create --agentId ... --chatId ...`
- `attach --cdp=chrome|URL`
- 未显式传 `--cdp`、但通过 `--agentId + --chatId` 或 `--session agent@chat` 自动接管受管实例的普通命令

示例：

```bash
./browser_playwright create --agentId agent-a --chatId chat-001
./browser_playwright --session agent-a@chat-001 eval 'navigator.userAgent'
./browser_playwright --session agent-a@chat-001 goto https://example.com
```

## 推荐验收

```bash
./browser_playwright create --agentId agent-a --chatId chat-001
./browser_playwright --session agent-a@chat-001 eval 'navigator.userAgent'
./browser_playwright --session agent-a@chat-001 snapshot
```

重点检查：

- `create` 成功返回 session
- attach 后 session 不崩溃
- `eval 'navigator.userAgent'` 返回预期 Chrome UA
- 不再出现 `target closed: could not read protocol padding: EOF`
