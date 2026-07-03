# browser_playwright 迭代 20260508-6 使用手册

## 变更说明

本次迭代让 `browser_playwright` 的普通 Playwright 命令也能自动接管 `browser_instance`：

- 如果没有显式传入 `--cdp`，但提供了 `--agentId + --chatId`，会先检查对应 CDP 实例是否存在
- 如果没有显式传入 `--cdp`，但提供了 `--session agent@chat`，也会按该会话拆解出的 `agentId + chatId` 先检查实例
- 检查流程固定先执行 `browser_instance get`
- 若实例不存在，再自动执行 `browser_instance create`
- `--agentId`、`--chatId`、`--session` 都会先统一转换为小写，再执行这条接管链路

这样调用方不需要再先手工执行一次 `create` 命令。

补充：

- 这里描述的是独立 `browser_playwright` 工具的行为
- 最终统一入口 `browser` 在后续收口中，已经改为直接调用内嵌的 instance 模块完成同样的自动接管
- 因此 `browser open --session agent@chat ...` 不再要求额外存在外部 `browser_instance` 二进制

## 自动实例接管

### 方式一：通过 `--agentId + --chatId`

```bash
./browser_playwright --agentId agent-a --chatId ctrip-home eval 'document.body ? document.body.innerText.slice(0, 1000) : ""'
```

执行逻辑等价于：

1. `browser_instance get --agentId agent-a --chatId ctrip-home`
2. 若不存在，则执行 `browser_instance create --agentId agent-a --chatId ctrip-home`
3. 自动补齐 `--cdp=ws://127.0.0.1:<port>/devtools/browser`
4. 自动使用 `agent-a@ctrip-home` 作为 Playwright 会话名

### 方式二：通过 `--session agent@chat`

```bash
./browser_playwright --session agent-a@ctrip-home snapshot
```

执行时会自动拆解：

- `agentId = agent-a`
- `chatId = ctrip-home`

然后继续按同样的 `get -> create` 逻辑接管实例。

## `--session` 优先级

如果同时传入了 `--session` 和 `--agentId` / `--chatId`，则以 `--session` 为准。

例如：

```bash
./browser_playwright --session agent-a@ctrip-home --agentId agent-b --chatId other eval 'location.href'
```

实际检查的是：

```bash
./browser_instance get --agentId agent-a --chatId ctrip-home
```

说明：

- `--session` 会先被拆解成真实的 `agentId + chatId`
- 即使又传入了不同的 `--agentId` / `--chatId`，也不会覆盖 `--session`
- 最终会话名仍统一落到 `agent-a@ctrip-home`
- 如果传入的是 `AGENT-A@CTRIP-HOME`，实际也会按 `agent-a@ctrip-home` 解析和匹配

## 不触发自动接管的场景

以下情况不会自动创建或检查 `browser_instance`：

- 已显式传入 `--cdp`
- 既没有传 `--agentId + --chatId`，也没有传可拆解的 `--session agent@chat`

例如：

```bash
./browser_playwright --cdp=http://127.0.0.1:9222 attach
./browser_playwright --session demo eval 'document.title'
```

这两种情况都会沿用原有 Playwright 行为，不触发实例托管。

## 推荐用法

```bash
./browser_playwright --agentId agent-a --chatId ctrip-home goto https://www.ctrip.com
./browser_playwright --session agent-a@ctrip-home snapshot
./browser_playwright --session agent-a@ctrip-home eval 'document.body ? document.body.innerText.slice(0, 1000) : ""'
```

如果更希望显式表达“先创建再 attach”，也仍然可以继续使用：

```bash
./browser_playwright create --agentId agent-a --chatId ctrip-home
```

## 验收建议

```bash
./browser_playwright --agentId agent-a --chatId ctrip-home eval 'location.href'
./browser_playwright --session agent-a@ctrip-home --agentId ignored --chatId ignored snapshot
./browser_playwright --cdp=http://127.0.0.1:9222 attach
```

重点检查：

- 未显式传 `--cdp` 时，会先尝试 `get`，不存在再自动 `create`
- `--session agent@chat` 能正确拆解为实例身份
- `--session` 与 `--agentId` / `--chatId` 冲突时，以 `--session` 为准
- 已显式传 `--cdp` 时，不触发自动实例接管

对 `browser` 统一入口的补充验收：

```bash
./browser open --session agent-a@ctrip-home https://www.ctrip.com
./browser --session agent-a@ctrip-home snapshot
```

重点检查：

- 当 `browser_instance.json` 中不存在对应实例时，会自动创建并写入状态文件
- 不需要外部单独编译或调用 `browser_instance`
- 若默认 daemon 端口残留旧进程，需先清理旧 daemon 后再复测当前产物
