# browser 迭代 20260517-1 使用手册

## 本次完成内容

- `browser` 新增 `scope` 命令
- `browser scope` 稳定返回空数组 `[]`
- `browser command` 已补充 `scope`
- `scope` 由 `browser` 插件包装层直接处理，不会再下钻到 Playwright daemon

## 命令说明

```bash
./browser scope
```

返回：

```json
[]
```

说明：

- 返回空数组表示 Browser 插件不开放容器侧的 `reuse`、`agent`、`provider`、`thinking` 四项通用配置
- `scope` 不会触发 `healthz`、`/command`、daemon 自动拉起或其他运行时副作用

```bash
./browser command
```

返回值会包含：

```json
["command","daemon","help","instance","name","param","scope","start","stop","..."]
```

说明：

- `browser` 既然实现了 `scope`，就必须在 `command` 中显式声明 `scope`

## 验收重点

- 执行 `./browser scope` 返回 `[]`
- 执行 `./browser command` 返回值中包含 `scope`
- 执行 `./browser scope` 时不会代理到 Playwright daemon
- 插件日志仍固定写在同目录 `browser.log`

## 对应需求

- [/path/to/deepright/cli/module/connect/browser/iteration/20260517-1/REQUIREMENT.md](REQUIREMENT.md)
