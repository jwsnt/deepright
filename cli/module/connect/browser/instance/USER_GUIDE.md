# browser_instance 使用手册

## 简介

`browser_instance` 负责按 `AgentId + ChatId` 维度创建、复用、查询、重启和关闭本机 Chrome CDP 实例，并把实例状态写入同目录下的 `browser_instance.json`。

在 Windows WSL / WSL2 下，`create` 和 `init` 默认调用插件同目录的 `browser_launcher.sh`。

## 命令总览

```bash
./browser_instance help
./browser_instance shutdown --agentId AGENT --chatId CHAT
./browser_instance create --agentId AGENT --chatId CHAT
./browser_instance init --agentId AGENT --chatId CHAT
./browser_instance restart --agentId AGENT --chatId CHAT
./browser_instance stop --agentId AGENT --chatId CHAT
./browser_instance shutdown
./browser_instance list
./browser_instance get --agentId AGENT --chatId CHAT
```

## help

```bash
./browser_instance help
```

行为：

- `help` 会输出 `create / init / restart / stop / shutdown / list / get` 的完整使用手册
- 手册会明确说明 Windows WSL / WSL2 下，`create` / `init` 的受管实例目录固定落在 `C:\ProgramData\deepright\chrome_<随机后缀>`
- 手册会明确说明 Windows WSL / WSL2 下，`stop` 在原流程结束后会继续并发清理 `C:\ProgramData\deepright` 下全部 `chrome*` 目录
- `help` 只负责说明命令行为，不会修改实例状态

## create

```bash
./browser_instance create --agentId demo-agent --chatId chat-001
./browser_instance create --agentId demo-agent --chatId chat-001 --headless none
./browser_instance create --agentId demo-agent --chatId chat-001 --chrome /Applications/Google\\ Chrome.app/Contents/MacOS/Google\\ Chrome
```

默认行为：

- `--agentId`、`--chatId` 会先统一转成小写
- 端口由 `agentId + chatId` 的稳定哈希得到，固定落在 `20000+`
- 如果该身份已存在健康 CDP 实例，则直接复用
- 会优先读取 Browser 插件配置里的 `meta.chrome` 和 `meta.headless`
- `meta.chrome` 不可用时，退回命令行 `--chrome` 和系统自动探测
- Windows WSL / WSL2 下，`meta.chrome` 不可用时优先回退到 `/mnt/c/Program Files/Google/Chrome/Application/chrome.exe`
- `meta.headless=false/FALSE/False` 时以有头模式启动
- `meta.headless=true/TRUE/True` 时以无头模式启动
- `meta.headless` 为空时退回命令行 `--headless`
- `meta.headless` 解析失败时默认按 `--headless=new`
- Windows WSL / WSL2 下，会调用插件同目录的 `browser_launcher.sh`，由其转调内置的 `browser_instance_wsl` 获取/复用实例
- 新建实例前，会准备实例专属的受管目录作为 `--user-data-dir`
- 在 macOS / Linux / Windows 原生环境里，如果 `chrome_${port}` 不存在，会基于当前系统 Chrome 的 `User Data` 根目录复制一份精简副本；已存在则直接复用
- 复制时会过滤 `CacheStorage`、`OptGuideOnDeviceModel` 和其他易失缓存目录，同时保留 `Default/WebStorage`、`Default/IndexedDB`、`Default/Local Storage`
- 在 Windows WSL / WSL2 里，真实 `--user-data-dir` 由 `browser_instance_wsl` 决定并返回；CLI 仍通过原有 `profileDir` 字段对外返回
- WSL2 下新的实例目录固定落在 `C:\ProgramData\deepright\chrome_<随机后缀>`
- 如果这个目录首次创建，会优先尝试从 `C:\ProgramData\deepright\chrome_def` 复制一份精简副本
- 如果 `chrome_def` 不存在，或复制途中任意文件失败，则只记录日志并回退为空目录，不阻断本次实例创建
- 成功后会更新 `browser_instance.json`
- 后续在 `get` / `list` / `create` / `restart` 重载状态时，会按 `--browser_expired` 清理失效或超时实例

## init

```bash
./browser_instance init --agentId demo-agent --chatId chat-001
```

