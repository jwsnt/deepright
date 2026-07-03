# Browser 迭代 20260509-2 使用手册

## 变更说明

本次迭代为 `browser` 代理的全部 Playwright 能力补齐执行日志，并把自动注入 Chrome Cookie 的实际内容一并记录到插件日志中。

同时，顶层 `help` 现在会直接输出完整插件手册，满足统一 Plugin 规范下“可执行文件自带使用说明”的要求。

本目录手册为本轮迭代说明，配套的正式手册也已经同步更新：

- `../../USER_GUIDE.md`
- `../../playwright/USER_GUIDE.md`
- `../../../USER_GUIDE.md`
- `../../../../integration/USER_GUIDE.md`

## 验收结果

当前代码已经覆盖本次需求，关键点如下：

- `./browser help` 可直接输出完整插件使用手册
- 插件运行日志固定写入 `browser` 二进制同目录下的 `browser.log`
- 通过 `browser` 代理执行的 Playwright 命令会写入结构化执行日志
- Chrome Cookie 自动注入时，会把本次实际注入的 `cookies` 数组和 `cookieCount` 一并写入日志
- 对应逻辑已经同步到 `integration` 统一插件运行目录约定中

## help 命令

```bash
./browser help
```

帮助信息会覆盖以下内容：

- 插件基础命令：`name`、`param`、`command`、`help`
- 插件生命周期命令：`start`、`stop`
- Playwright 代理命令用法
- `daemon` 与 `instance` 子命令说明
- `browser.log`、`browser.pid`、`browser_instance.json` 等运行文件位置说明

## 日志文件

`browser.log` 固定写在 `browser` 可执行文件同目录。

如果 `browser` 作为 `integration` 插件运行，且通过主应用 `config/config.json` 解析到应用启动目录，则日志路径会统一收口为：

```text
<app-dir>/plugins/browser.log
```

## 日志内容

`browser.log` 采用 JSON 行格式，一行一条事件。

常见事件：

- `event=browser_instance_list`：插件级 `start/stop` 前后的实例快照
- `event=browser_plugin_daemon`：插件级 daemon 启停结果
- `event=command`：每一次 Playwright 命令执行
- `event=navigation`：导航与页面切换相关耗时
- `event=cookie`：Chrome Cookie 自动注入诊断

其中 `event=command` 至少包含：

- `stage=start|finish|error`
- `session`
- `command`
- `args`
- `flags`

常见补充字段：

- `elapsedMs` / `elapsed`：命令完成或失败耗时
- `result`：命令结果摘要，例如输出文件、页签数量、返回消息
- `error`：失败原因

其中 `event=cookie` 至少会记录：

- `action=inject|skip`
- `host`
- `targetHost`
- `cachePath`
- `cookieCount`
- `cookies`
- `cacheHit`
- `injected`

常见补充字段：

- `reason`：为什么跳过注入，例如 `host-already-injected`
- `cookieErr`：读取 Cookie 或注入 Cookie 失败时的错误信息

其中 `event=navigation` 会记录：

- `session`
- `command`
- `target`
- `gotoCostMs`
- `gotoCostExact`

## 使用示例

```bash
./browser help
./browser start
./browser --session agent-a@ctrip-home goto https://www.ctrip.com
./browser --session agent-a@ctrip-home eval 'document.title'
./browser --session agent-a@ctrip-home snapshot
./browser stop
```

执行完后可直接查看：

```bash
cat ./browser.log
```

如果希望直接从当前迭代目录回看正式手册，可以继续查看：

```bash
cat ../../USER_GUIDE.md
cat ../../playwright/USER_GUIDE.md
```

## 与 Integration 的关系

- `integration plugins start --name browser` 会复用同一套 Browser 插件生命周期逻辑
- 统一插件目录下的 `browser.log` 会同时记录插件生命周期和 Playwright 命令日志
- 这轮能力已经同步到 `integration` 手册和主手册说明中
