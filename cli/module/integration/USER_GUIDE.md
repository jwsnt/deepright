# Integration 使用手册

## 简介

Integration 将 cli-get、proxy、cron、static 四个模块整合为一个完整的 HTTP 服务，统一端口、共享参数，并共享同一份 SQLite 数据。

## 安装

```bash
cd /path/to/deepright/cli/module/integration
/opt/homebrew/bin/go build -o integration ./
```

已验证可编译产物：当前目录下已成功生成 `integration` 可执行文件。

## 使用方法

```bash
./integration [选项...]
```

也支持显式生命周期子命令：

```bash
./integration serve [选项...]
./integration start [选项...]
./integration stop [--pid-file PATH]
./integration restart [选项...]
./integration plugins exec --key KEY --command 'SUBCOMMAND [ARGS...]' [plugin flags...]
./integration knowledge update-time --timestamp 时间戳 --agentId Agent名称
./integration knowledge last-update [--agentId Agent名称]
./integration standalone get
./integration standalone set --value true|false
./integration standalone reset
./integration agent export --agent AgentId [--output ./AgentId.zip]
./integration agent import --input ./AgentId.zip
./integration splash [--logo PATH] [--duration 5s] [--async]
./integration notify [--title TEXT] [--message TEXT]
./integration skills-warning [--refresh]
./integration file-last-update --file 路径 [--agent AgentId]
./integration backup-clean [--agent-dir 路径] [--archive-after 24h] [--delete-after 72h]
```

如果你沿用本次迭代的默认目录约定，也可以显式写成：

```bash
./integration --agent-dir agent --default-dir config --site site --host http://xxx.com
```

也支持在应用目录下的 `config/config.json` 作为启动配置。也就是与 `integration` 同目录的 `config` 文件夹内固定读取：

```json
{
  "host": "http://www.dr.cn",
  "port": 9090,
  "agentDir": "agent",
  "default_dir": "config",
  "site": "site",
  "queue": 1000,
  "retry_interval": 10000,
  "retry_times": 1,
  "http": {
    "http_connect_timeout": 15000,
    "http_socket_timeout": 45000,
    "http_timeout": 45000,
    "debug": false
  }
}
```

优先级固定为：命令行 `--参数` > `config/config.json` > 内置默认值。
也就是说：

- 执行 `./integration --host http://www.deepright.cn` 时，最终 `host` 仍然是 `http://www.deepright.cn`
- 执行 `./integration` 且 `config/config.json` 里写了 `"host": "http://www.dr.cn"` 时，最终 `host` 为 `http://www.dr.cn`
- 未在命令行和 `config/config.json` 中提供的参数，才会回退到手册中的默认值

`config.json` 键名支持和参数名对应的多种常见写法，例如 `agent-dir`、`agent_dir`、`agentDir` 都会识别为 `--agent-dir`；`pid-file`、`log-file`、`port` 等启动参数也同理。`cli-get` 的 HTTP 相关配置只从 `config/config.json.http` 读取，不再读取旧的平铺 `http_timeout` / `http_connect_timeout` / `http_socket_timeout` / `http_debug` 写法，也不从其他运行态文件回退。

`config/config.json` 只作为主应用启动配置使用；同目录下其余模板文件或目录（如 `SOUL.md`、`USER.md`、`skills/`）才会在创建新 Agent 或补齐 `DEF_AGENT` 时复制到 Agent 工作目录。新建出来的 Agent `config.json` 会固定初始化为空对象 `{}`，不会继承主应用配置内容。

### 参数说明

| 参数 | 必填 | 默认值 | 说明 | 模块 |
|------|------|--------|------|------|
| `--agent-dir` | 否 | macOS: `~/Library/Application Support/deepright/agent`；WSL: `~/deepright/agent`；其他系统: `./agent` | Agent 根目录路径；目录不存在时自动创建；若目录为空则自动补齐 `DEF_AGENT`，并确保 `DEF_AGENT/skills` 存在 | 共享 |
| `--default-dir` | 否 | `./config` | 新建 Agent 和空 `agent-dir` 启动补齐 `DEF_AGENT` 时使用的默认模板目录；会复制其中的模板文件，并为 Agent 单独初始化空 `config.json` | 共享 |
| `--port` | 否 | `8080` | HTTP 服务端口 | proxy + static + cron API |
| `--host` | 否 | `https://www.deepright.cn` | 上游服务地址 | proxy + cli-get + cron执行 |
| `--device` | 否 | 自动生成 | 设备ID | 共享 |
| `--agent-cache` | 否 | `120000` | Agent 元数据缓存 TTL（毫秒） | 共享 |
| `--site` | 否 | `./site` | 静态站点目录；默认取当前应用目录下的 `site` | static |
| `--pid-file` | 否 | 默认当前应用目录下的 `integration.pid`；WSL 下为 `~/deepright/integration.pid` | 后台启动时使用的 PID 文件路径 | integration 生命周期 |
| `--log-file` | 否 | 默认当前应用目录下的 `integration.log`；WSL 下为 `~/deepright/integration.log` | 后台启动时写入的日志文件路径 | integration 生命周期 |
| `--connect_timeout` | 否 | `15000` | 上游服务连接超时（毫秒） | proxy |
| `--knowledge_update_interval` | 否 | `7200000` | knowledge `lastUpdate` 透传阈值（毫秒） | proxy |
| `--knowledge_update_lock` | 否 | `1800000` | knowledge 更新申请锁窗口（毫秒） | proxy |
| `--install_app` | 否 | 空 | 额外待安装应用，逗号分隔，会合并到 `/install_app` 返回中 | proxy |
| `--reply` | 否 | `<开始执行>可通过新消息更新任务` | 三方插件开始执行时的推送文案 | proxy |
| `--sleep` | 否 | `3000` | cli-get 心跳请求失败或非 200 时的重试等待时间（毫秒） | cli-get |
| `--thread` | 否 | `20` | 执行 Worker 数量 | cli-get |
| `--queue` | 否 | `1000` | cli-get 本地任务队列容量；队列满时暂停发 `/cli/get` | cli-get |
| `--retry_interval` | 否 | `10000` | cli-get `/cli/pub` 失败后的重试等待时间（毫秒） | cli-get |
| `--retry_times` | 否 | `1` | cli-get `/cli/pub` 首次失败后允许额外重试的次数 | cli-get |
| `--http_timeout` | 否 | `45000` | HTTP 总超时（毫秒） | cli-get |
| `--http_connect_timeout` | 否 | `15000` | HTTP 连接超时（毫秒） | cli-get |
| `--http_socket_timeout` | 否 | `45000` | HTTP 读取超时（毫秒） | cli-get |
| `--idle_timeout` | 否 | `90` | 连接池空闲超时（秒） | cli-get |

### 示例

```bash
# 基本用法
./integration

# 前台显式启动
./integration serve

# 后台启动
./integration start

# 停止后台进程
./integration stop

# 重启后台进程
./integration restart

# 开启本机独占模式
./integration standalone set --value true

# 关闭本机独占模式
./integration standalone set --value false

# 导出 Agent
./integration agent export --agent DEF_AGENT --output ./DEF_AGENT.zip

# 导入 Agent
./integration agent import --input ./DEF_AGENT.zip

# 复制 Agent 的工作目录与知识库到另一个已创建的 Agent
./integration agent copy --source DEF_AGENT --target DEF_AGENT_COPY

# 手动触发一条本地通知
./integration notify --title "DeepRight通知" --message "普通对话已完成"

# 完整配置
./integration --agent-dir ./agent --port 8080 --host http://api.example.com:9998 \
  --site ./web --thread 5 --queue 2000 --retry_interval 5000 --retry_times 1 --sleep 1000 \
  --knowledge_update_interval 7200000 --knowledge_update_lock 1800000
```

### 启动行为

- 未传 `--agent-dir` 时，macOS 默认使用 `~/Library/Application Support/deepright/agent`，WSL 默认使用 `~/deepright/agent`，其他系统默认使用启动目录下的 `./agent`
- 如果默认或显式指定的 `agent` 目录不存在，程序会在启动时自动创建
- 如果 `agent` 目录为空（包含刚自动创建完成的场景），程序会先把 `default-dir` 中除根 `config.json` 外的内容复制到 `DEF_AGENT/`，再额外写入一个空的 `DEF_AGENT/config.json`，并确保 `DEF_AGENT/skills` 存在
- 未传 `--default-dir` 时，默认使用启动目录下的 `./config`
- `GET /api/agent/init?name=...` 创建 Agent 时，会把 `default-dir` 中除根 `config.json` 外的内容复制到新 Agent 目录，并额外写入一个空的 `config.json`
- `GET /api/agent/export?agent_id=...` 会导出指定 Agent 的 zip，zip 内保留顶层 Agent 目录，并自动过滤该 Agent 一级目录中的 `chrome*`、`data`、`tmp`
- `POST /api/agent/import` 支持导入 export 生成的 zip 或一个完整 Agent 目录；如果同名 Agent 已存在，会拒绝导入并提示先删除同名 Agent
- `GET /api/copy?source_agentId=...&target_agentId=...` 会把 source Agent 的 `app/`、`data/`、`skills/`、`SOUL.md`、`USER.md`、`Knowledge.md` / `knowledge.md`，以及 `knowledge/<agentId>` 同步到已存在的 target Agent，但不会覆盖 target 的 `config.json`
- 如果 `default-dir` 不存在、不是目录，或复制过程中出错，启动补齐与 `/api/agent/init` 都会失败；`/api/agent/init` 还会回滚刚创建的空 Agent 目录
- 未传 `--site` 时，默认使用启动目录下的 `./site`
- macOS 下共享 sqlite 固定使用 `~/Library/Application Support/deepright/data`，知识库目录固定使用 `~/Library/Application Support/deepright/knowledge`
- WSL 下共享运行目录固定使用 `~/deepright`
- WSL 下插件目录固定使用 `~/deepright/plugins`
- WSL 下知识库目录固定使用 `~/deepright/knowledge`
- WSL 下 `integration.pid` 与 `integration.log` 固定写入 `~/deepright/`
- macOS 下如果以 `.app` 形式运行，插件目录固定读取 `integration.app/Contents/Resources/plugins`
- macOS 下如果通过 `integration.app` 点击启动，会同步拉起一个居中的透明 Logo 浮层；Logo 使用资源目录下的 `site/icon.png`，动画总时长固定约 `5` 秒，并保持原有浏览器异步打开逻辑不变
- 如果应用目录下存在 `config/config.json`，启动时会先读取其中的参数，再由显式 `--参数` 覆盖
- 启动成功后会在后台延迟约 200ms 异步打开浏览器，不会阻塞 integration 主服务启动
- 默认打开地址为 `http://localhost:8080/site/#app`
- 如果你通过 `--port` 指定了其他端口，则会改为 `http://localhost:<port>/site/#app`
- 自动打开浏览器时会优先尝试以最大化窗口启动
- macOS 会按 `Google Chrome`、`Google Chrome for Testing`、`Microsoft Edge`、`Brave Browser`、`Chromium` 的顺序查找浏览器；若都不存在，则回退到系统 `open`
- Linux 会按 `google-chrome`、`google-chrome-stable`、`chromium-browser`、`chromium`、`microsoft-edge`、`microsoft-edge-stable`、`brave-browser` 的顺序查找；若都不存在，则回退到 `xdg-open`
- Windows（含 WSL）会优先查找常见安装目录中的 `Chrome`、`Chrome for Testing`、`Edge`、`Chromium`；若未命中，再从 PATH 中查找 `chrome`、`msedge`、`chromium`、`brave`
- 如果自动打开浏览器失败，只记录日志，不影响 integration 服务继续运行
- `start` 会在后台启动服务并创建 PID 文件
- 如果 `start` 期间因为端口占用或其他启动异常导致服务没有起来，CLI 会把明确失败原因输出到控制台，同时写入 `integration.log`
- `GET /api/standalone` 可读取当前运行中的本机独占状态，`POST /api/standalone=true` 与 `POST /api/standalone=false` 可直接切换该状态
- 也可以通过 `integration standalone get|set|reset` 在本机调用同一套运行时能力
- `POST /api/standalone=true|false` 与 `GET/POST /api/shutdown` 只接受来自 `localhost` 或 `127.0.0.1` 的请求
- 当 `standalone=true` 时，这个 `--port` 端口上的所有 HTTP 服务都只允许 `localhost`、`127.0.0.1`、`::1` 访问，不仅是 `/api/...`，也包括 `/site/...` 等静态页面
- 非本机请求不会收到额外的 JSON 或静态页响应，而是会被服务端直接断开连接
- `standalone` 只修改当前运行进程内存，重启后会恢复为默认关闭
- `stop` 会先尽力停止已配置插件，再释放 HTTP 服务、cli-get、cron、连接池和活跃子进程，最后安全退出
- `POST /api/shutdown` 或 `GET /api/shutdown` 在来自 `localhost` 或 `127.0.0.1` 时会返回接受成功，并在约 5 秒后触发与 `integration stop` 等效的关闭流程
- 这个延迟关闭流程也会先按插件自身 `stop` 命令关闭所有已启动插件，再回收 integration 主进程启动的资源
- 对 `browser` 插件，`stop` 会自动补齐 `--connect-bin` 指向当前 `integration`，确保插件按当前 `integration` 运行目录下的 `plugins/browser_instance.json`、`browser.pid` 清理受管实例和后台进程，不留下残留 `obscura` / monitor 子进程
- 如果 `stop` 时 integration 自身没有可关闭进程，命令默认成功，并清理残留的 PID 文件
- 如果某个已配置插件当前并未启动，`stop` 会直接跳过，不会因此把 `integration stop` 判为失败
- 如果某个插件 `stop` 失败，CLI 会把失败原因输出到控制台和生命周期日志，但不会因此中断 integration 主进程的关闭
- `restart` 等价于先 `stop` 再 `start`
- `knowledge update-time` 通过 `integration` 主二进制直接收口了 Knowledge 更新时间写入能力，无需切换到独立 `knowledge` 二进制
- `knowledge last-update` 通过 `integration` 主二进制直接收口了 Knowledge 最后更新时间读取能力，并与 `/knowledge_lastUpdate` 复用同一份 integration 子模块实现
- `skills-warning` 通过 `integration` 主二进制直接收口了 SKILL 解析告警读取能力
- `file-last-update` 通过 `integration` 主二进制直接收口了文件最后更新时间查询能力
- `backup-clean` 通过 `integration` 主二进制直接收口了 Agent `User/Soul` 备份文件归档与清理能力
- macOS、Windows 与 WSL 下，普通对话与备忘录任务的 SSE 在整体结束后都会触发系统通知；通知标题固定为 `DeepRight通知`
- 普通对话通知内容会直接显示 `用户最后一条问题摘要`；正常完成与异常结束都会沿用同一摘要，超过 `10` 个字符会自动截断为 `...`
- macOS 下通过系统原生通知显示，并跟随当前 `integration` 进程所属应用图标；Windows 与 WSL 下通知使用系统信息图标
- 启动服务后会在后台每分钟自动扫描一次 `SKILL.md`，并把解析失败结果同步到共享 `data` sqlite 的 `skills_warning` 表
- 启动服务后会在后台每 30 秒扫描一次 `add-request` 待处理消息，并在命中后立即桥接生成 `task_detail`
- 当插件配置里的 `router_disable=false` 时，这类由 `add-request` 桥接生成的 `task_meta` / `task_detail` 也会继承同一值
- 备忘录相关的 `router_disable` 默认值固定为 `true`
- 右上角 `SWARM` 开关与 `router_disable` 的映射固定为：`SWARM 开启 -> router_disable=false`、`SWARM 关闭 -> router_disable=true`
- 任务真正执行并转发 `/v1/chat/completions` 时，会优先使用当前 `task_detail.router_disable` 写入 `metadata.router_disable`，不会回退到 Agent `config.json` 里的默认值
- `install_app` 通过 `integration` 主二进制统一收口当前机器待安装应用探测能力

### 启动 Logo 浮层

- 可以单独执行 `./integration splash` 预览与 `integration.app` 相同的 macOS 启动 Logo 动画
- 默认会自动读取当前 integration 资源目录中的 `site/icon.png`
- `--logo` 可覆盖图片路径，`--duration` 可覆盖动画时长，`--async` 会在拉起浮层后立即返回

例如：

```bash
cd /path/to/deepright/cli/module/integration
./integration --agent-dir agent --site site --host https://www.deepright.cn
```

## 构建交付

