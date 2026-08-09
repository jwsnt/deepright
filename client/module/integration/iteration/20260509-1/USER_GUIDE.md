# Integration 迭代 20260509-1 使用手册

## 变更说明

本次迭代为 `integration` 补充了 Agent 元数据中的 `plugins` 字段，并统一覆盖两条请求链路：

- 转发 `/v1/chat/completions` 时注入的 `metadata`
- `cli/get` 与 `cli/pub` 提交的 Agent 元数据

## `plugins` 字段规则

- `plugins` 位于 Agent 元数据顶层，与 `timezone`、`deviceId`、`terminal`、`git`、`gateway`、`sys`、`app`、`agents` 同级
- 字段值为插件 `key` 数组，不是展示名
- 只有同时满足“已配置且已启动”的插件才会写入
- “已配置”来自当前 `integration` 二进制 `meta-list` 返回的插件配置视图
- “已启动”默认通过插件运行目录下的 `<plugin-key>.pid` 判断；为兼容旧版 `browser` 运行态，`integration` 仍额外兼容 `.browser_playwright/browser_playwright.pid`，只有这些运行态 PID 文件存在且对应进程仍存活时，才会保留该插件
- 返回结果会按插件 `key` 排序
- 若没有任何同时满足“已配置且已启动”的插件，则不会写入 `plugins`

## 行为示例

假设当前存在如下状态：

- `./integration connect meta-list` 返回 `browser`
- `./plugins/browser.pid` 与 `./plugins/.browser_playwright/browser_playwright.pid` 都不存在

则：

- `browser` 仍然属于“已配置插件”
- 但不会出现在转发请求或 `cli/get` / `cli/pub` 的 `metadata.plugins` 中

只有在 `browser.pid` 或 `.browser_playwright/browser_playwright.pid` 存在且对应进程仍存活时，`plugins` 才会包含 `browser`

## 补充说明

- `meta-list` 仍是“已配置插件”视图，不等于“已启动插件”列表
- 更新 `module/release/integration` 二进制后，需要重启当前 release 目录下的 integration 进程，新规则才会生效
