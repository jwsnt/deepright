# browser_playwright 迭代 20260508-4 使用手册

## 变更说明

本次迭代补充了两项行为：

- 当未显式传入 `--session`，但提供了 `--agentId` 和 `--chatId` 时，会自动使用 `agentId@chatId` 作为会话名
- 首跳导航会先按目标域名注入 Chrome Cookie，再使用 `domcontentloaded` 作为完成条件，降低动态站点第一次 `goto` 卡超时的概率

## 自动 Session

### 未指定 `--session`

```bash
./browser_playwright --agentId agent-a --chatId ctrip-home eval 'document.body ? document.body.innerText.slice(0, 1000) : ""'
```

等价于：

```bash
./browser_playwright --session agent-a@ctrip-home eval 'document.body ? document.body.innerText.slice(0, 1000) : ""'
```

### 已指定 `--session`

```bash
./browser_playwright --session agent-a@ctrip-home --agentId agent-b --chatId ctrip-home eval 'document.body ? document.body.innerText.slice(0, 1000) : ""'
```

说明：

- 显式传入的 `--session` 优先级最高
- 只有在 `--session` 缺失时，才会根据 `agentId + chatId` 自动推导
- 若三者都未提供，仍回退为默认会话 `default`
- `--agentId`、`--chatId`、`--session` 会先统一转换为小写再参与推导和匹配

## 首跳 Cookie 注入

- `open`、`goto`、`tab-new`、`attach/create` 后首次进入页面时，会先尝试按目标域名注入当前系统 Chrome Cookie
- 首跳导航完成条件固定为 `domcontentloaded`
- 该行为对动态站点首跳更稳定，尤其适合需要登录态的网站

## 兼容性说明

- 现有显式 `--session` 用法保持不变
- `create --agentId --chatId` 原有的 `agentId@chatId` 会话规则保持不变
- 相关行为可直接被统一入口 `browser` 复用