需求要求最终交付一个主二进制 `integration`，并同时附带 `plugins` 目录中的插件二进制。推荐按下面步骤重新构建到 `module/release`：

```bash
cd /path/to/deepright/cli/module
sh ./build.sh
```

构建完成后的目录结构示例：

```text
module/release/
├── config/
├── integration
└── plugins/
    ├── browser
    ├── email
    └── feishu
    ├── obscura/
    │   └── release/
    │       ├── linux/
    │       └── mac/
    └── playwright/
        └── driver/
```

补充说明：

- `build.sh` 会把 `module/config` 打包到交付物的 `config/` 目录；其中 `config/config.json` 仅供主应用启动读取，不会把其中字段复制进新建 Agent；其他模板文件或目录供新建 Agent / 补齐 `DEF_AGENT` 时复制
- macOS `.app` 形态下，主应用配置固定读取 `integration.app/Contents/Resources/config/config.json`；普通目录交付形态下固定读取应用同级 `config/config.json`
- `plugins/browser` 是浏览器插件二进制，收口了 Playwright、CDP 实例管理和 Chrome Cookie 导出能力
- `plugins/browser name` 固定返回 `{"key":"browser","name":"浏览器"}`
- `plugins/browser param` 固定返回 `["cookie_path"]`
- `plugins/browser command` 会返回插件支持的完整命令集合，便于上层统一探测能力
- `plugins/browser create --agentId ... --chatId ...` 会自动创建或复用实例，并固定使用 `Agent@Chat` 作为 Playwright 会话名
- `integration plugins exec --key browser --command 'instance init' --agentId ... --chatId ...` 可直接通过 integration 主二进制转发任意插件命令
- `plugins/browser start` / `plugins/browser stop` 会关闭 `browser_instance.json` 中记录的全部 CDP 服务，并清理当前 `agent-dir` 下所有 `chrome_${port}` 结构的受管实例目录
- 当这些命令由 `integration` 触发时，会自动携带 `--connect-bin`，因此 Browser 会回到当前 `integration` 的 `plugins/` 目录清理 `browser.pid`、`browser_instance.json` 以及其中记录的 `obscura` / monitor 子进程
- `plugins/browser daemon start|stop|serve` 可单独调试底层 Playwright daemon
- `plugins/browser --browser-timeout ...` 可为单次浏览器命令设置总超时，默认 `120s`
- `plugins/browser` 的 `eval` 同时兼容 `eval '<js>'` 与 `eval --code '<js>'` 两种写法，便于兼容历史上层调用
- `plugins/browser` 的导航类命令会自动读取并注入当前机器 Chrome Cookie，不需要再手工拼 Cookie 参数
- `plugins/browser` 代理的 Playwright / CDP 命令默认会统一使用标准 Chrome UA；在 `attach/create` 受管 CDP 场景下，也会实际覆盖到已连接页面和后续新页面，避免暴露 Linux headless UA
- `plugins/browser` 启动后会从同级目录读取 `plugins/playwright/driver` 与 `plugins/obscura/release/...`
- `browser.log`、`browser.pid`、`browser_instance.json` 也会写在 `plugins/` 目录，与 `browser` 二进制同级；`.browser_playwright/` 作为状态目录保留在同级目录下
- `browser.log` 采用 JSON 行日志，除了插件 `start/stop` 生命周期外，还会记录每一次底层 Playwright 命令执行
- Browser 自动注入 Chrome Cookie 时，会把本次实际注入的 `cookies` 数组、`cookieCount`、目标 host 一并写入 `browser.log`
- 无论通过 `integration plugins start --name browser`、顶层 browser 代理命令还是 browser 内部自动拉起 daemon，Browser 运行态都会优先统一写入当前 `integration` 应用启动目录下的 `plugins/`
- 统一浏览器会在受管 `Agent@Chat` 会话访问时刷新活跃时间；后台默认每分钟按 `browser_instance.json` 里的 `lastActiveAt` 检查一次，并自动释放空闲超过 `--browser_expired` 分钟的实例，默认 10 分钟
- 如需显式覆盖自动释放阈值，可在 `plugins/browser create` 或 `plugins/browser instance create` 上指定 `--browser_expired`

## Agent 元数据中的 `plugins`

- `integration` 在转发 `/v1/chat/completions` 时，会把 Agent 元数据里的 `plugins` 一并注入到请求 `metadata`
- `integration` 的 `cli/get` 与 `cli/pub` 也会带上同一份 Agent 元数据，因此 `plugins` 规则与 HTTP 转发保持一致
- `plugins` 保存的是插件 `key`，不是展示名
- 只有同时满足“已配置且已启动”的插件才会写入该字段
- “已配置”来自 `meta-list` / `ListMetaConfig(false)` 返回的插件配置视图
- “已启动”默认通过插件运行目录下的 `<plugin-key>.pid` 判断；为兼容旧版 `browser` 运行态，`integration` 仍额外兼容 `.browser_playwright/browser_playwright.pid`，只有这些运行态 PID 文件存在且对应进程仍存活时，才会计入 `plugins`
- 如果当前没有任何同时满足“已配置且已启动”的插件，则请求里的 Agent 元数据不会包含 `plugins`
- `meta-list` 本身仍然是“已配置插件”视图，不能直接当作“已启动插件”判断结果使用

## `/v1/chat/completions` Metadata 收口

- `integration` 收口后的 `/v1/chat/completions` 会把请求体中的 `metadata` 与共享 Agent 元数据合并后一起转发到上游
- 发往上游 `/cli/get` 的心跳请求统一为 `{ "messages": [{"role":"user","content":""}], "metadata": ... }`
- 这条链路不再依赖 URL Query 传递业务开关
- 如果请求体中原本已经传了同名 `metadata` 字段，则默认保留请求体传入值；但 `lastResponse`、`sandbox_path` 等运行时收口字段仍会由 integration 按当前会话重新计算后覆盖
- 当请求体中的 `model` 命中 `token_store` 中保存的模型配置时，还会把当前模型下已配置且非空的 `__url`、`__model`、`__model_fast`、`__model_thinking`、`__model_multi_input`、`__model_multi_output` 一并补充到转发 `metadata`
- 这些 `__*` 字段只读取当前请求 `model` 命中的那一条配置；未配置或为空字符串的字段不会出现在转发 `metadata` 中
- 如果某个 Agent 工作目录下的 `config.json.media` 非空，则会把同一份对象补充到 `metadata.agents[].media`
- 如果某个 Agent 工作目录下存在 `Knowledge.md`，则在真正发送 `/v1/chat/completions`、`/cli/get` 或 integration 内部 cron 请求前，会把该文件内容实时补充到对应的 `metadata.agents[].knowledge`
- 如果找不到 `Knowledge.md`，则会继续回退读取同级的 `knowledge.md`
- `Knowledge.md` / `knowledge.md` 只会按 Agent 自己的工作目录逐个读取，不会跨 Agent 复用
- `media` 是 Agent 维度的 JSON 对象；当前 Site 侧会按 `模型服务商名 -> 多组参数` 的结构写入，例如 `"media":{"gemini":{"aspectRatio":"16:9","imageSize":"2K"}}`
- 如果请求体 `metadata` 中显式传入 `knowledge_commit`，则会按 `metadata.agentId` 维度把最新提交值写回共享 sqlite 的 `knowledge_runtime.knowledge_commit`
- 如果请求体 `metadata` 中显式传入 `knowledge_commit: true`，则在对应 SSE 响应完整结束后，会把当前时间同时写回该 Agent 的知识库最后更新时间和知识库更新申请锁时间
- 如果当前会话已经存在 SSE 响应日志，则会额外补充 `metadata.lastResponse`
- `/v1/chat/completions` 会按当前请求里的 `metadata.chat` 查询该会话最近一次 SSE 响应时间，并写成 Unix 毫秒时间戳
- `/cli/get` 当前没有显式 `chat` 字段，因此会从本地 `chat_log` 中取最近一次活跃的 `page_session` 会话，作为“当前 Chat”去查询最近一次 SSE 响应时间
- integration 内部 cron 执行请求也会按自身 `chatId` 补充同样的 `metadata.lastResponse`
- 在真正发送 `/v1/chat/completions`、`/cli/get`，以及 integration 内部 `memo`、`email`、`feishu` 等最终转发到上游 `/v1/chat/completions` 的请求前，还会按最终生效的 `chatId` 重新计算顶层 `metadata.sandbox_path`
- `sandbox_path` 与 `knowledge` 同层，只表示“当前会话可访问的目录白名单路径”；不会挂到 `metadata.agent`、`metadata.agents[]` 或其他 Agent 维度字段下
- 只有当前会话沙盒模式为 `filepick` 或 `filepick_net`，且共享 sqlite 中该 `chatId` 的 `allowed_dir` 在 `trim` 后非空时，才会补充 `metadata.sandbox_path`
- 当当前会话为 `net`、`off`、无沙盒记录，或 `allowed_dir` 为空字符串时，最终报文不会携带 `metadata.sandbox_path`；如果外部请求体里手工传了旧值，也会在转发前被 integration 删除或覆盖
- 发往上游的最终报文会统一收口为：
- `/v1/chat/completions`：`{ "messages": [...], "stream": ..., "metadata": ..., "model": ... }`
- `/cli/get`：`{ "messages": [{"role":"user","content":""}], "metadata": ... }`
- integration 内部 cron 执行请求：`{ "messages": [...], "stream": true, "metadata": ..., "model": ... }`
- `thinking`、`html`、`router_disable` 只会保留在 `metadata` 内，不再继续向上游发送旧的顶层布尔字段
- 转发时如果请求体里自带 `metadata.agent`，integration 也会在发往上游前删掉，只保留 `metadata.agentId` 与 `metadata.agents[]`
- `media` 不跟随 Agent metadata cache；每次真正发送 `/v1/chat/completions`、`/cli/get` 或 integration 内部 cron 请求前，都会重新读取对应 Agent 最新的 `config.json`

### `media` 字段说明

- `POST /api/config?agentId=xxx` 可以直接持久化 Agent 级别的 `media`
- 当前前端会把 `media` 组织为 `模型服务商名 -> 多组参数`，用于描述多模态输出参数
- 当前前端在通过 `POST /api/edit?agentId=xxx&path=config.json` 回写完整 `config.json` 时，请求体也会兼容额外的 `media` 字段，方便与 `/api/config` 使用同一份结构
- 典型 `config.json` 片段如下：

```json
{
  "media": {
    "gemini": {
      "aspectRatio": "16:9",
      "imageSize": "2K"
    }
  }
}
```

例如：

```bash
curl 'http://127.0.0.1:8080/v1/chat/completions' \
  -H 'Content-Type: application/json' \
  -d '{"model":"gpt-4","messages":[{"role":"user","content":"hi"}],"stream":true,"metadata":{"hello":"world","extract":"true"}}'
```

也可以通过 HTTP 触发延迟自关闭：

```bash
curl -X POST 'http://127.0.0.1:8080/api/shutdown'
```

`integration` 实际转发到：

```text
https://www.deepright.cn/v1/chat/completions
```

并把请求体中的 `metadata` 追加成类似：

```json
{
  "hello": "world",
  "extract": "true",
  "__url": "https://provider.example/v1",
  "__model_fast": "deepseek-fast",
  "__model_multi_input": "deepseek-vision"
}
```

- 上例中的 `__url`、`__model_fast`、`__model_multi_input` 仅用于说明当前模型配置命中后的自动补充效果；如果当前模型未配置这些字段，则不会追加

## `/api/plugins/meta`

- `integration` 对外提供 `GET /api/plugins/meta`
- 该接口复用共享的插件元数据实现，每次请求都会实时读取插件目录，不走旧缓存
- 当前只会把以下文件视为插件候选：
  - 无后缀且可执行的程序
  - 后缀为 `.py`、`.js`、`.go` 的脚本文件
- 会跳过目录、隐藏文件以及其他不符合条件的文件
- 如果某个候选文件读取信息或执行 `name` / `param` / `scope` / `command` / `help` 失败，接口会跳过该文件并输出日志，不会因为单个坏文件中断返回
- 每个插件会并发读取 `name`、`param`、`scope`、`command`、`help`
- 返回中的每个插件对象包含：
  - `key`
  - `name`
  - `param`
  - `scope`
  - `meta`
  - `router_disable`
- `param` 使用对象数组格式，例如 `[{"appId":""},{"appSecret":""}]`
- `param` 中每一项的 key 是参数名，value 是占位提示；插件未提供提示时返回空字符串
- `scope` 表示该插件支持哪些容器配置项，可选值为：
  - `reuse`
  - `agent`
  - `provider`
  - `thinking`
  - `swarm`

## `/api/plugins/exec`

- `integration` 对外提供 `GET /api/plugins/exec?key=x&command=y&param1=value1&param2=value2...`
- `key` 为插件主键，`command` 为插件命令文本，支持多级子命令，例如 `instance init`
- 其余 query 参数会被统一转换为 `--param value` 的 CLI 参数后透传给插件
- 如果某个 query 参数值为空，则只透传对应的 `--param`
- `command` 中的空格需要按 URL 规则转义，例如 `instance%20init`
- 如果请求没有显式传 `connect-bin`，integration 会自动补齐当前 integration 主二进制路径
- 返回值中的 `data.command` 为实际执行的参数数组，`data.output` 为插件输出解析结果

例如：

```bash
curl 'http://127.0.0.1:8080/api/plugins/exec?key=browser&command=instance%20init&agentId=A&chatId=chat-001'
```

对应 CLI：

```bash
./integration plugins exec --key browser --command 'instance init' --agentId A --chatId chat-001
```
- 如果插件未实现 `scope` 命令，则默认返回 `["reuse","agent","provider","thinking","swarm"]`

## 文件最后更新时间收口

- HTTP 提供 `GET /file/lastUpdate?file=...&agentId=...`
- CLI 提供 `./integration file-last-update --file ... [--agent/--agentId ...]`
- 返回值是目标文件距离“现在”的毫秒数，纯文本输出
- 当 `file` 是绝对路径时，会直接按绝对路径解析，并兼容大小写不一致
- 当 `file` 是相对路径时，会按指定 Agent 的 workspace 解析，因此必须传 `agentId` 或 `agent`
- 不支持 `~` 路径
- 文件和目录都支持

例如：

```bash
curl 'http://127.0.0.1:8080/file/lastUpdate?agentId=A&file=USER.md'
./integration file-last-update --agent A --file USER.md
./integration file-last-update --file /abs/path/to/USER.md
```

## Agent 备份清理收口

- CLI 提供 `./integration backup-clean [--agent-dir ...] [--archive-after 24h] [--delete-after 72h]`
- 命令会扫描每个 Agent workspace 根目录下与 `USER/SOUL` 相关的备份文件
- 文件名中带 `bak` 或明显时间戳，且最后更新时间超过 `--archive-after` 时，会移动到对应 workspace 下的 `bak/`
- `bak/` 不存在时会自动创建
- `bak/` 中已存在同名文件时，会自动追加递增后缀，避免覆盖旧文件
- `bak/` 目录中的文件如果最后更新时间超过 `--delete-after`，会被直接删除
- 当前生效中的 `USER.md`、`SOUL.md` 不会被误处理；`USER_GUIDE.md` 这类普通文档也不会命中
- 默认阈值分别为 `24h` 和 `72h`
- Agent 根目录解析继续复用 integration 现有优先级：`--agent-dir` > 主应用 `config/config.json` > `AGENT_DIR` > 默认目录
- 该命令属于轻量本地 CLI，不依赖插件运行时初始化，可单独执行

例如：

```bash
./integration backup-clean
./integration backup-clean --agent-dir ./agent
./integration backup-clean --archive-after 24h --delete-after 72h
```

## Agent 元数据中的 `skills`

- `integration` 会在同一份 Agent 元数据里输出每个 Agent 的 `skills`
- `/v1/chat/completions`、`cli/get`、`cli/pub` 以及 integration 内部 cron 执行链路，都会复用同一份 `skills` 输出
- `skills` 字段会在每次请求时实时遍历 Agent 的 `skills` 目录及其子孙目录
- 即使 `integration` 启动时配置了较长的 `--agent-cache`，修改 `SKILL.md` 后，下一次 metadata 输出也会立即反映最新内容
- `--agent-cache` 仍然保留，但主要作用于设备信息、knowledge、plugins 等其他共享字段，不再缓存 `skills` 本身
- `skills[].compatibility` 同时兼容 YAML 字符串和字符串列表两种声明；对外输出时始终规范化为字符串，列表项之间使用 `; ` 连接

