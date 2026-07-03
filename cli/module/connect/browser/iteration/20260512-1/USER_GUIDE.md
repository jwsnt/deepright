# browser 迭代 20260512-1 使用手册

## 本次完成内容

- `browser help` 现在会输出统一插件手册，并补充 Obscura 运行时收口说明
- `browser help` 同时内嵌代理后的 Playwright 详细命令与 usage，避免再切换到独立二进制查询
- `browser` 默认日志文件固定为插件同目录下的 `browser.log`
- 后台 Playwright daemon 以脱离前台命令生命周期的方式启动，避免父进程 stdout/stderr 管道失效后自灭
- daemon 就绪检查不只看端口监听，还会通过健康接口确认归属，避免误连旧版本后台进程

## 常用命令

```bash
./browser help
./browser daemon help
./browser instance help
./browser create --agentId agent-a --chatId chat-001
./browser --session agent-a@chat-001 goto https://example.com
./browser instance create --agentId agent-a --chatId chat-001
./browser instance restart --agentId agent-a --chatId chat-001
./browser instance shutdown --agentId agent-a --chatId chat-001
./browser instance destroy --agentId agent-a --chatId chat-001
```

## Help 收口说明

- `browser help` 会统一展示插件生命周期说明
- `browser help` 会额外说明 Obscura 运行时路径 `./obscura/release/<platform>/obscura`
- `browser help` 会内嵌代理后的 Playwright 命令手册，包括常见命令、daemon 子命令和共享 flags
- `browser instance help` 用于查看底层 CDP 实例命令

## 日志与后台进程

- `browser.log` 默认位于 `browser` 可执行文件同目录
- `browser daemon start` 启动的后台进程会脱离前台会话，不继承临时 stdout/stderr 管道
- `browser stop` 会同步等待 daemon 退出，并完成 runtime 级资源清理

## 验收重点

- `browser help` 中应同时看到 Obscura 收口说明和 Playwright 详细手册
- `browser.log` 会在插件目录生成
- daemon 启动后父命令退出，后台进程仍保持存活
- daemon 就绪检查基于健康接口，不只依赖端口是否被监听

## 对应需求

- [/path/to/deepright/cli/module/connect/browser/iteration/20260512-1/REQUIREMENT.md](REQUIREMENT.md)
