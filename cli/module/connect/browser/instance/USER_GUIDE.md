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
- 手册会明确说明 Windows WSL / WSL2 下，`stop` 只会关闭指定实例，不会清理其他受管 profile 目录
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
- 会先查询该实例，并执行一次 `instance shutdown` 尝试关闭旧 CDP 和旧 Chrome
- 如果实例原本不存在或已被手动关闭，仍会继续创建，不会因为 `instance not found` 中断
- 然后按与 `create` 完全一致的 `chrome`、端口、`user-data-dir` 解析规则重新创建实例
- `user-data-dir` 如果已存在则直接复用；如果不存在才按 `create` 的复制逻辑准备
- `init` 强制使用有头模式；在 Windows WSL / WSL2 下会调用 `browser_launcher.sh`，并走 `browser_instance_wsl` 的有头获取逻辑
- 创建过程会等待新的 CDP 可用后返回；不会复用旧的存活 CDP
- 后续在 `get` / `list` / `create` / `restart` 重载状态时，会自动清理已经退出的 Chrome/CDP 状态
- 非 WSL 的同步启动会使用 `--remote-debugging-address=0.0.0.0`
- 在 Windows WSL / WSL2 里，脚本会负责启动 `browser_instance_wsl`，并把其返回的 `user-data-dir` 映射到原有 CLI 的 `profileDir`
- `init` 会强制重建同一 `agentId + chatId` 的 CDP，并以有头模式启动；`create` 才会复用可用的已有 CDP
- WSL 的启动等待上限读取 integration 的 `config.json`：`browser.init_timeout`（单位：秒）；默认配置为 `300`
- 通过集成页面初始化时，新的有头 Chrome 实际启动成功后才会显示“完成”按钮；点击按钮会调用 `instance shutdown`

## stop

```bash
./browser_instance stop --agentId demo-agent --chatId chat-001
```

行为：

- 必须显式传入 `--agentId` 和 `--chatId`
- 会直接强制结束指定实例对应的 Chrome 进程
- 成功关闭后会删除该实例对应的状态记录
- 在 Windows WSL / WSL2 下，不会额外删除 `chrome_def` 或任何其他 `chrome_*` 目录
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
- Windows WSL / WSL2 下，如果有头 Chrome 已被手动关闭，仍会完成状态清理并返回 `OK`
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

`browser_instance create` / `browser_instance init` 在 Windows WSL2 下，会通过 `browser_instance_wsl` 返回真实的 `profileDir`，并统一落在 `C:\ProgramData\deepright\chrome_<随机后缀>`。其中 `init` 会先关闭旧实例、以有头模式重新启动，并读取 `config.json` 的 `browser.init_timeout`（单位：秒，默认 `300`）作为启动等待上限。

`browser_instance stop` 在 Windows WSL2 下只会关闭目标实例并清理对应状态；`chrome_def` 和其他 `chrome_*` 目录都会保留。