## Agent 元数据中的 `config.json` 字段

- `integration` 会在同一份 Agent 元数据里实时输出每个 Agent 的 `description`、`provider`、`thinking`、`router_disable`
- `GET /api/swarm_agent` 会基于这份 Agent 元数据实时过滤出 `router_disable=false` 的 Agent，供 Site 的 `@ Agent` 菜单直接复用
- `/v1/chat/completions`、`cli/get`、`cli/pub` 以及 integration 内部 cron 执行链路，都会复用同一份输出
- 这些字段来自对应 Agent 工作目录下的 `config.json`
- 即使 `integration` 启动时配置了较长的 `--agent-cache`，修改 `config.json` 后，下一次 metadata 输出也会立即反映最新内容

## Agent 元数据中的 `version` 与 `sandbox`

- `integration` 会在同一份 Agent 元数据里输出每个 Agent 的 `version`
- `version` 来自 `--agent-dir/<agentId>/config.json` 中的 `version`
- `version` 只在当前 Agent metadata 缓存周期首次扫描时读取一次；缓存未失效前不会因为 `--agent-dir/<agentId>/config.json` 中的 `version` 变化而立刻刷新
- `version/provider` 不会持久化到 sqlite；`version` 只使用当前 `integration` 进程内的 Agent metadata 内存缓存
- 新建 Agent 与自动补齐出来的 `DEF_AGENT` 默认都会生成空的 `config.json`；如需 `version`、`provider`、`router_disable` 等字段，需要后续显式写入
- `/v1/chat/completions`、`/cli/get` 与 integration 内部 cron 都通过 `metadata.agents[]` 暴露当前 Agent 的 `version`
- `sandbox` 不读 `config.json`；会按当前 `chatId` 实时读取共享 sqlite 的 `cli_sandbox_state`
- 当请求包含有效的 `chatId` 时，对应 `metadata.agent.sandbox` 与当前 Agent 的 `metadata.agents[]` 都会带上实时 `sandbox`
- `agentId` 不参与沙盒状态命中，仅用于定位当前 Agent metadata 与运行日志
- 如果当前请求没有有效 `chatId`，或该会话从未写入沙盒状态，则 `sandbox` 输出空字符串
- `metadata.agent.sandbox` 与 `metadata.agents[].sandbox` 继续只表达沙盒模式，如 `filepick`、`filepick_net`、`net`
- 顶层 `metadata.sandbox_path` 则单独表达当前会话的目录白名单路径；同一 `chatId` 下即使切换不同 `agentId`，只要会话 `allowed_dir` 未变化，最终 `sandbox_path` 也保持一致

## Agent 元数据中的 `git`

- `integration` 会在同一份 Agent 元数据里输出 `git`
- `/v1/chat/completions`、`cli/get`、`cli/pub` 以及 integration 内部 cron 执行链路，都会复用同一份 `git` 输出
- `git` 字段会在每次 metadata 输出前实时重新探测当前机器上的 git 安装路径
- 即使 `integration` 启动时配置了较长的 `--agent-cache`，git 路径变化后，下一次 metadata 输出也会立即反映最新结果
- 如果当前机器未安装 git，则该字段为空字符串

## Agent 元数据中的 `provider`

- `integration` 会在同一份 Agent 元数据里输出每个 Agent 的 `provider`
- `/v1/chat/completions`、`cli/get`、`cli/pub` 以及 integration 内部 cron 执行链路，都会复用同一份 `provider` 输出
- `provider` 字段来自对应 Agent 工作目录下的 `config.json.provider`
- 如果 `config.json` 不存在，或未声明 `provider`，则该字段为空字符串

## `/api/swarm_agent`

```bash
curl http://127.0.0.1:8080/api/swarm_agent
```

成功时返回：

```json
["DEF_AGENT", "planner"]
```

说明：

- `/api/swarm_agent` 仅支持 `GET`；其他方法会返回 `405 Method Not Allowed`
- 接口会实时扫描共享 Agent 元数据，只返回其中 `router_disable=false` 的 Agent ID
- 传入查询参数 `agentId=当前AgentId` 时，返回结果会额外过滤掉当前 Agent 自身
- 返回结果按 Agent 元数据扫描顺序输出
- 如果当前没有任何 Agent 开启 SWARM，则返回空数组 `[]`
- Site 居中会话输入框在当前会话开启 `SWARM` 时，会直接复用这个接口填充 `@ Agent` 菜单

## Install App 接口

```bash
curl http://127.0.0.1:8080/install_app
```

说明：

- `GET /install_app` 返回当前机器待安装应用的 JSON 字符串数组
- 当前已收口的检测项为 `git` 和 `python3`
- 主应用 `config/config.json` 可配置 `install_app`，并按当前操作系统读取 `linux`、`wsl`、`mac` 对应数组
- Linux 读取 `install_app.linux`，macOS 读取 `install_app.mac`，Windows/WSL 读取 `install_app.wsl`
- 启动时可通过 `--install_app a,b,c` 追加自定义待安装应用
- `install_app` 中的每个元素都表示一个本地应用名；接口会按当前操作系统检查是否已安装，已安装项不会出现在返回列表中
- 接口会把自动探测结果、`config/config.json` 当前系统对应配置、`--install_app` 指定值做去重合并，并对安装状态缓存 5 分钟
- `config/config.json` 示例：

```json
{
  "install_app": {
    "linux": ["node", "python"],
    "wsl": ["node", "python", "docker"],
    "mac": ["node", "python", "xcode-select"]
  }
}
```

- 如果 `git` 和 `python3` 都未安装，则返回：

```json
["git", "python3"]
```

- 如果使用：

```bash
./integration --install_app node,python,git,python3
```

则接口可能返回：

```json
["git", "node", "python", "python3"]
```

- 如果未检测到任何已支持应用，则返回空数组 `[]`

## Agent 元数据中的 `knowledge`

- `integration` 会在同一份 Agent 元数据里按需补充 `knowledge`
- 该字段与 `plugins` 一样属于可选字段，不满足条件时不会输出空对象
- 一旦出现，会在以下链路中保持一致：
  - `/v1/chat/completions`
  - `cli/get`
  - `cli/pub`
  - integration 内部发起的 cron 执行请求
- `knowledge` 结构如下：

```json
{
  "knowledge": {
    "path": "/abs/path/knowledge",
    "lastUpdate": 0
  }
}
```

- `path`
  - 未指定 Agent 时固定为 `--agent-dir/knowledge` 绝对路径
  - 指定 `metadata.agentId` 时固定为 `--agent-dir/knowledge/<agentId>` 绝对路径
- `lastUpdate`
  - 来自应用启动目录下共享 `data` sqlite 的 `knowledge_runtime.last_update`
  - 未初始化更新时间时为 `0`
- `knowledgeCommit`
  - 来自共享 sqlite 的 `knowledge_runtime.knowledge_commit`
  - 与 `lastUpdate` 一样按 `agent_id` 独立保存
- 转发 `/v1/chat/completions` 前还会额外检查 `knowledge.lastUpdate`
  - 如果 `lastUpdate` 距离当前请求时间未超过 `--knowledge_update_interval`（默认 `7200000` 毫秒，即 2 小时），则转发前删除 `knowledge.lastUpdate`
  - 如果已超过该时间，则继续检查共享 sqlite 中最近一次知识库更新申请时间
  - 如果该申请时间距离当前请求未超过 `--knowledge_update_lock`（默认 `1800000` 毫秒，即 30 分钟），同样删除 `knowledge.lastUpdate`
  - 只有知识库已过期且锁窗口也已过期时，才保留 `lastUpdate` 原样转发，并把当前请求时间写入共享 sqlite
- 如果请求体 `metadata` 中显式传入 `knowledge_commit: true`，则这次请求转发时会强制保留 `knowledge.lastUpdate`，不再检查 `knowledge_update_interval` 和 `knowledge_update_lock`
- 若 `--agent-dir/knowledge` 目录不存在，则 metadata 中不会包含该字段
- `integration` 不在入口层单独维护 knowledge 逻辑，而是直接复用共享的 Agent 元数据内核输出

## 统一日志

- `integration` 已收口：
  - `/v1/chat/completions` 请求日志
  - `/v1/chat/completions` SSE 响应日志
  - `cli/get` 日志
  - `cli/pub` 日志
- 统一日志表为当前应用目录 `data` SQLite 中的 `agent_message_log`
- 表字段：
  - `agent_id`
  - `chat_id`
  - `content`
  - `log_type`
  - `created_at`
- 索引为 `agent_id + chat_id + log_type + created_at`
- 另外还新增了 `chat_id + log_type + created_at` 索引，用于加速按会话查询最近一次 SSE 响应时间，也就是 `metadata.lastResponse` 的查找路径
- `log_type` 固定取值：
  - `0`：`/v1/chat/completions` 请求
  - `1`：`/v1/chat/completions` SSE 响应分段
  - `2`：`cli/get`
  - `3`：`cli/pub`
- `cli/get` 只有在服务端返回可执行任务时才记录日志；当响应 `content` 为 `null` 或空字符串时不记录
- `cli/pub` 统一日志中的 `content` 保存的是 `GZIP+Base64` 之前的原始执行结果，便于直接恢复与导出查看
- `/api/restore` 现在会额外合并返回同一 `agentId + chat` 下的 `cli/get` 与 `cli/pub` 记录，并继续保持统一 `data[]` 时间线输出
- 合并返回的 CLI 记录会保留原始 `content`，不在 restore 链路中提前裁剪成摘要，便于 site 直接恢复右侧 `CMD` 子任务历史
- 合并排序固定为先按 `createdAt` 升序，再按 `id` 升序，保证前端可以按单一时间线消费消息与 CLI 事件

### `http.debug` 明细日志

- 主应用 `config/config.json` 可通过 `http.debug=true` 开启 `cli/get` / `cli/pub` 的详细日志，默认关闭
- 该日志写入 `integration.log` / 标准日志，不额外写入统一 SQLite 日志表
- 开启后会记录：
  - `cli/get` 请求远程主机超时时间，并附带本次耗时和当前 HTTP 超时配置
  - `cli/get` 返回待执行任务时的原始报文和时间
  - `cli/pub` 回传执行结果时的状态、原始结果和时间

## 最近轮次日志导出

- `integration` 提供 `GET /log_round?agentId=xxx&chatId=yyy&round=zzz`
- 同时提供 `GET /log_skill?agentId=xxx&chatId=yyy&round=zzz&start=aaa&close=bbb`
- 以及 `GET /log_skill_status?agentId=xxx&chatId=yyy&round=1`
- 也支持 CLI：

```bash
./integration log-round --agent A --chat chat-001 --round 3
```

- `round` 以 `/v1/chat/completions` 请求为轮次边界
- `round=1` 表示最后一轮
- `round=3` 表示从倒数第三轮开始导出到当前最新日志
- 导出数据包含：
  - 会话请求
  - 所有 SSE 响应
  - `cli/get`
  - `cli/pub`
- 多段 SSE 响应会在导出时合并为一条 Markdown 表格记录
- 导出文件会写入对应 Agent 工作目录下的 `tmp/`，并返回绝对路径
- `/log_skill` 返回导出文件绝对路径与 `sizeK`
- `/log_skill_status` 会检查当前会话最近一轮完整 SSE 响应里的 `cli/get` 次数
- 默认阈值来自主应用 `config/config.json.skill_extract`；未配置时默认值为 `10`
- 如果显式传入 `round`，会把它当作本次检查使用的阈值覆盖默认值
- 导出 Markdown 时，内容列会输出可读结果而不是原始报文：
  - `请求`：优先提取 `/v1/chat/completions` 请求里的 `messages[].content`，兼容单条 `message`
  - `响应`：提取并合并 SSE `delta.content`
  - `工具请求（cli/get）`：从 `cli/get` 返回任务的 `content` JSON 中提取 `cmd`
  - `工具响应（cli/pub）`：提取 `messages[].content`，历史原始执行结果日志则直接输出原文

## Knowledge CLI 收口

按 `integration` 的二进制和 CLI 收口原则，Knowledge 的更新时间写入命令也可以直接从 `integration` 调用：

```bash
./integration knowledge update-time --timestamp 1715337600000 --agentId demo-agent
./integration knowledge update-time --timestamp 1715337600000 --agent-id demo-agent
```

说明：

- 该命令会复用共享 `knowledge` 模块能力，不在 `integration` 内重复实现表结构与状态逻辑
- `update-time` 现在要求同时传 `--timestamp` 和 `--agentId`
- 默认写入当前应用启动目录对应的共享 sqlite：`<app-dir>/data`
- `lastUpdate` 统一使用 Unix 毫秒时间戳
- 成功后会输出当前 Knowledge 状态 JSON，其中包含：
  - `path`
  - `lastUpdate`
- 这也意味着通过 `integration` 更新后的 `lastUpdate`，会被同一启动目录下的 metadata 复用到：
  - `/v1/chat/completions`
  - `cli/get`
  - `cli/pub`
  - cron 执行链路

## 生命周期日志

- `integration start`、`stop`、`restart` 的关键行为都会追加写入 `integration.log`
- 如果后台 `start` 失败，例如端口已被占用、监听失败或其他启动前异常，失败原因会同时出现在：
  - 当前命令的控制台 stderr
  - `integration.log` 中的 `[integration lifecycle] start failed: ...`
- `start` 在等待就绪期间如果子进程提前退出，也会优先透传子进程上报的失败原因，而不是只给出笼统超时
- `stop` 在目标进程已经不存在时仍会按成功处理；如果插件 PID 已缺失、插件进程已退出，生命周期日志会记录 `plugin stop skipped` 或清理陈旧 PID，而不会把整个停止流程判为失败

### `config/config.json` 运行态字段

- 以 HTTP 服务模式启动 `integration` 时，会把解析后的实际运行参数直接写回主应用 `config/config.json`
- 文件中会同时保留静态启动配置和运行态字段；例如 `app`、`app-dir`、`resources-dir`、`db`
- 其中 `db` 会写入应用启动目录下 `data` 的绝对路径
- `integration stop` 不会删除主应用 `config/config.json`
- `knowledge` 元数据会基于 `agent-dir` 和共享 sqlite 共同解析；也就是说：
  - 知识库目录固定取 `--agent-dir/knowledge`
  - 共享 sqlite 仍默认取 `<app-dir>/data`

示例：

```json
{
  "app": "/absolute/path/to/integration",
  "app-dir": "/absolute/path/to",
  "db": "/absolute/path/to/data",
  "port": 8080,
  "host": "https://www.deepright.cn",
  "agent-dir": "/absolute/path/to/agent",
  "device": "",
  "agent-cache": 120000,
  "site": "/absolute/path/to/site",
  "connect_timeout": 15000,
  "sleep": 3000,
  "thread": 20,
  "queue": 1000,
  "retry_interval": 10000,
  "retry_times": 1,
  "http_timeout": 45000,
  "http_connect_timeout": 15000,
  "http_socket_timeout": 45000,
  "http_debug": false,
  "idle_timeout": 90
}
```

## CLI 示例

Integration 的 CLI 既可以启动统一 HTTP 服务，也支持通过 `cron` 子命令直接创建和查询备忘录（定时任务）数据。

## 统一 metadata

`integration` 会统一复用 `agentcore` 输出 Agent 元数据，不在入口层重复实现字段拼装逻辑。

当前以下链路都会使用同一份 metadata：

- `/v1/chat/completions`
- `cli/get`
- `cli/pub`
- integration 内部发起的 cron 执行请求

其中：

- `plugins` 与 `knowledge` 仍按共享元数据统一输出
- `agents[].skills` 会在每次请求时实时刷新
- `/v1/chat/completions`、`/cli/get` 与 integration 内部 cron 请求发送前，会再按每个 Agent 工作目录实时读取 `Knowledge.md`；若不存在则回退到 `knowledge.md`，并补充到对应的 `agents[].knowledge`
- `/v1/chat/completions`、`/cli/get` 与 integration 内部 cron 请求发送前，还会按当前会话补充 `lastResponse`
- 该字段表示当前 Chat 最近一次收到 SSE 响应的时间，格式为 Unix 毫秒时间戳

如果当前应用启动目录下已存在 `knowledge` 目录，则上述链路中的 metadata 会额外包含：

```json
{
  "knowledge": {
    "path": "/app/knowledge",
    "lastUpdate": 0
  }
}
```

说明：

- `path` 为 `--agent-dir/knowledge` 的绝对路径
- `lastUpdate` 来自 `<app-dir>/data` 中的 `knowledge_runtime.last_update`
- 若知识库尚未初始化，则 metadata 中不会出现 `knowledge`

