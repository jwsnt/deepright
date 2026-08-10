# Browser 迭代 20260509-1 使用手册

## 变更说明

本次迭代为 `browser` 补齐统一 Plugin 规范下的生命周期行为。

除了原有的 Browser Instance 清理能力外，顶层 `start` / `stop` 现在还会真正托管后台 Playwright daemon 进程，使插件启动状态可以通过 `browser.pid` 被统一识别。

## 插件元信息

```bash
./browser name
```

返回：

```json
{"key":"browser","name":"浏览器"}
```

```bash
./browser param
```

返回：

```json
["headless","chrome"]
```

```bash
./browser command
```

返回：

```json
["command","daemon","help","instance","name","param","start","stop","goto", "..."]
```

说明：

- `browser` 已补齐 Plugin 规范要求的 `name`、`param`、`command`、`help`、`start`、`stop`
- 可执行文件名固定为 `browser`，并与 `name` 返回的 `key` 保持一致

## 生命周期行为

### start

```bash
./browser start
```

行为：

- 先读取 `browser_instance.json`
- 列出并关闭其中全部受管 CDP 实例，相当于插件级重启前清理
- 启动同目录下 `browser.pid` 对应的后台 Playwright daemon
- 在 `browser.log` 中记录实例清理前后快照，以及 daemon 启动结果
- 如果传入 `--connect-bin` 且对应主应用 `config/config.json` 可解析到应用启动目录，则这里的“同目录”会强制收口到 `integration` 应用启动目录下的 `plugins/`

### stop

```bash
./browser stop
```

行为：

- 先读取 `browser_instance.json`
- 列出并关闭其中全部受管 CDP 实例，相当于插件级清理
- 关闭同目录下 `browser.pid` 对应的后台 Playwright daemon
- 在 `browser.log` 中记录实例清理前后快照，以及 daemon 关闭结果
- 如果传入 `--connect-bin` 且对应主应用 `config/config.json` 可解析到应用启动目录，则这里的“同目录”会强制收口到 `integration` 应用启动目录下的 `plugins/`
- 受管实例清理会实际结束对应的 `obscura` 进程；在统一插件生命周期场景下，`integration stop` / `integration plugins stop --name browser` 会补齐 `--connect-bin`，确保 Browser 能按正确运行目录一并回收残留 monitor 子进程

## 默认文件位置

如果没有显式传参，插件生命周期默认使用以下路径，且都与 `browser` 二进制同级：

- `browser.pid`
- `browser.log`
- `.browser_playwright/`
- `browser_instance.json`

当 `--connect-bin` 指向 `integration` 且可从主应用 `config/config.json` 解析出 `app-dir` 时，Browser 会统一改为使用：

- `app-dir/plugins/browser.pid`
- `app-dir/plugins/browser.log`
- `app-dir/plugins/.browser_playwright/`
- `app-dir/plugins/browser_instance.json`

插件交付目录示意：

```text
plugins/
├── browser
├── browser.log
├── browser.pid
├── browser_instance.json
├── obscura/
│   └── release/
└── playwright/
    └── driver/
```

## 与 Integration 的关系

- `integration plugins start --name browser` 实际调用的就是 `browser start`
- `integration plugins stop --name browser` 实际调用的就是 `browser stop`
- `integration` 判断 Browser 插件是否已启动时，依赖的是 `plugins/browser.pid`
- 通过 `integration` 触发时，运行期日志、PID、daemon 状态目录和实例状态文件都会统一写在 `integration` 应用启动目录下的 `plugins/`

## 补充说明

- `browser daemon start|stop|serve` 仍然保留，适合独立调试底层 Playwright daemon
- 统一插件生命周期场景下，优先使用顶层 `browser start|stop`
- 顶层 `./browser help` 会输出完整插件手册，供统一插件管理场景直接查看
- 顶层受管参数 `--agentId`、`--chatId`、`--session` 在运行时会先统一转换为小写，避免生命周期重启后因大小写不同造成实例或会话匹配偏差
