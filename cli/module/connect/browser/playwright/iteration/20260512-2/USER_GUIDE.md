# browser_playwright 迭代 20260512-2 使用手册

## 变更说明

本次迭代为 `browser_playwright` 的 Playwright 代理链路补齐了更完整的浏览器指纹，并保持 `attach/create` 场景稳定可用。

当前默认行为：

- 默认 Chrome UA 仍固定为 `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36`
- `navigator.userAgentData` 会返回 macOS / Chrome 指纹信息
- `navigator.platform` 固定为 `MacIntel`
- `navigator.maxTouchPoints` 固定为 `5`
- `window.screen` 固定为 `2560x1440`
- WebGL `UNMASKED_VENDOR_WEBGL` / `UNMASKED_RENDERER_WEBGL` 默认返回 Apple GPU 指纹
- 时区与语言改为跟随当前系统；在 macOS 上优先读取系统偏好设置，其余环境回退到运行环境变量与系统时区链接

## 导航与 Cookie

- `open`、`goto`、`attach/create` 后首个目标页仍会先按目标域名自动注入本机 Chrome Cookie
- 首跳导航完成条件固定为 `domcontentloaded`
- 这样可以减少动态站点第一次跳转卡在完整 `load` 的问题

## 受影响命令

- `attach --cdp=chrome|URL`
- `create --agentId ... --chatId ...`
- 普通 `open`
- 普通 `goto`
- 通过受管 `Agent@Chat` 会话执行的 `eval`、`snapshot`、`requests` 等命令

## 推荐验收

```bash
./browser_playwright create --agentId agent-a --chatId chat-001
./browser_playwright --session agent-a@chat-001 eval 'navigator.userAgent'
./browser_playwright --session agent-a@chat-001 eval 'navigator.userAgentData'
./browser_playwright --session agent-a@chat-001 eval 'navigator.maxTouchPoints'
./browser_playwright --session agent-a@chat-001 eval '[screen.width, screen.height]'
./browser_playwright --session agent-a@chat-001 eval 'Intl.DateTimeFormat().resolvedOptions().timeZone'
./browser_playwright --session agent-a@chat-001 goto https://example.com
```

如需检查 WebGL，可执行：

```bash
./browser_playwright --session agent-a@chat-001 eval '(() => { const canvas=document.createElement("canvas"); const gl=canvas.getContext("webgl") || canvas.getContext("experimental-webgl"); const dbg=gl && gl.getExtension("WEBGL_debug_renderer_info"); return { vendor: gl && dbg ? gl.getParameter(dbg.UNMASKED_VENDOR_WEBGL) : null, renderer: gl && dbg ? gl.getParameter(dbg.UNMASKED_RENDERER_WEBGL) : null }; })()'
```

## 预期结果

- `create` 成功返回 session
- attach 后 session 不崩溃
- `navigator.userAgentData` 可用
- `navigator.maxTouchPoints` 返回 `5`
- `screen.width/height` 返回 `2560 / 1440`
- WebGL vendor / renderer 返回配置的 Apple 指纹
- 页面内语言与时区和当前系统一致
- 首跳导航不会因等待完整 `load` 而更容易卡超时