## `/mapping` 静态映射

`integration` 新增了 `GET /mapping/$agentId/*` 静态资源映射，用来直接暴露每个 Agent 目录下的 `app/` 内容。

目录关系示例：

```text
runtime/
├── agent/
│   ├── agent-a/
│   │   └── app/
│   └── agent-b/
│       └── app/
└── knowledge/
```

访问规则：

- `GET /mapping/<agentId>/`
  - 直接映射到 `<agent-dir>/<agentId>/app/`
- `GET /mapping/<agentId>/<相对路径>`
  - 直接返回对应静态文件内容
- 与 nginx 静态目录映射一致，适合托管静态生成的 HTML、JS、CSS 与图片资源

约束：

- 只支持 `GET` / `HEAD`
- 路径必须包含 `agentId`
- 根目录固定解析为 `filepath.Join(--agent-dir, agentId, "app")`
- 路径不能跳出 `app` 根目录；如 `..` 会返回 `400`
- `app` 目录不存在时返回 `404`，后续补齐目录与文件后可直接访问，无需重启服务

示例：

```bash
curl http://127.0.0.1:8080/mapping/agent-a/
curl http://127.0.0.1:8080/mapping/agent-a/index.html
curl http://127.0.0.1:8080/mapping/agent-a/assets/main.js
```

## `/knowledge` 静态映射

`integration` 也直接收口了 Knowledge 的静态访问入口：

- `GET /knowledge`
  - 返回当前 `--agent-dir/knowledge/` 根目录的树形结构
- `GET /knowledge/<相对路径>`
  - 访问目录时返回该目录的树形结构
  - 访问文件时直接返回文件原始内容

约束：

- 只支持 `GET`
- 根目录固定解析为 `--agent-dir/knowledge`
- 路径不能跳出 `knowledge` 根目录；如 `..` 会返回 `400`
- `knowledge` 目录不存在时返回 `404`

示例：

```bash
curl http://127.0.0.1:8080/knowledge
curl http://127.0.0.1:8080/knowledge/README.md
curl http://127.0.0.1:8080/knowledge/docs/guide.txt
```

## `/knowledge_path`

`integration` 也直接收口了 Knowledge 的真实路径接口：

- `GET /knowledge_path`
  - 未传 `agentId` 时，返回当前 `--agent-dir/knowledge` 根目录的真实文件系统绝对路径
  - 传 `agentId` 时，返回 `--agent-dir/knowledge/<agentId>` 的真实文件系统绝对路径

约束：

- 只支持 `GET`
- 根目录固定解析为 `--agent-dir/knowledge`
- `agentId` 非空时会继续拼接为 `--agent-dir/knowledge/<agentId>`
- `knowledge` 目录不存在时返回 `404`
- 如果该路径存在但不是目录，返回 `500`

示例：

```bash
curl http://127.0.0.1:8080/knowledge_path
curl 'http://127.0.0.1:8080/knowledge_path?agentId=agent-a'
```

## `/knowledge_lastUpdate`

`integration` 也直接收口了 Knowledge 的最后更新时间接口：

- `GET /knowledge_lastUpdate`
  - 返回当前知识库最后更新时间
  - 输出格式为 `yyyy-MM-dd HH:mm`
  - 传 `agentId` 时返回对应 Agent 的知识库最后更新时间
- `./integration knowledge last-update`
  - 返回与 HTTP 接口相同格式的知识库最后更新时间

说明：

- 底层数据来自 `<app-dir>/data` 中按 `agent_id` 保存的 `knowledge_runtime.last_update`
- HTTP 与 CLI 入口复用同一份 integration 内部 `knowledge` 子模块，不再在入口层重复格式化逻辑
- 若知识库尚未初始化，则返回值以当前运行时实现为准

同时也直接收口了 `connect` 与插件相关 CLI，无需单独切换到 `connect` 二进制：

```bash
cd /path/to/deepright/cli/module/integration
./integration list-plugins
./integration connect meta-list
./integration plugins start --name feishu
./integration plugins stop --name feishu
./integration plugins status --name feishu
```

也支持直接查看各层命令帮助：

```bash
./integration help
./integration cron create --help
./integration cron create-cron --help
./integration cron find-meta --help
./integration cron find-detail --help
./integration cron delete-meta --help
./integration cron delete-detail --help
./integration token --help
./integration connect --help
./integration plugins --help
```

说明：

- `integration help` 负责列出所有顶层可直接调用的生命周期、cron、connect、plugins 命令
- `integration token` 可直接读取共享 sqlite 中保存的模型与密钥，不必单独调用 HTTP `/api/token`
- `integration token --agentId A --function 完成任务 --thinking 1 --input 2 --total 3 --cache 4 --model deepseek` 会直接写入一条 Token 消费明细
- 如果不写 `token` 子命令，`integration` 会按默认服务模式解析参数；像 `--agentId`、`--thinking` 这类消费字段必须写在 `integration token` 后面
- Connect 相关 CLI 统一从 `integration connect <subcommand>` 进入，例如 `integration connect meta-create`、`integration connect meta-get`、`integration connect meta-list`
- `integration cron <subcommand> --help` 会给出对应命令的参数说明、注意事项和案例
- `integration plugins --help` 会给出插件 start/stop/status 的说明和案例
- `integration cron create` / `integration cron create-cron` 额外支持 `--schema JSON`
- `integration cron create` / `integration cron create-cron` 使用 `--router_disable[=true|false]`，不传时默认 `true`
- `--schema` 会写入 `task_meta.response_schema`，周期拆分出的每条 `task_detail` 都会继承它
- `cycle=0` 的一次性任务会把 `--schema` 直接继承到首条明细
- `--router_disable` 会写入 `task_meta.router_disable`；创建时同步生成的首批 `task_detail` 也会继承同一值
- cron 执行时如果 `response_schema` 非空，会透传到上游 LLM 请求的 `response_format.type=json_schema`

启动服务示例：

```bash
cd /path/to/deepright/cli/module/integration
./integration --port 8080 --host http://127.0.0.1:9998
```

直接通过 CLI 创建周期任务：

```bash
cd /path/to/deepright/cli/module/integration
./integration cron create --content "每15分钟检查一次上游接口健康" --model "OpenAI" --thinking true --router_disable false --rawTime "2026-05-03 10:00" --cycle 4 --chatId "chat-001" --agent "demo-agent"
./integration cron create --content "整理日报" --schema '{"type":"object","properties":{"summary":{"type":"string"}},"required":["summary"]}' --model "OpenAI" --rawTime "2026-05-08 09:00" --cycle 0 --agent "demo-agent"
```

直接通过 CLI 创建自定义 Cron：

```bash
cd /path/to/deepright/cli/module/integration
./integration cron create-cron --content "工作日中午整理异常日志" --model "OpenAI" --thinking true --router_disable false --cron "10 12 * * 1-5" --chatId "chat-001" --agent "demo-agent"
./integration cron create-cron --content "提取结构化日报" --schema '{"type":"object","properties":{"todo":{"type":"array"}},"required":["todo"]}' --model "OpenAI" --cron "0 18 * * 1-5" --agent "demo-agent"
```

直接通过 CLI 查询任务元数据：

```bash
cd /path/to/deepright/cli/module/integration
./integration cron find-meta --content "每15分钟检查一次上游接口健康" --model "OpenAI" --chatId "chat-001"
```

直接通过 CLI 查询任务明细：

```bash
cd /path/to/deepright/cli/module/integration
./integration cron find-detail --metaId cron_1 --content "每15分钟检查一次上游接口健康" --model "OpenAI" --cycle 4 --chatId "chat-001"
```

直接通过 CLI 删除任务元数据：

```bash
cd /path/to/deepright/cli/module/integration
./integration cron delete-meta --id meta_1
```

直接通过 CLI 删除任务明细：

```bash
cd /path/to/deepright/cli/module/integration
./integration cron delete-detail --detailId detail_1
```

直接通过顶层 CLI 读取模型密钥：

```bash
cd /path/to/deepright/cli/module/integration
./integration token
./integration token --provider deepseek
./integration token --agentId A --function 完成任务 --thinking 1 --input 2 --total 3 --cache 4 --model deepseek
```

输出示例：

```json
[
  {
    "deepseek": {
      "token": "aaa",
      "__url": "https://api.example.com/v1",
      "__model": "deepseek-chat",
      "__model_fast": "deepseek-fast",
      "__model_thinking": "deepseek-reasoner"
    }
  }
]
```

```json
{
  "deepseek": {
    "token": "aaa",
    "__url": "https://api.example.com/v1",
    "__model": "deepseek-chat",
    "__model_fast": "deepseek-fast",
    "__model_thinking": "deepseek-reasoner"
  }
}
```

直接通过顶层 CLI 查看可用插件：

```bash
cd /path/to/deepright/cli/module/integration
./integration list-plugins
```

返回示例：

```json
[
  {
    "key": "email",
    "name": "邮件",
    "param": [
      {
        "email": ""
      },
      {
        "email_pop3": ""
      },
      {
        "email_smtp": ""
      },
      {
        "email_password": ""
      },
      {
        "email_whitelist": ""
      }
    ]
  },
  {
    "key": "feishu",
    "name": "飞书",
    "param": [
      {
        "appId": ""
      },
      {
        "appSecret": ""
      }
    ]
  }
]
```

- `param` 字段和 `/api/plugins/meta` 一致，统一使用对象数组格式

直接通过顶层 CLI 查看已配置插件：

```bash
cd /path/to/deepright/cli/module/integration
./integration connect meta-create --key feishu --meta '{"appId":"cli-app","appSecret":"cli-secret"}' --callback ignored --agent A --chatId chat-001 --model OpenAI
./integration connect meta-update --key feishu --meta '{"appId":"new-app","appSecret":"new-secret"}' --callback ignored --agent A --chatId chat-002 --model OpenAI
./integration connect meta-list
./integration connect meta-get --key feishu
```

直接通过插件命令管理插件进程：

```bash
cd /path/to/deepright/cli/module/integration
./integration plugins start --name feishu --connect-bin ./integration
./integration plugins status --name feishu
./integration plugins stop --name feishu --pid-file ./plugins/feishu.pid
```

返回示例：

```json
[
  {
    "key": "feishu",
    "name": "飞书",
    "meta": {
      "appId": "cli-app",
      "appSecret": "cli-secret"
    },
    "stream": true,
    "callback": "/abs/path/plugins/feishu",
    "agentId": "A",
    "chatId": "chat-001",
    "model": "OpenAI",
    "thinking": true,
    "createdAt": "2026-05-05T10:00:00+08:00",
    "updatedAt": "2026-05-05T10:00:00+08:00"
  }
]
```

说明：

- `key` 是插件运行时主键，后续启动、停止、日志和配置链路应优先使用它
- `name` 是插件展示名，可用于 UI 展示或兼容旧输入
- `meta-create` / `meta-update` 推荐统一使用 `--key <plugin-key>`，`--name` 只保留兼容
- 最终用户优先使用 `./integration connect meta-get --key <plugin-key>` 读取单个插件配置
- `list-plugins` 不依赖 connect 服务是否启动，会直接扫描当前插件目录；macOS `.app` 形态固定为 `integration.app/Contents/Resources/plugins`，其他场景默认是当前启动目录下的 `./plugins`
- `list-plugins` 与 `/api/plugins/meta` 使用同一套本地扫描规则：仅识别无后缀可执行程序，以及 `.py` / `.js` / `.go` 脚本文件；目录和异常文件会被跳过并记录日志
- 在 `integration` 中，`meta-list` 返回的是已配置插件视图，`meta` 已经从 JSON 字符串解析为对象
- 如需查看 Connect 独立二进制的原始数据库记录视图，请直接使用 `./connect meta-list`
- `connect meta-get` 仅作为底层实现与兼容入口保留

直接通过顶层 CLI 写入三方请求：

```bash
cd /path/to/deepright/cli/module/integration
./integration connect add-request --key feishu --externalId msg-1 --content "HELLO WORLD"
./integration connect add-request --key feishu --content "HELLO WORLD" --artifacts "/tmp/a.txt,/tmp/b.txt" --original '{"text":"HELLO WORLD"}'
./integration connect add-request --key feishu --content "HELLO WORLD" --status 1 --created 1777852800
./integration connect add-request --key feishu --content "提取结构化消息" --schema '{"type":"object","properties":{"title":{"type":"string"}},"required":["title"]}'
```

返回示例：

```json
{
  "id": 1,
  "key": "feishu",
  "name": "feishu",
  "externalId": "msg-1",
  "content": "HELLO WORLD",
  "request": "HELLO WORLD",
  "status": 0,
  "createdAt": "2026-05-02T00:00:00Z"
}
```

说明：

- 最终用户主流程优先使用 `./integration connect add-request`
- `--key` 为推荐主参数，表示插件运行时主键；`--name` 仍保留作兼容输入
- `--content` 为推荐主参数；`--request` 仍保留作兼容输入
- `--original` 为推荐主参数；`--raw-request` 仍保留作兼容输入
- `--externalId` 与 `key` 组成唯一键；重复写入会直接报错
- `--status` 支持 `0=待处理`、`1=已启动`、`2=已完成`、`3=已过期`、`4=已回复`
- `--created` 支持 Unix 时间戳或 RFC3339 时间；不传时默认写当前时间
- `--schema` 为新可选参数，值为 Json String
- 该参数会先持久化到 `connect_request.response_schema`
- 在桥接成一次性 cron 任务时，会透传给任务明细的 `task_detail.response_schema`
- 该命令会复用启动时初始化的共享 SQLite 连接，不会为每次请求单独开关数据库

## Response Schema 透传

- `POST /api/connect/request` 支持 `schema` 参数，CLI `integration connect add-request` 支持 `--schema`
- connect 请求桥接为一次性 cron 任务时，会继承最后一条 request 的 `responseSchema`
- `integration cron find-meta` / `integration cron find-detail` 返回结果新增 `responseSchema`
- cron 审计日志同样会记录 `responseSchema`

直接通过顶层 CLI 查询请求与写入响应：

```bash
cd /path/to/deepright/cli/module/integration
./integration connect request-list --name feishu --status 0 --limit 20
./integration connect add-response --name feishu --request-id 1 --response "HELLO BACK"
./integration connect response-list --name feishu --request-id 1 --limit 20
```

说明：

- `request-list` 支持 `--status`、`--after-id`、`--limit`
- `add-response` 成功后，会自动把对应请求状态更新为 `4`
- `response-list` 可按 `--name`、`--request-id`、`--after-id`、`--limit` 查询

直接通过顶层 CLI 启停插件：

```bash
cd /path/to/deepright/cli/module/integration
./integration plugins start --name feishu
./integration plugins stop --name feishu
./integration plugins status --name feishu
```

说明：

- `plugins start` 会自动补齐 `--connect-bin` 指向当前 `integration` 二进制
- `plugins stop` 也会自动补齐 `--connect-bin`，避免 Browser 插件停机时误读错误运行目录而留下 `obscura` / monitor 残留进程
- `--name` 既支持插件 `key`，也兼容展示名或插件路径
- `plugins stop` / `plugins status` 支持额外传入 `--pid-file`

如果你想通过 HTTP API 创建 cron 任务，也可以调用统一接口：

```bash
curl -X POST 'http://127.0.0.1:8080/api/cron/create?agentId=demo-agent' \
  -H 'Content-Type: application/json' \
  -d '{
    "content": "每15分钟检查一次上游接口健康",
    "model": "OpenAI",
    "thinking": true,
    "rawTime": "2026-05-03 10:00",
    "cycle": 4,
    "chatId": "chat-001",
    "type": "cron"
  }'
```

自定义 Cron 示例：

```bash
curl -X POST 'http://127.0.0.1:8080/api/cron/create?agentId=demo-agent' \
  -H 'Content-Type: application/json' \
  -d '{
    "content": "工作日中午整理异常日志",
    "model": "OpenAI",
    "thinking": true,
    "cycle": -1,
    "cron": "10 12 * * 1-5",
    "chatId": "chat-001",
    "type": "cron"
  }'
```

## 服务架构