行为：

- 必须显式传入 `--agentId` 和 `--chatId`
- 会先执行一次 `instance get`
- 只有在 `instance get` 确认该 CDP 已存在且可用时，才会继续执行一次 `destroy`
- 如果实例原本不存在，会直接继续创建，不因为 `instance not found` 中断
- 然后按与 `create` 完全一致的 `chrome`、端口、`user-data-dir` 解析规则重新创建实例
- `user-data-dir` 如果已存在则直接复用；如果不存在才按 `create` 的复制逻辑准备
- `init` 强制使用有头模式；在 Windows WSL / WSL2 下会调用 `browser_launcher.sh`，并走 `browser_instance_wsl` 的有头获取逻辑
- 创建过程会先等新的 CDP 可用；在 macOS 和 Windows WSL / WSL2 下会继续阻塞到该 Chrome/CDP 退出，在 Linux 原生环境下则会在 ready 后立即返回
- 后续在 `get` / `list` / `create` / `restart` 重载状态时，会自动清理已经退出的 Chrome/CDP 状态
- 非 WSL 的同步启动会使用 `--remote-debugging-address=0.0.0.0`
- 在 Windows WSL / WSL2 里，脚本会负责启动 `browser_instance_wsl`，并把其返回的 `user-data-dir` 映射到原有 CLI 的 `profileDir`

## stop

```bash
./browser_instance stop --agentId demo-agent --chatId chat-001
```

行为：

- 必须显式传入 `--agentId` 和 `--chatId`
- 会直接强制结束指定实例对应的 Chrome 进程
- 成功关闭后会删除该实例对应的状态记录
- 在 Windows WSL / WSL2 下，会在原有 stop 流程结束后，并发删除 `C:\ProgramData\deepright` 下全部 `chrome*` 目录，包括 `chrome_def`
- 每个目录的删除成功和失败都会写入 `browser.log`
- 任意目录删除失败都不会阻断命令返回
- 成功时输出 `OK`

## shutdown

```bash
./browser_instance shutdown --agentId demo-agent --chatId chat-001
./browser_instance shutdown
```

行为：

- 会直接强制结束对应 Chrome 进程
- 在 Windows WSL / WSL2 下，会额外在 Windows 宿主机侧执行 `Stop-Process -Id <pid> -Force` 并确认端口释放
- 关闭失败会写 `browser.log`，同时命令直接返回错误
- 在 Windows WSL / WSL2 下，`shutdown` 不会删除 `chrome_*` 目录
- 如果删除后 `browser_instance.json` 已经没有任何实例记录，则会一并删除这个文件
- 不传 `--agentId` / `--chatId` 时，会兼容清理旧版本遗留的固定端口 CDP；如果它已经不存在，则默认成功
- 如需顺带删除实例目录，可配合内部清理标记使用

## restart

```bash
./browser_instance restart --agentId demo-agent --chatId chat-001
```

行为：

- 先执行一次 `stop`
- 再用同一组 `AgentId + ChatId` 重新 `create`
- 端口保持稳定，正常情况下会回到同一个端口

## list

```bash
./browser_instance list
```

行为：

- 自动清理已死亡进程
- 自动清理已不再健康的 CDP 记录
- 自动清理空闲超时实例
- 空闲释放时会直接结束对应 Chrome 进程
- 返回当前仍然存活的实例数组

## get

```bash
./browser_instance get --agentId demo-agent --chatId chat-001
```

行为：

- 先做一次与 `list` 相同的健康清理
- 找到实例后刷新 `lastActiveAt`
- 返回指定实例的最新 `agentId/chatId/port/pid/cdp`

## WSL2 说明

`browser_instance create` / `browser_instance init` 在 Windows WSL2 下，会通过 `browser_instance_wsl` 返回真实的 `profileDir`，并统一落在 `C:\ProgramData\deepright\chrome_<随机后缀>`。

`browser_instance stop` 在 Windows WSL2 下，会先按原有流程关闭目标实例并清理状态；然后并发删除 `C:\ProgramData\deepright` 下全部 `chrome*` 目录，并把每个删除成功或失败都写入 `browser.log`，但不会因此让 stop 失败。