```
                    ┌─────────────────────────────────┐
                    │     Integration (:8080)          │
                    │                                  │
  POST /v1/chat/ ──►  Proxy 模块                      │
  completions       │  (注入metadata, SSE流式转发)     │──► 上游服务
                    │                                  │
  GET /api/agentId ►  Proxy 模块                      │
                    │  (返回所有AgentId列表)            │
                    │                                  │
  GET /api/deviceId ► Proxy 模块                      │
                    │  (返回共享 deviceId)             │
                    │                                  │
  GET /site/* ──────►  Static 模块                     │
                    │  (静态文件服务)                   │
                    │                                  │
                    │  cli-get 模块 (后台线程)          │
                    │  (心跳上报 + 本地队列 + 执行 +    │
                    │   发布重试)                       │──► 上游服务
                    │                                  │
                    │  cron 模块 (后台线程 + HTTP API)  │
                    │  (定时生成任务 / 执行 / 落库)      │──► 上游服务
                    └─────────────────────────────────┘
```

## HTTP 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/v1/chat/completions` | SSE 流式代理，注入 Agent 元数据 |
| GET | `/api/agentId` | 返回所有 Agent ID 的 JSON 数组 |
| GET | `/api/swarm_agent` | 返回当前已开启 SWARM 的 Agent ID 数组 |
| GET | `/api/deviceId` | 返回共享 `deviceId` 的 JSON 对象 |
| GET | `/api/plugins/meta` | 返回当前可用插件的名称、参数定义与已填参数 |
| GET | `/api/plugins/status` | 返回指定插件当前是否已启动 |
| POST | `/api/plugins/config` | 创建或更新指定插件配置 |
| GET | `/api/plugins/log` | 以 SSE 持续返回指定插件日志 |
| POST | `/api/plugins/start` | 启动指定插件 |
| POST | `/api/plugins/stop` | 停止指定插件 |
| POST | `/api/cmd` | 仅本机可调用的命令执行接口 |
| POST | `/api/kill` | 仅本机可调用的命令终止接口 |
| GET | `/api/folder` | 打开 Agent 工作目录 |
| GET | `/api/skills` | 获取 Agent 技能列表 |
| GET | `/api/files` | 浏览文件列表 |
| POST | `/api/skill_state` | 按会话切换技能目录禁用状态 |
| GET | `/api/data` | 读取文本文件 |
| GET | `/api/workspace` | 获取 Agent 工作目录路径 |
| POST | `/api/edit` | 写入文本或二进制文件 |

补充说明：

- 当请求来源不是 `localhost`、`127.0.0.1` 或 `::1` 时，插件管理接口只保留只读能力
- 远程仍可读取 `/api/plugins/meta`、`/api/plugins/status`、`/api/plugins/log`
- 其中 `/api/plugins/meta` 会返回脱敏后的插件列表
- 远程请求会被拒绝访问 `/api/plugins/config`、`/api/plugins/start`、`/api/plugins/stop`、`/api/plugins/exec`
- 这里的“远程请求”以访问入口 Host 为准；通过非 `localhost` / `127.0.0.1` / `::1` 的域名、别名、LAN IP 或反代入口访问时，也会按远程模式处理
- `/api/folder` 在 WSL 场景下会优先尝试 `xdg-open`
- 如果 `xdg-open` 无法把目录窗口带到前台，integration 会继续尝试通过 PowerShell 拉起 Explorer，并按目标目录匹配实际窗口后执行置前
- 如果前台化分支失败，仍会回退到既有 `explorer.exe` / `cmd.exe` 打开链路，避免目录完全打不开
- 站点里的消息路径浮层和左侧虚拟文件系统都复用 `/api/folder`
- 当前端收到 `/api/folder` 成功响应时，会补一个轻提示：`目录已尝试置前打开，若仍未看到请查看任务栏`

如果当前 Agent 工作目录下的 `config.json.router_remote` 已保存有效值，则右侧备忘录生成的任务明细和插件桥接生成的任务明细在真正执行 `/v1/chat/completions` 时，也会额外注入：

```json
{
  "metadata": {
    "router_remote": ["remote_1", "remote_2"]
  }
}
```
| GET | `/api/del` | 删除文件或目录 |
| GET | `/api/raw` | 读取文件 Base64 原文 |
| GET | `/api/heartbeat` | 查看 cli-get 心跳状态 |
| GET | `/api/agent/init` | 新建 Agent 目录，并复制默认模板配置 |
| GET | `/api/copy` | 复制指定 Agent 的工作目录与知识库到另一个已存在 Agent |
| GET | `/api/agent/delete` | 删除 Agent 目录 |
| GET | `/api/agent/create` | 在 Agent 下新建文件或目录 |
| POST | `/api/upload` | 上传文件到 Agent 临时目录 |
| POST | `/api/config` | 更新 Agent 配置；也支持删除指定模型配置 |
| GET/POST | `/api/token` | 保存或读取模型密钥 |
| POST | `/api/message_insert/add` | 新增或覆盖一条待上传插入消息 |
| POST | `/api/message_insert/del` | 将指定插入消息标记为取消 |
| POST | `/api/message_insert/delete` | 物理删除指定插入消息的未终态记录 |
| POST | `/api/cron/create` | 创建定时任务 |
| POST | `/api/cron/detail/metadata` | 查询定时任务元数据 |
| POST | `/api/cron/delete` | 删除定时任务 |
| POST | `/api/cron/detail/delete` | 删除任务明细 |
| POST | `/api/cron/detail/list` | 查询任务明细 |
| POST | `/api/cron/detail/status` | 更新任务明细状态 |
| POST | `/api/cancel` | 取消指定会话流 |
| POST | `/api/restore` | 恢复指定会话日志 |
| GET | `/api/download` | 下载文件或目录 |
| GET | `/site/*` | 静态文件服务 |

## /api/agent/init

`GET /api/agent/init?name=xxx`

说明：

- 仅负责创建新的 Agent 目录
- 创建成功后，会把 `--default-dir` 指向目录中除根 `config.json` 外的内容复制到该 Agent 目录，并额外生成空的 `config.json`
- 未显式传 `--default-dir` 时，默认复制当前应用启动目录下的 `config/`
- 如果 `default-dir` 不存在、不是目录或复制失败，会返回错误且不会留下半成品 Agent 目录
- `name` 仍然必须是单段 Agent 名称，不能包含空格或 ` /\\:*?"<>|`

## /api/copy

`GET /api/copy?source_agentId=xxx&target_agentId=yyy`

说明：

- `target_agentId` 必须已经存在；该接口不会自动创建新 Agent
- 会同步 `app/`、`data/`、`skills/`、`SOUL.md`、`USER.md`、`Knowledge.md` / `knowledge.md`，以及同级 `knowledge/<agentId>`
- 如果 source Agent 同时存在 `Knowledge.md` 和 `knowledge.md`，则优先同步 `Knowledge.md`，与运行时透传规则保持一致
- target 上这些路径会先按 source 状态重建；如果 source 缺失某个受管路径，对应 target 路径也会被删除
- 不会覆盖 target Agent 自己的 `config.json`，因此仍可保留 target 通过 `/api/agent/init` 生成的基础配置
- CLI 也提供同一套收口：`integration agent copy --source SOURCE --target TARGET`

## /api/agent/create

`GET /api/agent/create?agentId=xxx&name=yyy&type=zzz`

说明：

- `name` 表示 Agent workspace 内相对路径，支持 `docs/data`、`tmp/a/b` 这类带子目录的路径
- 每个路径段必须非空，且不能为 `.`、`..`
- 路径段中不允许包含空格和 `\:*?"<>|`
- 禁止绝对路径、`~`、`../` 或其他越界写入
- `type=0` 创建目录，`type=1` 创建文件；父目录不存在时会自动补齐
- 已存在、Agent 不存在、参数缺失等返回语义继续保持原样

## /api/restore

`POST /api/restore`

说明：

- 继续沿用现有 `/api/restore` 收口，不新增新的恢复接口；返回结构仍保持当前统一 `data[]` 记录数组格式
- 在原有 `chat_log` 恢复结果之外，接口会继续尝试合并同一 `agentId + chatId` 下的 CLI 事件日志，把 `cli/get`、`cli/pub` 一并返回给前端
- CLI 日志查询范围会与消息 restore 保持一致；如果本次请求带有 `lastId`，CLI 日志也会按同一增量边界续拉，避免前端重复消费
- 合并返回的 CLI 记录会保留 `id`、`agentId`、`chatId`、`content`、`logType`、`createdAt`，并通过 `role=cli/get`、`role=cli/pub` 明确标识类型
- `cli/get` 的 `content` 会保留原始任务载荷，兼容直接 `cmd` 字段、嵌套 `message`、`messages[].content` 等既有格式
- `cli/pub` 的 `content` 会保留原始执行结果，兼容纯文本输出与 JSON 包裹的消息结构；restore 链路不会额外发明 `cmd restore` 专用字段
- 最终返回结果会统一按 `createdAt` 升序排序；`createdAt` 相同时再按 `id` 升序排序，便于前端按单一时间线配对 `cli/get -> cli/pub`
- 如果当前环境下 CLI 事件日志查询失败，不会影响原有 `chat_log` restore 主流程；接口会按现有容错语义继续返回已拿到的消息记录
- 这次补充只覆盖 restore 对 CLI 子任务 `cmd` 的恢复能力，继续复用现有 CLI 日志写入链路和存储表，不新增新的日志表、消息表或额外后台任务

## /api/cmd

`POST /api/cmd`

请求体示例：

```json
{
  "agentId": "demo-agent",
  "chatId": "chat_001",
  "cmd": "pwd && ls",
  "timeout": 60000,
  "tid": "optional-task-id"
}
```

说明：

- 仅允许来自 `127.0.0.1`、`::1` 或 `localhost` 的请求执行
- `agentId`、`chatId`、`cmd` 为必填
- Agent 必须存在，已删除目录不会通过校验
- 若当前 `chatId` 会话未开启沙盒，则由当前进程 Shell 执行 `shell -c <cmd>`
- 若当前 `chatId` 会话已配置沙盒模式，则按当前系统改走对应 helper：
  - macOS：`integration.app/Contents/Helpers/<mode>/CLI_SANDBOX.app/Contents/MacOS/CLI_SANDBOX --cmd <cmd>`
  - WSL/Linux：`<integration-exec-dir>/helpers/<mode>/CLI_SANDBOX --cmd <cmd>`
- 若 `cli/get` 返回任务里带有 `subOps.exempted=true`，则即使当前会话已配置沙盒模式，也会直接走原始 Shell 执行链路
- 默认超时 `180000ms`，与 `cli-get` 一致
- 任何包含 `rm` 的命令都会被拒绝
- 执行记录写入共享 SQLite 的 `cmd_log`

## 会话沙盒

- 沙盒状态按 `chatId` 维度保存到共享 SQLite 的 `cli_sandbox_state`
- `chatId` 为空时，`/api/sandbox_status` 与 `/api/sandbox=*` 都会直接报错
- 仅支持 3 个有效模式：
  - `filepick`
  - `net`
  - `filepick_net`
- 读取当前会话沙盒模式：

```text
GET /api/sandbox_status?chatId=chat-001
```

- 读取接口只按 `chatId` 命中；即使请求里携带 `agentId`，也不会参与状态定位
- 写入当前会话沙盒模式：

```text
POST /api/sandbox=filepick?agentId=A&chatId=chat-001
POST /api/sandbox=net?agentId=A&chatId=chat-001
POST /api/sandbox=filepick_net?agentId=A&chatId=chat-001
POST /api/sandbox=filepick?agentId=A&chatId=chat-001&dir=%2FUsers%2Fme%2FDesktop
POST /api/sandbox=off?agentId=A&chatId=chat-001
```

- 写接口仍要求 `agentId` 与 `chatId`；其中 `agentId` 只用于日志，`chatId` 用于定位当前会话沙盒状态
- `filepick` / `filepick_net` 可选传入 `dir`，显式持久化当前 `chatId` 对应的 `allowed_dir`；未传时仍按当前系统走目录选择流程
- `allowed_dir` 表示当前会话的目录白名单路径；只有 `filepick` / `filepick_net` 且该值在 `trim` 后非空时，转发 `/v1/chat/completions`、上报 `cli/get`，以及 integration 内部 `memo`、`email`、`feishu` 等最终聊天请求才会附带顶层 `metadata.sandbox_path`
- `net`、`off`、无记录，或空白 `allowed_dir` 都不会上报 `metadata.sandbox_path`
- `off` 表示关闭沙盒，并直接删除该 `chatId` 的数据库记录
- CLI 也使用同一套协议：

```bash
./integration sandbox --agentId A --chatId chat-001
./integration sandbox --agentId A --chatId chat-001 --sandbox filepick_net
./integration sandbox --agentId A --chatId chat-001 --sandbox filepick --dir /Users/me/Desktop
./integration sandbox --agentId A --chatId chat-001 --sandbox off
```

## /api/kill

`POST /api/kill`

请求体示例：

```json
{
  "agentId": "demo-agent",
  "chatId": "chat_001",
  "cmd": "sleep 10",
  "tid": "optional-task-id"
}
```

说明：

- 仅允许来自 `127.0.0.1`、`::1` 或 `localhost` 的请求执行
- `agentId`、`chatId`、`cmd` 为必填
- Agent 必须存在
- 仅终止当前 Integration 进程内由 `/api/cmd` 启动且仍在运行的命令
- 优先按 `agentId + chatId + tid + cmd` 精确匹配；未提供 `tid` 时回退按 `agentId + chatId + cmd`
- 匹配不到活动命令时返回 `404`
- kill 日志写入共享 SQLite 的 `kill_log`

`kill_log` 结构：

```sql
CREATE TABLE kill_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  agent_id TEXT NOT NULL,
  chat_id TEXT NOT NULL,
  tid TEXT NOT NULL,
  cmd TEXT NOT NULL,
  received_at TEXT NOT NULL,
  completed_at TEXT NOT NULL DEFAULT ''
)
```

索引：

```sql
CREATE INDEX IF NOT EXISTS idx_kill_agent_chat_time
ON kill_log(agent_id, chat_id, received_at);
```

## Cron 集成

Integration 已内置 cron 的两部分能力：

- HTTP API：创建、查询、删除和更新 cron 任务
- 后台执行器：每分钟检查未来任务并执行到上游 `/v1/chat/completions`
- CLI 查询支持 `find-meta` 和 `find-detail`
- CLI 删除支持 `delete-meta` 和 `delete-detail`
- cron 的 `task_meta` / `task_detail` 数据库操作会同步写入 `cron_meta_log` / `cron_detail_log`
- `task_meta` / `task_detail` 新增 `type` 字段，默认 `cron`
- `task_detail` 主表索引为 `idx_detail_agent_chat_time_type(agent_id, chat_id, exec_time, task_type)`

## 模型密钥接口

### `/api/token`

Integration 与 proxy 保持一致，统一提供模型密钥读写接口。

`GET /api/token`

响应示例：

```json
{
  "status": 0,
  "models": {
    "openai": {
      "token": "Bearer sk-openai",
      "__url": "https://api.openai.example/v1",
      "__model": "gpt-4.1",
      "__model_fast": "gpt-4.1-mini",
      "__model_thinking": "o3",
      "__model_multi_input": "gpt-4.1-vision",
      "__model_multi_output": "gpt-image-1"
    },
    "kimi": {
      "token": "Bearer sk-kimi",
      "__url": "",
      "__model": "",
      "__model_fast": "",
      "__model_thinking": "",
      "__model_multi_input": "",
      "__model_multi_output": ""
    }
  },
  "updatedAt": {
    "openai": "2026-05-03T14:30:00+08:00",
    "kimi": "2026-05-03T14:30:00+08:00"
  }
}
```

`POST /api/token`

请求体示例：

```json
{
  "models": {
    "openai": {
      "token": "Bearer sk-openai",
      "__url": "https://api.openai.example/v1",
      "__model": "gpt-4.1",
      "__model_fast": "gpt-4.1-mini",
      "__model_thinking": "o3",
      "__model_multi_input": "gpt-4.1-vision",
      "__model_multi_output": "gpt-image-1"
    },
    "kimi": "Bearer sk-kimi"
  }
}
```

行为说明：

- 模型名称为唯一键
- `token` 为必填，`__url`、`__model`、`__model_fast`、`__model_thinking`、`__model_multi_input`、`__model_multi_output` 可为空
- 当请求来源不是 `localhost`、`127.0.0.1` 或 `::1` 时，接口返回中的每个 `token` 都会统一替换成 `**********`，包括 `GET /api/token` 和远程 `POST /api/token` 的响应体
- 这种远程读取掩码只影响接口返回值，不会改写 SQLite 中真实保存的密钥
- 当请求来源不是 `localhost`、`127.0.0.1` 或 `::1` 时，`POST /api/token` 不允许新增模型；只允许更新已存在的模型或删除已有模型
- 已存在模型会更新 token、扩展字段和最后更新时间
- 每次写入都会新增一条更新日志到 `proxy_agent_provider_log`
- 如果共享数据库中仍存在旧表 `token_store_log`，启动时会自动改名为 `proxy_agent_provider_log`
- 数据保存在共享 SQLite `data` 中的 `token_store` 和 `proxy_agent_provider_log` 表
- 模型真实密钥只保存在 SQLite 中；模型真正执行时仍按数据库中的真实 `token` 发起请求，不受远程读取掩码影响

也支持直接通过 CLI 读取：

```bash
./integration token
./integration token --provider deepseek
```

说明：

- `integration token` 会读取主应用 `config/config.json` 中的 `db` / `app-dir` 解析共享 SQLite `data`
- 不带 `--provider` 时输出按模型名排序后的 JSON 数组；每个模型值都是对象，包含 `token`、`__url`、`__model`、`__model_fast`、`__model_thinking`、`__model_multi_input`、`__model_multi_output`
- 带 `--provider` 时输出单个 JSON 对象；模型值同样为上述对象结构
- 如果指定模型不存在，则输出空对象 `{}`
- 当传入 `--agentId`、`--model`、`--function`、`--thinking`、`--input`、`--total`、`--cache` 时，会改为写入 Token 消费明细并返回 `{"status":0,"record":...}`
- 发布目录中的二进制同样要写成 `./integration token ...`；像 `/path/to/deepright/cli/module/release/integration --agentId ...` 这种不带 `token` 子命令的写法，会被当成服务启动参数并报错

## 插入待处理消息接口

### `/api/message_insert/add`

`POST /api/message_insert/add`

请求体示例：

```json
{
  "agentId": "agent-a",
  "chatId": "chat-001",
  "tid": 1718966400000,
  "message": "这是一条待插入的排队消息"
}
```

行为说明：

- 按 `chatId + tid` 作为唯一键写入共享 SQLite `data`
- 已存在同一条记录时，会直接覆盖 `agentId`、`message` 并把状态重置为 `0`
- 状态枚举固定为：
  - `0`：待上传
  - `1`：已上传
  - `2`：取消

### `/api/message_insert/del`

`POST /api/message_insert/del`

请求体示例：

```json
{
  "chatId": "chat-001",
  "tid": 1718966400000
}
```

行为说明：

- 不物理删除记录，而是把该条消息状态更新为 `2`
- 如果对应记录不存在，接口仍返回成功，但 `affected=false`

### `/api/message_insert/delete`

`POST /api/message_insert/delete`

请求体示例：

```json
{
  "chatId": "chat-001",
  "tids": [1718966400000, 1718966400001]
}
```

行为说明：

- 物理删除指定 `chatId` 下、状态仍为未终态的插入消息记录
- 这类删除不会把状态改成 `2`，而是直接从 `message_insert` 表移除
- 主要用于前端在“会话已结束，但仍残留旧的插入跟踪记录”时做恢复清理，防止后续再次切换会话时重复恢复
- 前端待发送队列中的消息如果正式轮到自动发送，或用户明确从队列里删除这条消息，也会先调用这个接口清掉对应 `tid`；只有接口返回成功，前端才会继续出队
- 如果目标记录不存在，接口仍返回成功，只是 `affected=0`

### `cli/pub` 插入消息上报

- `integration` 内部 `cli/get -> exec -> cli/pub` 链路在真正提交 `/cli/pub` 前，会先从本地 `message_insert` 表读取当前 `chatId` 下状态为 `0` 且尚未上报过的记录
- 单次最多附带 `5` 条，写入 `cli/pub` 请求体中的：

```json
{
  "insert": [
    { "tid": "1718966400000", "message": "..." }
  ]
}
```

- `/cli/pub` 返回成功后，这批 `tid` 只会标记为“已上报一次”，后续不再重复通过 `cli/get` 上报
- 只有当 integration 收到响应报文中 `metadata.__PROCESS__ = rag_insert` 且 `metadata.__TID__` 相同，这条消息才会自动更新为 `1`
- 如果上层在会话结束后决定不再把残留插入消息按“插入态”恢复，而是降级回普通待发送消息，应调用 `/api/message_insert/delete` 清掉这些旧记录，避免重复恢复
- 如果读取或回写状态失败，不会中断原有 `cli/pub` 主链路，但会在标准日志里记录错误

### 内置 CLI-Get 队列与重试

- `integration` 内置的 `cli-get` 当前实际为两段式流水线：
  - `cli/get -> taskQueue -> execute workers -> publishQueue -> cli/pub`
- 心跳线程不会因为执行 Worker 正忙而阻塞；只有当本地 `taskQueue` 已满时，才会暂停发新的 `/cli/get`
- 本地 `taskQueue` 是纯内存队列，不做持久化恢复
- 执行 Worker 在真正执行前会重新检查任务 `ddl`
- 如果当前时间已超过 `ddl`：
  - 任务会被直接丢弃
  - 不会执行命令
  - 不会提交 `/cli/pub`
  - 会输出包含 `tid` 的日志
- 执行结果会先进入 `publishQueue`，由独立发布 Worker 提交 `/cli/pub`
- `/cli/pub` 返回网络错误、超时、HTTP 非 `200`、或响应解析失败时，会按 `--retry_interval` 与 `--retry_times` 做重试
- 发布重试不会占住执行 Worker
- 发布重试允许同一 `tid` 因超时或网络异常而重复推送

### `/api/plugins/meta`

`GET /api/plugins/meta`

响应示例：

```json
{
  "status": 0,
  "data": [
    {
      "key": "feishu",
      "name": "飞书",
      "param": [
        {
          "appId": ""
        },
        {
          "appSecret": ""
        }
      ],
      "meta": {
        "appId": "cli-app",
        "appSecret": "cli-secret"
      }
    }
  ]
}
```

说明：

- 当请求来源不是 `localhost`、`127.0.0.1` 或 `::1` 时，该接口仍可读取，但会返回脱敏后的插件列表
- 远程响应会保留 `key`、`name`、`scope` 等基础信息，并清空 `param`、`meta`、`callback`、`agentId`、`chatId`、`model` 等运行期配置
- 该接口会把本地实时插件扫描结果与 `connect meta-list` 的已保存配置合并返回
- 不复用 `connect-cache` 的插件发现缓存；每次请求都会实时重新扫描插件并重新读取最新已保存 meta
- 默认扫描当前启动目录下的 `./plugins`，且只扫描当前层，不递归子目录
- 仅识别无后缀可执行程序，以及 `.py` / `.js` / `.go` 脚本文件
- 目录和不符合条件的文件会被直接跳过；单个候选文件探测失败时只记日志并继续处理其他插件
- `key` 字段优先取插件 `name` 输出中的 `key`；未显式提供时回退为文件名，脚本文件会自动去掉扩展名
- `meta` 字段为已配置参数，未配置时返回空对象 `{}`
- `router_disable` 字段来自已保存的 `connect_meta.router_disable`，未配置时默认为 `true`
- 返回结果可用于前端或其他模块动态展示可安装/可配置插件
- `site` 左上角即时通讯扇形菜单会直接消费该接口
- 当前接口只支持 `GET`

### `/api/plugins/config`

`POST /api/plugins/config?key=feishu&agentId=A&model=OpenAI`

请求示例：

```text
POST /api/plugins/config?key=feishu&meta=%7B%22appId%22%3A%22cli-app%22%2C%22appSecret%22%3A%22cli-secret%22%7D&stream=true&agentId=A&chatId=chat-001&model=OpenAI&thinking=true&router_disable=false
```

成功响应示例：

```json
{
  "status": 0,
  "data": {
    "id": 1,
    "name": "feishu",
    "meta": "{\"appId\":\"cli-app\",\"appSecret\":\"cli-secret\"}",
    "stream": true,
    "callback": "/abs/path/plugins/feishu",
    "agentId": "A",
    "chatId": "feishu",
    "model": "OpenAI",
    "thinking": true,
    "router_disable": false,
    "createdAt": "2026-05-05T11:40:00+08:00",
    "updatedAt": "2026-05-05T11:40:00+08:00"
  }
}
```

- `router_disable` 为可选布尔参数，默认 `true`
- 该值会写入共享 `connect_meta.router_disable`
- 当请求来源不是 `localhost`、`127.0.0.1` 或 `::1` 时，该接口会直接返回 `403`
- 对 `key=feishu`：
  - 未传 `chatId` 时，会自动落成固定值 `feishu`
  - 如果页面勾选“复用当前会话”，前端会显式传入当前会话的 `chatId`
  - `agentId` 始终以插件页面当前选中的 Agent 为准，勾选复用也不会自动切换或锁定 Agent

### `/api/plugins/status`

`GET /api/plugins/status?key=feishu`

说明：

- 当请求来源不是 `localhost`、`127.0.0.1` 或 `::1` 时，该接口仍可读取
- 使用插件生命周期链路当前的插件定位逻辑定位插件
- 默认按插件目录下的同名 `.pid` 文件判断插件进程是否仍在运行
- 也支持通过 `pid-file` 查询参数覆盖默认 PID 文件路径

响应示例：

```json
{
  "status": 0,
  "data": {
    "key": "feishu",
    "name": "飞书",
    "path": "/abs/path/plugins/feishu",
    "pid": 12345,
    "pidFile": "/abs/path/plugins/feishu.pid",
    "started": true
  }
}
```

失败响应示例：

```json
{
  "status": 1,
  "content": "agentId is required"
}
```

说明：

- `key` 必填，插件运行时主键，必须能匹配到插件 `name` 命令返回的 `key`
- `name` 仅保留兼容旧调用，展示名不能再作为运行时主键；新调用统一传 `key`
- `meta` 可选，配置表单 JSON 字符串；默认 `{}`，但必须是合法 JSON
- `stream` 可选，是否支持流式回复；默认 `false`
- `agentId` 必填，绑定的 AgentId
- `chatId` 可选，绑定的会话 ID
- `model` 必填，模型名称，且必须已经通过 `/api/token` 注册
- `thinking` 可选，是否深度思考；默认 `false`
- 当前接口只支持 `POST`
- 首次调用时使用 Connect 的 `meta-create --key <plugin-key>` 创建配置
- 当同 key 插件配置已存在时，会自动切换为 `meta-update`
- `callback` 不接受外部传入，始终自动解析为插件可执行文件的绝对路径
- 未传 `chatId` 时，会按插件自身规则归一化；当前 `feishu` 会固定写成 `feishu`
- 创建或更新失败时，会直接返回失败原因，例如缺少参数、插件不存在、Agent 不存在、模型未注册、`meta` 不是合法 JSON

### `/api/plugins/log`

`GET /api/plugins/log?key=feishu&last=10`

响应示例：

```text
event: log
data: 2026-05-05 10:00:00,收到消息 A

event: log
data: 2026-05-05 10:00:03,收到消息 B

event: error
data: log file not found: release/plugins/feishu.log
```

说明：

- `key` 必填，使用插件运行时主键
- `last` 可选，默认先补发最后 `10` 行
- 这是远程请求保留的插件读取接口之一；非 `localhost` / `127.0.0.1` / `::1` 访问不会被拦截
- 插件日志文件路径固定为 `release/plugins/插件名.log`
- 例如 `feishu` 固定读取 `release/plugins/feishu.log`，`email` 固定读取 `release/plugins/email.log`
- 不允许再根据当前工作目录、上级目录或其他候选目录推断日志路径
- 返回协议为 SSE，每一行日志对应一个 `event: log`
- 文件不存在或流式读取过程中被删除时，会先返回一条 `event: error`，然后关闭连接
- 当前接口只支持 `GET`
- 为兼容旧调用，当前也接受 `plugin` 查询参数，但建议统一改为 `key`

### `/api/plugins/start`

`POST /api/plugins/start?key=feishu`

响应示例：

```json
{
  "status": 0,
  "data": {
    "path": "/abs/path/plugins/feishu",
    "command": ["start", "--connect-bin", "./connect"],
    "output": {
      "status": "started",
      "pid": 56157
    }
  }
}
```

说明：

- 当请求来源不是 `localhost`、`127.0.0.1` 或 `::1` 时，该接口会直接返回 `403`

### `/api/plugins/stop`

`POST /api/plugins/stop?key=feishu`

说明：

- 当请求来源不是 `localhost`、`127.0.0.1` 或 `::1` 时，该接口会直接返回 `403`
- `key` 必填，使用插件运行时主键
- 除 `name` 之外的查询参数都会原样透传给插件
- 接口会优先执行 `<plugin> start|stop`，失败后自动兼容 `<plugin> --start|--stop`
- 两个接口都只支持 `POST`

### `/api/cron/create`

`POST /api/cron/create?agentId=xxx`

请求体字段：

| 字段 | 必填 | 说明 |
|------|------|------|
| `content` | 是 | 任务内容 |
| `model` | 是 | 模型名称 |
| `thinking` | 否 | 是否深度思考 |
| `router_disable` | 否 | 是否关闭 router，默认 `true` |
| `rawTime` | `cycle != -1` 时必填 | 首次开始时间，格式 `YYYY-MM-DD HH:MM` |
| `cycle` | 是 | `0=仅一次, 1=工作日, 2=自然日, 3=每小时, 4=每15分钟, 5=每30分钟, -1=自定义Cron` |
| `cron` | `cycle=-1` 时必填 | 自定义 Cron 表达式 |
| `chatId` | 否 | 绑定会话 ID |
| `type` | 否 | 任务类型，默认 `cron`；Connect 场景可传具体模块名，如 `feishu` |

周期任务示例：

```json
{
  "content": "每天上午生成巡检摘要",
  "model": "OpenAI",
  "thinking": true,
  "router_disable": false,
  "rawTime": "2026-05-04 09:30",
  "cycle": 1,
  "chatId": "chat-001"
}
```

高频任务示例：

```json
{
  "content": "每15分钟检查一次上游接口健康",
  "model": "OpenAI",
  "thinking": true,
  "router_disable": false,
  "rawTime": "2026-05-03 10:00",
  "cycle": 4,
  "chatId": "chat-001"
}
```

自定义 Cron 示例：

```json
{
  "content": "工作日中午整理异常日志",
  "model": "OpenAI",
  "thinking": true,
  "router_disable": false,
  "cycle": -1,
  "cron": "10 12 * * 1-5",
  "chatId": "chat-001"
}
```

成功响应：

```json
{
  "status": 0,
  "id": 1,
  "cron": "10 12 * * 1-5",
  "agentId": "demo-agent",
  "type": "cron"
}
```

### `/api/cron/detail/metadata`

`POST /api/cron/detail/metadata`

支持查询参数：

- `agentId` / `agent`
- `chatId` / `chat`
- `model`
- `content`
- `cycle`
- `type`
- `time` / `rawTime` / `date`
- `from` / `to`

返回示例：

```json
{
  "status": 0,
  "data": [
    {
      "id": 1,
      "cycle": 4,
      "rawTime": "2026-05-03 10:00",
      "agentId": "demo-agent",
      "type": "cron",
      "model": "OpenAI",
      "thinking": true,
      "router_disable": false,
      "cron": "*/15 * * * *",
      "content": "每15分钟检查一次上游接口健康",
      "chatId": "chat-001"
    }
  ]
}
```

说明：

- 返回的每条任务元数据都包含 `router_disable` 字段，表示该任务是否关闭 router；未设置时默认为 `true`

### `/api/cron/detail/list`

`POST /api/cron/detail/list`

支持查询参数：

- `metaId` / `meta`
- `agentId` / `agent`
- `chatId` / `chat`
- `model`
- `content`
- `cycle`
- `type`
- `time` / `date`
- `from` / `to`

说明：

- `metaId` 支持传 `1` 或 `cron_1`
- 未指定时间条件时，默认优先返回当前时间之后的明细；已保留的已完成明细也会返回

### `/api/cron/delete`

`POST /api/cron/delete`

支持查询参数：

- `id` / `metaId` / `meta`
- 或 `find-meta` 同款过滤条件

说明：

- 删除元数据时会同时删除其关联的“未完成”明细
- `started = 3` 的已完成明细会保留
- 删除操作会写入 `cron_meta_log` 与 `cron_detail_log`

### `/api/cron/detail/delete`

`POST /api/cron/detail/delete`

支持查询参数：

- `detailId` / `detail`
- `metaId` / `meta`
- 或 `find-detail` 同款过滤条件

说明：

- 删除时未指定时间条件，不会默认限制为未来数据

### 行为说明

- `/api/raw` 支持 Agent workspace 下的相对路径，也支持文件系统绝对路径（大小写不敏感匹配）；`~` 路径不支持
- `chatId` 会写入 `task_meta`，并在创建 `task_detail` 时继承下去
- `type` 会写入 `task_meta`，并在创建 `task_detail` 时继承下去；未传时默认 `cron`
- 如果某条待执行明细没有 `chatId`，执行器才会回退生成 `metaId@detailId`
- `cycle=1/2` 会在创建时补齐未来 5 天窗口的任务明细
- `cycle=3/4/5` 会在创建时按小时、15 分钟、30 分钟展开未来 5 天窗口的全部任务明细
- `cycle=-1` 时使用自定义 Cron 表达式，创建元数据时不立即展开首批明细，而由后台检查器持续补齐
- CLI 与 HTTP 创建任务复用同一套元数据写入与明细展开逻辑
- 创建时会检查指定 `agentId` 是否存在
- `--agent-dir` 既支持传 Agent 根目录，也支持直接传某个具体 Agent 目录
- 未显式传入的非 cron 模块必填参数会优先从主应用 `config/config.json` 读取，当前主要用于补全 `agent-dir` 与 `device`
- 创建时还会检查指定 `model` 是否已在 `/api/token` 注册，且 token 非空
- `task_meta` 与 `task_detail` 都不会保存模型 Token
- 后台执行器会在执行时根据 `model` 动态从 SQLite `token_store` 查询对应密钥，并设置到请求头 `Authorization`
- 任务元数据查询支持 `AgentId / ChatId / model / content / cycle / 开始执行时间范围`
- 任务元数据查询支持额外按 `type` 过滤
- 任务明细查询支持 `metaId / AgentId / ChatId / model / content / cycle / 执行时间范围`
- 任务明细查询支持额外按 `type` 过滤
- 任务元数据删除支持 `id` 或 `find-meta` 同款过滤条件
- 任务元数据删除同样支持复用 `type` 过滤条件
- 任务明细删除支持 `detailId`、`metaId` 或 `find-detail` 同款过滤条件
- 任务明细删除同样支持复用 `type` 过滤条件
- 删除 Agent 时，会同步删除该 Agent 关联的全部任务元数据和全部任务明细
- 每分钟执行前会先检查 Agent 是否已被删除，以及任务模型是否仍已配置密钥；不满足条件的待运行明细不会执行
- 每分钟检查任务元数据时，如果发现 Agent 已删除，或任务模型不存在/未填写密钥，会清理对应元数据和未完成明细，保留已完成明细
- 每分钟还会检查最近 24 小时内 `started = 3` 且 `task_type != cron` 的已完成明细，并尝试把任务最终文本结果通过对应插件自动回推给三方
- 如果 `task_detail.result_content` 是 ```json ... ``` 或 ``` ... ``` 包裹的 JSON object / array，integration 会先去掉 Markdown 外壳并标准化为紧凑 JSON，再传给插件 `send`
- 备忘录任务明细执行时，请求 `/v1/chat/completions` 的 metadata 会附带 `cron_type`
- 普通周期任务写入 `cron_type=cron`
- 插件桥接生成的任务写入对应插件 `key`，例如 `cron_type=feishu`
- 自动回推只会处理 connect 原始消息状态仍为“已启动”的记录；成功后当前消息会更新为“已回复”，更早且仍为“已启动”的同插件消息会更新为“已完成”
- 未指定查询维度表示该维度全部匹配
- 任务明细未指定时间条件时，默认优先返回当前时间之后的数据；已保留的已完成明细也会返回
- 删除任务明细时，如果未指定时间条件，不会默认限制为未来数据

共用的数据表包括：

- `task_meta`
- `task_detail`
- `proxy_agent_provider_log`
- `cron_meta_log`
- `cron_detail_log`
- `chat_log`
- `cmd_log`
- `kill_log`

## 注意事项

- 各模块共享同一份 Agent 元数据缓存
- 共享的 Agent 元数据新增 `git` 字段，表示本机 git 可执行文件绝对路径；macOS/Linux 与 Windows 分别按系统方式探测，失败时返回空字符串
- proxy 和 static 共用同一个 HTTP 端口和路由
- cli-get 以后台 goroutine 运行，不影响 HTTP 服务
- cli-get 后台链路当前实际为“心跳线程 + 本地任务队列 + 执行 Worker + 本地发布队列 + 发布重试”
- cron 也以后台 goroutine 运行，默认每分钟检查和执行一次
- 相同名称的命令行参数（host、device、agent-dir、agent-cache）在所有模块间共享
- 会话日志与恢复接口沿用 proxy 语义：Q 单条写入，A 按 SSE 增量块实时落库，X 表示主动取消；同时记录 `chatType`（用户会话/定时任务）和 `responseType`（正常/异常）
- `integration` 默认不带子命令时仍然启动统一 HTTP 服务
- `integration cron create` / `integration cron create-cron` / `integration cron find-meta` / `integration cron find-detail` / `integration cron delete-meta` / `integration cron delete-detail` 同时兼容 `--key=value` 与 `--key value` 两种传参方式
- HTTP 服务模式下，`--agent-dir` 默认取当前目录下的 `./agent`，不存在时会自动创建；`--site` 默认取当前目录下的 `./site`
- CLI 创建时，Agent 根目录优先取 `--agent-dir`，其次取主应用 `config/config.json` 中的 `agent-dir`，再次取环境变量 `AGENT_DIR`，最后回退当前目录下的 `./agent`
## 查看 SKILL 解析告警

HTTP:

```bash
curl http://127.0.0.1:8080/skills_warning
curl http://127.0.0.1:8080/skills_warning?refresh=1
```

CLI:

```bash
./integration skills-warning
./integration skills-warning --refresh
```

说明：

- `/skills_warning` 与 `integration skills-warning` 都读取共享 sqlite `data` 中的 `skills_warning` 表
- `refresh=1` 或 `--refresh` 会先立即重新扫描，再返回最新结果
- 默认扫描根目录使用当前运行配置里的 `agent-dir/skills`
- 返回结构中的 `path`、`reason`、`time` 分别代表错误文件绝对路径、解析失败原因、最近一次扫描时间
- 文件修复后，对应告警会在下一轮自动同步时删除

## 20260524-1 更新

### /api/agent/create name语义升级
- `name` 升级为 workspace 内相对路径，支持 `docs/data`、`tmp/a/b` 格式
- 每个路径段必须非空、不能为 `.` 或 `..`、不能包含空格和 `\:*?"<>|`
- 禁止绝对路径、`~`、`../` 或其他越界写入
- `type=0` 创建目录，`type=1` 创建文件，父目录不存在自动补齐
- 收口 Proxy 和 Site 两侧行为一致

## 20260524-2 更新

### /api/agent/init 默认模板初始化
- `/api/agent/init` 改为从 `--default-dir` 复制默认模板初始化新 Agent
- 未传 `--default-dir` 时，默认使用应用启动目录下的 `config/`
- `module/build.sh` 会把 `module/config` 一并打包到交付物 `config/`；其中 `config.json` 仅供主应用读取，新 Agent 会改为生成空的 `config.json`
- 复制失败时会回滚新建 Agent 目录，避免留下半初始化目录

## 20260525-1 更新

### 空 Agent 根目录启动补齐
- 启动时如果 `--agent-dir` 指向空目录，会把 `--default-dir` 中除根 `config.json` 外的内容复制到 `DEF_AGENT/`，并生成空的 `DEF_AGENT/config.json`
- 复制完成后仍会确保 `DEF_AGENT/skills` 存在，和 `/api/agent/init` 使用同一套默认模板来源
- `default-dir` 缺失、不是目录或复制失败时，启动会直接报错，避免生成半初始化的默认 Agent
# 2026-05-24 更新

- Integration 模块现在统一使用 `router_disable` 作为 SWARM 对应的规范字段。
- `router_disable=true` 表示关闭，`router_disable=false` 表示开启。
- UI 上仍显示 `SWARM` 这个开关名，但 CLI 与接口统一使用 `router_disable`。
- `/api/cron/create`、插件配置链路、插件元数据返回、以及桥接生成的定时任务都已切到 `router_disable`。
- 历史 `swarm` 字段只作为旧库迁移来源，不再作为 Integration 的新入参。

---

## 迭代 20260603-1：插件文件类型识别收口

- `GET /api/plugins/meta` 改为在 `integration` 模块内直接扫描插件目录
- 顶层 CLI `./integration list-plugins` 已同步使用同一套本地扫描规则
- 候选文件：无后缀且可执行的程序，后缀为 `.py`、`.js`、`.go` 的脚本文件
- 目录、隐藏文件和其他无关文件会被直接跳过
- 单个候选文件失败时只输出日志并跳过，不会让接口报错或崩溃

请求：

```text
GET /api/plugins/meta
```

CLI：

```bash
./integration list-plugins
```

## 迭代 20260610-2：插件参数结构收口

- `GET /api/plugins/meta` 中每个插件的 `param` 已统一改为对象数组格式
- 顶层 CLI `./integration list-plugins` 同步输出相同结构
- 新格式示例：`[{"appId":""},{"appSecret":""}]`
- 对象中的 key 是参数名，value 是占位提示；未提供提示时返回空字符串
- `integration/USER_GUIDE.md` 与本次迭代手册已同步更新

## 迭代 20260605-1：服务启动自动打开浏览器

- integration 服务启动成功后，会在后台延迟约 200ms 异步打开浏览器，不阻塞主服务
- 自动打开地址统一为 `http://localhost:<port>/site/#app`
- 自动打开浏览器时，会优先按操作系统规则查找指定浏览器，并附带最大化参数启动；若无命中则回退到系统默认浏览器
- 浏览器打开失败只记日志，不会让 integration 启动失败

### macOS

- 查找顺序：Google Chrome → Google Chrome for Testing → Microsoft Edge → Brave Browser → Chromium → 回退 `open`

### Linux

- 查找顺序：google-chrome → google-chrome-stable → chromium-browser → chromium → microsoft-edge → microsoft-edge-stable → brave-browser → 回退 `xdg-open`

### Windows（含 WSL）

- 优先从常见安装目录查找 Chrome、Edge、Chromium；若未命中，再从 PATH 中查找；仍不命中回退 `cmd /c start /max`

## 迭代 20260606-1：插件远程执行接口收口

- 新增 `GET /api/plugins/exec`
- 新增 `integration plugins exec` CLI 子命令
- `command` 支持多级子命令文本，例如 `instance init`
- 除 `key`、`command` 之外的 query / CLI 参数都会转成 `--flag value` 透传给插件

请求：

```text
GET /api/plugins/exec?key=browser&command=instance%20init&agentId=A&chatId=chat-001
```

- `key` 必填，`command` 必填，`command` 里的空格需要按 URL 规则转义

CLI：

```bash
./integration plugins exec --key browser --command 'instance init' --agentId A --chatId chat-001
```

## 迭代 20260607-3：macOS 固定运行目录

- macOS 下共享运行时目录固定为 `~/Library/Application Support/deepright`
- 共享 sqlite 固定读写 `~/Library/Application Support/deepright/data`
- `knowledge` 目录固定为 `~/Library/Application Support/deepright/knowledge`
- `--agent-dir` 默认固定为 `~/Library/Application Support/deepright/agent`
- 如果 `integration` 以 `.app` 形式运行，插件固定从 `integration.app/Contents/Resources/plugins` 读取
- `config`、`site` 等资源目录也会优先按 `.app` 的 `Contents/Resources` 解析，避免误读 `Contents/MacOS`

---

## 迭代 20260614-1：Agent 导入与导出能力

本次迭代为 `integration` 新增了 Agent 导入与导出能力，便于备份单个 Agent 或在机器间迁移。

## HTTP 接口

- `GET /api/agent/export?agent_id=xxx`
- 返回一个 zip 文件
- zip 内会保留顶层 Agent 目录
- 导出时会过滤该 Agent 一级目录中的 `chrome*`、`data`、`tmp`

- `POST /api/agent/import`
- 支持导入 export 生成的 zip，或导入一个完整 Agent 目录结构
- 如果待导入 Agent 与现有目录重名，会直接拒绝导入，并提示先删除同名 Agent
- zip 导入会先解压，再完成导入，并清理临时 zip 文件

## CLI

- `integration agent export --agent DEF_AGENT --output ./DEF_AGENT.zip`
- `integration agent import --input ./DEF_AGENT.zip`
- `integration agent import --input /path/to/DEF_AGENT`

## 说明

- 导入成功后会刷新 Agent 元数据缓存
- 这次变更不修改现有 Agent 初始化、删除、配置保存等行为

---

## 迭代 20260614-2：/install_app 区分操作系统与已安装检测

本轮迭代把 `install_app` 的配置来源升级为主应用 `config/config.json` 的按操作系统结构，同时保持 `--install_app` 仍然作为额外追加项。

适用规则：

- Linux 读取 `install_app.linux`
- macOS 读取 `install_app.mac`
- Windows 和 WSL 读取 `install_app.wsl`
- `--install_app` 依旧使用逗号分隔字符串，并与当前系统配置、自动探测结果统一去重合并
- 每个 `install_app` 元素都表示一个本地应用名；当前系统如果已安装该应用，就不会出现在 `/install_app` 返回中
- 应用安装状态会缓存 5 分钟

`config/config.json` 示例：

```json
{
  "install_app": {
    "linux": ["node", "python"],
    "wsl": ["node", "python", "docker"],
    "mac": ["node", "python", "xcode-select"]
  }
}
```

示例：

```bash
./integration --install_app git,python3
curl http://127.0.0.1:8080/install_app
```

接口返回会合并：

- 当前机器自动探测缺失的 `git`、`python3`
- `config/config.json` 中当前操作系统对应的数组
- `--install_app` 传入的额外条目

---

## 迭代 20260614-3：Token 消费记录查询

本次迭代为 `integration token` 增加了本地 Token 用量查询能力，并保持原有模型密钥读取与消费写入模式兼容。

## 查询入口

- 兼容原需求中的顶层写法：

```bash
integration token --n 500
integration token --n 500 --start "2026-06-14 12:00:00" --close "2026-06-14 14:00:00"
```

- 新增独立子命令：

```bash
integration token get --n 500
integration token get --n 500 --start "2026-06-14 12:00:00" --close "2026-06-14 14:00:00"
integration token get --help
```

- 可选参数：
  - `--agentId` / `--agent`：仅查询指定 AgentId
  - `--n`：最近 N 条，默认 `500`
  - `--start`：开始时间
  - `--close`：结束时间，默认当前时间

## 数据来源

- 查询读取的是本地 SQLite 中 `token_consume_log` 表里的 Token 用量数据
- 聚合与过滤规则复用 `/api/consume` 的同一套底层查询逻辑
- CLI 在“最近 N 条”模式下会先取最新记录，再按时间升序输出，便于直接查看时间线

## 时间格式

- `--start` 与 `--close` 支持：
  - `yyyyMMdd-hhmmss`
  - `YYYY-MM-DD HH:MM:SS`

示例：

```bash
integration token get --n 100 --start "20260614-120000" --close "20260614-140000"
integration token get --n 100 --start "2026-06-14 12:00:00" --close "2026-06-14 14:00:00"
```

## 输出结构

- 查询输出与 HTTP `/api/consume` 一致：
  - `status`
  - `details`
  - `summary`

示例输出：

```json
{
  "status": 0,
  "details": [
    {
      "thinking": 12,
      "input": 24,
      "total": 36,
      "cache": 6,
      "model": "deepseek-chat",
      "agentId": "demo-agent",
      "function": "cli/get",
      "timestamp": 1781409720000
    }
  ],
  "summary": [
    {
      "model": "deepseek-chat",
      "thinking": 12,
      "input": 24,
      "total": 36,
      "cache": 6
    }
  ]
}
```

## 兼容性

- 以下旧命令保持不变：

```bash
integration token
integration token --provider deepseek
integration token --agentId demo-agent --model deepseek-chat --function cli/get --thinking 10 --input 20 --total 30 --cache 5
```

- 只有传入 `--n`、`--start`、`--close`，或显式使用 `token get` 时，才进入本地 token 用量查询模式

---

## 迭代 20260614-4：技能动态注入收口

## 本次更新

- `GET /api/skills?agentId=xxx` 仍然先返回 Agent 自身的技能名
- 主应用 `config/config.json` 中新增的 `skills` 数组会按顺序追加到返回结果
- `__internal_cron` 不再由接口硬编码追加，改为完全由主应用 `config/config.json.skills` 控制
- 当 `browser` 插件处于开启状态时，接口会追加 `__internal_browser`
- 当 `remote` 插件处于开启状态时，接口会追加 `__internal_remote`
- 返回结果会自动去重，保留首次出现的顺序

## 配置方式

主应用 `config/config.json` 示例：

```json
{
  "skills": [
    "__internal_cron",
    "__internal_demo"
  ]
}
```

## HTTP 用法

请求：

```text
GET /api/skills?agentId=A
```

返回示例：

```json
[
  "__internal_F",
  "__internal_cron",
  "__internal_demo",
  "__internal_browser",
  "__internal_remote"
]
```

说明：

- `agentId` 仍为必填
- `config/config.json.skills` 中声明了什么，接口就追加什么
- `browser`、`remote` 两个内部技能只会在对应插件实际开启时追加
- 如果 `skills`、Agent 自身技能、插件内部技能之间出现重名，只保留一份

## 同步结果

- `integration/main.go` 已改为本地组装 `/api/skills` 返回结果
- `integration/main_test.go` 已改为覆盖 config 驱动技能和运行中插件技能场景
- 本迭代手册对应当前目录下的 `REQUIREMENT.md`

---

## 迭代 20260615-1：HTTP 转发超时控制

## 本次更新

- `cli-get` 默认 HTTP 总超时从 `60000ms` 调整为 `45000ms`
- 主应用 `config/config.json` 新增 `http` 配置块，可集中配置：
  - `http_connect_timeout`
  - `http_socket_timeout`
  - `http_timeout`
  - `debug`
- 所有运行态启动配置都统一收口到主应用 `config/config.json` 的同名字段
- 以上 HTTP 配置只从主应用 `config/config.json.http` 读取，不再兼容旧的平铺写法，也不再从其他文件回退
- `http.debug=true` 时，会把 `cli/get` / `cli/pub` 明细日志写入 `integration` 标准日志

## 配置示例

```json
{
  "http": {
    "http_connect_timeout": 15000,
    "http_socket_timeout": 45000,
    "http_timeout": 45000,
    "debug": true
  }
}
```

## 明细日志口径

- `cli/get` 请求远程主机超时时间
  - 同时附带本次耗时和当前 HTTP 超时配置，便于区分连接超时、响应头等待超时和总超时
- `cli/get` 返回待执行任务时的原始报文、时间
- `cli/pub` 回传执行结果时的状态、结果、时间

## 同步结果

- `integration/main.go` 已支持从 `config/config.json.http` 读取 `cli-get` HTTP 配置
- `integration/main.go` 已支持 `http.debug` 详细日志
- `integration/main_test.go` 已补充嵌套 `http` 配置和详细日志测试

---

## 迭代 20260615-2：新建 Agent 默认配置与打包逻辑

## 本次更新

- 主应用目录下的 `config/` 现在拆分为两类用途
- `config/config.json` 只用于 `integration` 主应用启动配置
- `config/` 下除 `config.json` 外的模板文件和目录，例如 `SOUL.md`、`USER.md`、`skills/`，用于创建新 Agent 或补齐空 `agent-dir` 时复制到 Agent 工作目录
- 打包脚本会继续把整个 `config/` 目录打包到交付物中，确保主应用配置和 Agent 模板都随应用一起发布
- 新建 Agent 后会单独初始化一个空的 `config.json`，不会继承主应用 `config/config.json` 的内容

## 目录语义

主应用目录示例：

```text
config/
├── config.json
├── SOUL.md
├── USER.md
└── skills/
```

语义说明：

- `config/config.json`
  - 给 `integration` 主程序读取启动参数
  - 打包后保留在应用目录，或 macOS `.app` 的 `Contents/Resources/config/config.json`
- `config/` 中其他文件
  - 作为默认 Agent 模板
  - 在 `GET /api/agent/init?name=...` 或启动时自动补齐 `DEF_AGENT` 时复制到 Agent 目录

## 行为变化

- 创建新 Agent 时，不再把主应用 `config/config.json` 复制进 Agent 工作目录
- 新 Agent 的 `config.json` 会固定初始化为：

```json
{}
```

- 如果 `agent-dir` 为空，启动时自动创建的 `DEF_AGENT/` 也遵循同样规则
- `DEF_AGENT/skills` 仍会自动补齐

## 打包说明

- `build.sh` 会把 `module/config` 整体复制到 release 目录下的 `config/`
- macOS `.app` 打包时，也会把 release 中的 `config/` 复制到 `Contents/Resources/config/`
- 因此同一份 `config/` 同时承载：
  - 主应用启动配置入口 `config/config.json`
  - 默认 Agent 模板资源

## 使用示例

主应用 `config/config.json` 示例：

```json
{
  "host": "https://www.deepright.cn",
  "agentDir": "agent",
  "default_dir": "config",
  "site": "site"
}
```

当调用：

```bash
./integration
```

会发生：

- 主程序从 `config/config.json` 读取启动参数
- 如果 `agent-dir` 为空，会用 `config/` 中除 `config.json` 外的内容创建 `DEF_AGENT/`
- `DEF_AGENT/config.json` 单独写入空对象 `{}`，不会复制主应用配置

## 同步结果

- `integration/main.go` 已区分主应用 `config/config.json` 和 Agent 模板复制逻辑
- `integration/main_test.go` 已覆盖新建 Agent、补齐 `DEF_AGENT` 时不继承主应用 `config.json` 的场景
- `build.sh` 已保持 `config/` 目录进入 release 和 macOS `.app` 资源目录

---

## 迭代 20260615-3：插件 MD5 更新校验

## 变更说明

- `integration.app` 双击启动时，会先检查安装包内 `plugins/` 与运行时 `~/Library/Application Support/deepright/plugins/` 的插件二进制是否一致
- 只有插件 `MD5` 不一致时，才会更新运行时插件
- 如果检测到插件需要更新，但当前 `8080` 端口已被占用，则不会覆盖运行中的插件二进制，而是弹窗提示“有插件需要更新，请重启应用”
- 如果检测到插件需要更新，且当前未启动，则会先完成插件同步，再继续启动应用
- 插件更新使用“临时文件 + rename”的原子替换方式，避免运行中可执行文件被直接覆盖

## CLI

新增命令：

```bash
integration plugins sync-bundled --check
integration plugins sync-bundled
```

说明：

- `--check`：只检查安装包插件与运行时插件是否一致，不执行同步
- 不带 `--check`：同步所有 `MD5` 不一致的插件二进制

返回示例：

```json
{
  "status": 0,
  "data": {
    "bundledPluginDir": "/Applications/DeepRight.app/Contents/Resources/plugins",
    "runtimePluginDir": "/Users/demo/Library/Application Support/deepright/plugins",
    "needsUpdate": true,
    "pending": [
      {
        "name": "browser",
        "sourcePath": "/Applications/DeepRight.app/Contents/Resources/plugins/browser",
        "targetPath": "/Users/demo/Library/Application Support/deepright/plugins/browser",
        "sourceMD5": "xxx",
        "targetMD5": "yyy"
      }
    ],
    "updated": [],
    "checkOnly": true
  }
}
```

## 启动行为

- 双击 `integration.app` 时，如果本机已有运行中的 Integration，并且检测到插件需要更新：
  - 弹窗提醒重启应用
  - 不覆盖运行时插件
  - 仍会打开 `http://localhost:8080/site/#app`
- 双击 `integration.app` 时，如果当前未启动，并且检测到插件需要更新：
  - 先同步插件
  - 再启动 Integration
  - 最后打开 `http://localhost:8080/site/#app`

---

## 迭代 20260615-4：Windows WSL 默认路径规范

本次迭代为 `integration` 补充了 Windows WSL 场景下的统一默认运行目录，避免 `agent`、`plugins`、`knowledge`、`integration.pid`、`integration.log` 分散落在当前启动目录。

## 默认路径

- `--agent-dir` 默认改为 `~/deepright/agent`
- 插件运行目录固定为 `~/deepright/plugins`
- 知识库目录固定为 `~/deepright/knowledge`
- `integration.pid` 固定写入 `~/deepright/integration.pid`
- `integration.log` 固定写入 `~/deepright/integration.log`

## 自动创建

- 当 `~/deepright` 不存在时，启动阶段会自动创建
- 当 `~/deepright/plugins` 不存在时，运行时会自动创建
- 当 `~/deepright/agent` 不存在时，启动阶段会自动创建
- 当知识库运行时初始化时，`~/deepright/knowledge` 也会自动创建

## 说明

- 该行为仅在 `integration` 运行于 Windows WSL 时生效
- macOS 仍保持 `~/Library/Application Support/deepright`
- 普通 Linux 仍保持当前应用目录下的相对路径默认值

---

## 迭代 20260618-1：沙盒模式与系统隔离

## 本次更新

- 会话沙盒继续保持 3 种模式：
  - `filepick`
  - `net`
  - `filepick_net`
- macOS 仍然走原有 `CLI_SANDBOX.app` 路径，不改现有 bundle 结构
- WSL/Linux 新增独立 `bubblewrap` helper，发布后位于 `helpers/<mode>/CLI_SANDBOX`
- `/api/sandbox=*`、`/api/sandbox_status`、`integration sandbox` CLI 的协议不变
- `integration /api/cmd` 与内部 `cli/get -> exec -> cli/pub` 链路都会按当前系统自动选择对应沙盒 helper

## 运行路径

- macOS：
  - `integration.app/Contents/Helpers/<mode>/CLI_SANDBOX.app/Contents/MacOS/CLI_SANDBOX`
- WSL/Linux：
  - `<integration-exec-dir>/helpers/<mode>/CLI_SANDBOX`

## 同步结果

- `integration/main.go` 已支持双平台 helper 路径解析
- `cli/module/build.sh` 会在 linux 发布物里打包 WSL helper
- `integration/main_test.go` 已补充 linux `helpers/<mode>/CLI_SANDBOX` 解析测试

---

## 迭代 20260704-1：会话级技能目录禁用

## 本次更新

- `integration` 新增了虚拟文件系统技能目录的会话级禁用能力
- 技能目录以“直属包含 `SKILL.md` 的目录”为单位；禁用后，该目录及其全部子孙技能都会在当前 `chatId` 下失效
- 这组状态会持久化到共享 sqlite；刷新页面或重新进入同一会话后仍然生效，但不会影响其他会话
- `GET /api/skills?agentId=xxx&chatId=yyy` 现在会按当前会话的禁用状态过滤返回结果；不传 `chatId` 时保持原有行为
- `GET /api/files?path=xxx&chatId=yyy` 现在会额外返回技能目录状态字段，供前端渲染禁用、继承禁用和普通态
- `POST /api/skill_state` 用于切换某个技能目录在当前会话中的禁用/恢复状态

## `/api/files`

请求示例：

```text
GET /api/files?path=/Users/demo/agent/A/skills&chatId=chat-1
```

目录项会新增以下字段：

- `hasSkill`：当前目录直属是否存在 `SKILL.md`
- `skillDisabled`：当前目录是否处于禁用态
- `skillDisabledSelf`：当前目录是否由自身路径直接禁用
- `skillDisabledInherited`：当前目录是否因父级技能目录被禁用而继承禁用

返回示例：

```json
[
  {
    "name": "alpha",
    "type": "dir",
    "hasSkill": true,
    "skillDisabled": true,
    "skillDisabledSelf": true,
    "skillDisabledInherited": false
  }
]
```

## `/api/skill_state`

请求示例：

```json
{
  "chatId": "chat-1",
  "path": "/Users/demo/agent/A/skills/alpha",
  "disabled": true
}
```

返回示例：

```json
{
  "status": 0,
  "chatId": "chat-1",
  "path": "/Users/demo/agent/A/skills/alpha",
  "disabled": true,
  "disabledSelf": true,
  "disabledInherited": false,
  "disabledPaths": [
    "/Users/demo/agent/A/skills/alpha"
  ]
}
```

说明：

- `path` 必须是一个真实存在、且直属包含 `SKILL.md` 的目录
- 如果父级目录已经禁用，子级目录不会重复写入 `disabledPaths`
- 如果恢复父级目录，原本只是继承禁用的子级目录会自动恢复为普通态

## `/api/skills`

- `GET /api/skills?agentId=xxx&chatId=yyy` 会先取当前 Agent 原本可见的技能，再按 `chatId` 绑定的禁用目录过滤
- 被命中的技能目录以及其子孙技能都不会继续出现在返回数组中
- 不传 `chatId` 时，接口仍返回完整技能列表，兼容旧调用

---

## 迭代 20260705-1：restore 恢复 CLI 子任务历史

## 本次更新

- `/api/restore` 在原有消息恢复之外，会继续合并当前会话下的 `cli/get`、`cli/pub` 日志，供 site 重建右侧 `CMD` 子任务历史
- 返回结构仍保持现有统一 `data[]` 数组格式，不新增新的 restore 接口，也不新增额外的 `cmd restore` 专用字段
- 返回给前端的 CLI 记录会保留原始 `content`，兼容 `cmd`、`message`、`messages[].content` 等既有日志结构
- 合并后的消息记录与 CLI 事件会统一按 `createdAt`、`id` 升序排序，便于前端按单一时间线恢复
- 如果 CLI 事件日志查询失败，接口会继续按现有容错语义返回已拿到的消息记录，不影响原有 `chat_log` restore 主流程

## 相关需求目录

- Integration 主需求：`/path/to/deepright/cli/module/integration/REQUIREMENT.md`
- Integration 手册：`/path/to/deepright/cli/module/integration/USER_GUIDE.md`
- restore 恢复 CLI 子任务历史：`/path/to/deepright/cli/module/integration/iteration/20260705-1/REQUIREMENT.md`

---

## 迭代 20260707-1：会话沙盒改为按 `chatId` 命中

## 本次更新

- 会话沙盒状态从 `agentId + chatId` 改为仅按 `chatId` 保存与命中
- `/api/sandbox_status` 改为只依赖 `chatId`；即使请求里携带 `agentId`，也不会参与状态定位
- `/api/sandbox=*` 写接口仍要求 `agentId` 与 `chatId`；其中 `agentId` 仅用于日志，`chatId` 用于写入当前会话沙盒状态
- `metadata.agent.sandbox` 与 `metadata.agents[].sandbox` 都改为按当前 `chatId` 实时读取共享 sqlite
- `/api/cmd` 与内部 `cli/get -> exec -> cli/pub` 链路的沙盒命中都改为只看 `chatId`
- 跨系统 helper 选择保持不变：macOS 继续走 `CLI_SANDBOX.app`，WSL/Linux 继续走 `helpers/<mode>/CLI_SANDBOX`

## 行为说明

- `chatId` 为空时，读写都会直接报错
- `off` 会删除当前 `chatId` 的记录；无记录视为 `off`
- `filepick` / `filepick_net` 如显式传入 `dir`，会把 `allowed_dir` 按当前 `chatId` 持久化；未传时继续按当前系统走目录选择流程
- 写入日志会输出 `agentId`、`chatId` 以及 `from -> to` 的文本变更信息

---

## 迭代 20260707-2：顶层 `metadata.sandbox_path` 收口

## 本次更新

- `integration` 在转发 `/v1/chat/completions`、上报 `/cli/get`，以及内部 `memo`、`email`、`feishu` 等最终转发到上游 `/v1/chat/completions` 的请求前，会统一按当前 `chatId` 注入顶层 `metadata.sandbox_path`
- `sandbox_path` 与 `knowledge` 同层，只表示当前会话目录白名单路径；不替代 `metadata.agent.sandbox` 或 `metadata.agents[].sandbox` 的模式语义
- 字段值固定来自共享 sqlite 中当前会话的 `allowed_dir`；读取后会先做 `trim`，并按最终生效的 `chatId` 重新计算，不信任外部请求体手工传入的旧值
- 只有 `filepick` / `filepick_net` 且 `allowed_dir` 非空时才会上报；`net`、`off`、无记录、空字符串都会直接不传该字段，并清理残留旧值

## 行为说明

- `sandbox_path` 是会话维度字段，不与 `agentId` 绑定；同一 `chatId` 下切换不同 Agent 时，只要会话目录未变，最终值保持一致
- 该字段只新增在顶层 `metadata`；现有 `metadata.agent.sandbox`、`metadata.agents[].sandbox` 等字段继续保留，避免破坏兼容性
- 外部请求体即使显式传入错误的 `metadata.sandbox_path`，integration 也会在最终转发前按当前会话真实状态覆盖
