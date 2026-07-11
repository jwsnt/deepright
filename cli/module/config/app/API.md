# Integration API

`integration` 默认对外提供 HTTP 服务，默认地址为 `http://127.0.0.1:8080`。

主要接口统一提供 CLI 收口：

- 主入口：`integration api ...`
- 兼容别名：`integration service ...`
- 已有成熟本地 CLI 的能力优先复用原 CLI；其余命令通过 integration HTTP 服务调用
- `/api/shutdown` 只保留 HTTP 文档，不提供 `integration api shutdown` 包装

## CORS

服务端当前统一允许跨域访问，并为预检请求返回：

- `Access-Control-Allow-Origin: *`
- `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, HEAD, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type, Authorization, Accept, Origin, X-Requested-With`
- `Access-Control-Expose-Headers: Content-Type, Content-Length, Content-Disposition`

## CLI 约定

通用参数：

- `--addr URL`：指定 integration HTTP base address
- `--port 8080`：指定 integration 端口，当前仅支持 `8080`
- `--output PATH`：把响应体保存到文件

常用示例：

```bash
integration api heartbeat
integration api agent-id
integration api edit --agentId demo --path USER.md --content '# hello'
integration service cancel --chat chat-001
```

## 本地页面 URL 约定

适用场景：site 侧 assistant URL 小气泡的 `预览` / `打开`，以及任何主动探测或跳转到本地页面的场景。

当目标 URL 的 host 为 `localhost` 或 `127.0.0.1` 时，建议把上下文放进目标 URL 自身：

| 参数 | 用途 |
|---|---|
| `agentId` | 当前 Agent ID，供目标本地页面恢复 Agent 上下文 |
| `chatId` | 当前会话 ID，供目标本地页面恢复会话上下文 |
| `themeLabel` | 当前主题标签；深色传 `深色模式`，浅色传 `浅色模式` |

说明：

- 这三个参数不是 integration 服务端统一强制消费的公共 query；integration 通常只透传、打开或探测最终 URL。
- 真正消费这些参数的是目标本地页面。
- 原 URL 已有 query 时继续追加；若已存在同名键，建议按当前上下文覆盖。

## `cli/module/site/index.html` 主题设计说明

`themeLabel` 主要用于把当前站点主题透传给外部本地页面；DeepRight 自身首页 `cli/module/site/index.html` 的主题切换，当前由页面内状态驱动，而不是直接读取该 query。

当前实现约定：

- 主题状态保存在 `localStorage["theme"]`
- 页面根节点通过 `<html data-theme="light|dark">` 控制整套 CSS 变量
- 手动切换入口为页面主题按钮，对应 `toggleTheme()`
- 未手动覆盖时会自动切换主题：`07:00-18:59` 默认浅色，`19:00-06:59` 默认深色
- 若自动主题被禁用（例如特定低性能画像），则保持当前主题，不按时间自动切换

### 深浅模式的配色与设计风格

| 维度 | 浅色模式 (`light`) | 深色模式 (`dark`) |
|---|---|---|
| 整体风格 | 偏「液态玻璃 + 晴空卡片」风格，画面轻、透、亮 | 偏「深海 / 深空玻璃面板」风格，画面更沉、更聚焦、更有霓虹感 |
| 主背景 | `--bg-main: #dfe8f7`，页面底色叠加浅蓝到白色渐变 | `--bg-main: #06101f`，页面底色叠加深蓝到近黑渐变 |
| 主文字 | `--text-primary: #10203f`，次级文字 `#465b84` | `--text-primary: #eef4ff`，次级文字 `#afc4ea` |
| 卡片/侧栏 | 半透明白色玻璃层：`--bg-sidebar: rgba(255,255,255,.24)`、`--bg-card: rgba(255,255,255,.2)` | 深色半透明玻璃层：`--bg-sidebar: rgba(10,20,38,.44)`、`--bg-card: rgba(10,20,38,.42)` |
| 边框/阴影 | 白边框 + 轻阴影，`--border: rgba(255,255,255,.32)`，`--shadow: 0 20px 60px rgba(40,72,140,.16)` | 冷蓝边框 + 更重阴影，`--border: rgba(149,181,255,.18)`，`--shadow: 0 24px 64px rgba(0,0,0,.42)` |
| 氛围层 | 大面积亮色径向光斑与网格，强调通透感 | 暗色径向辉光与低亮网格，强调沉浸感与层次对比 |
| 强调色 | 主强调色沿用 `--accent: #6d95ff`，hover 为 `#8cafff` | 主强调色保持一致，但结合深背景后会更偏冷蓝发光效果 |
| 状态提示 | 完成提示偏高饱和蓝：`#0037d6`；暖色胶囊偏浅橙奶油色 | 完成提示偏青蓝荧光：`#9ff3ff`；暖色胶囊改为深琥珀底 + 浅橙字 |

### 关键主题参数

以下参数是 `index.html` 当前最关键、最稳定的一组主题设计令牌，适合在联调或后续页面扩展时对齐：

| 参数组 | 关键变量 | 浅色代表值 | 深色代表值 |
|---|---|---|---|
| 基础色板 | `--bg-main` / `--text-primary` / `--text-secondary` | `#dfe8f7` / `#10203f` / `#465b84` | `#06101f` / `#eef4ff` / `#afc4ea` |
| 玻璃材质 | `--glass-surface` / `--glass-surface-strong` / `--glass-panel` | `rgba(255,255,255,.18)` / `rgba(255,255,255,.28)` / `rgba(255,255,255,.12)` | `rgba(11,23,44,.42)` / `rgba(14,29,54,.6)` / `rgba(9,18,34,.5)` |
| 玻璃边缘 | `--glass-edge` / `--glass-edge-strong` / `--glass-highlight` | `rgba(255,255,255,.42)` / `rgba(255,255,255,.68)` / `rgba(255,255,255,.86)` | `rgba(169,198,255,.22)` / `rgba(214,229,255,.34)` / `rgba(218,231,255,.2)` |
| 背景氛围 | `--bg-gradient` / `--liquid-grid` | 浅蓝、白色、青色流体光斑；`rgba(255,255,255,.16)` 网格 | 深蓝、青蓝辉光；`rgba(255,255,255,.08)` 网格 |
| 弹层面板 | `--modal-panel-bg` / `--modal-panel-border` / `--modal-field-bg` | `#eef4ff` / `rgba(72,99,143,.24)` / `#ffffff` | `#101d33` / `rgba(169,190,227,.24)` / `#14233c` |
| 操作反馈 | `--completion-beacon-*` | 以深蓝完成态为主 | 以青蓝荧光完成态为主 |
| 协作暖色提示 | `--warm-pill-*` | 浅橙描边、奶油底、棕橙文字 | 深琥珀底、浅橙描边、米橙文字 |
| 代码块操作 | `--code-block-action-*` | 浅底深字，hover 更亮 | 深底浅字，hover 更蓝更亮 |

### 与视觉稳定性强相关的非颜色参数

这些参数不区分深浅模式，但直接决定当前页面的设计风格和观感密度：

- 圆角：`--radius: 28px`、`--radius-sm: 18px`、`--radius-xs: 12px`
- 动效：`--transition: all .32s cubic-bezier(.22,1,.36,1)`
- 侧边栏收起宽度：`--sidebar-width-current: 48px`、`--right-sidebar-width-current: 48px`
- 聊天区间距：`--chat-frame-side-gap: 72px`
- 内容宽度：`--chat-content-max-width: 780px`、`--chat-shell-max-width: 1040px`
- 欢迎卡片宽度：`--chat-greeting-card-max-width: 780px`、`--chat-greeting-text-max-width: 560px`
- 模态遮罩：`--modal-overlay-bg: rgba(19,24,36,.18)`、`--modal-overlay-blur: 5px`
- 插件日志动效延迟：`--plugin-log-delay-ms: 2000ms`

说明：

- 上述参数是当前 `index.html` 的实现快照，适合作为视觉对齐基线。
- 若外部本地页面只需要感知“当前是深色还是浅色”，继续使用 `themeLabel` 即可。
- 若需要与 DeepRight 首页做高保真视觉对齐，建议直接对齐这套 `data-theme` + CSS 变量体系。

## 路由总览

### 核心接口

| 方法 | 路径 | 功能 | CLI |
|---|---|---|---|
| POST | `/v1/chat/completions` | 聊天补全代理 | `integration api chat-completions` |
| GET | `/api/heartbeat` | 查看心跳状态 | `integration api heartbeat` |
| POST | `/api/cancel` | 取消会话流 | `integration api cancel` |
| POST | `/api/restore` | 恢复会话日志 / 历史分页回溯 | `integration api restore` |
| GET | `/api/chat_session_log` | 查询会话日志 | `integration api chat-session-log` |
| POST | `/api/cmd` | 执行命令 | `integration api cmd` |
| POST | `/api/kill` | 停止命令 | `integration api kill` |
| GET | `/api/log_cleanup_status` | 查询启动期日志清理状态 | `integration api log-cleanup-status` |
| GET, POST | `/api/shutdown` | 延迟关闭 integration | 无 CLI 包装 |

### Agent / 文件接口

| 方法 | 路径 | 功能 | CLI |
|---|---|---|---|
| GET | `/api/agentId` | 返回 Agent ID 列表 | `integration api agent-id` |
| GET | `/api/swarm_agent` | 返回 swarm Agent 列表 | `integration api swarm-agent` |
| GET | `/api/deviceId` | 返回 deviceId | `integration api device-id` |
| GET | `/api/folder` | 打开 Agent 目录或绝对路径 | `integration api folder` |
| GET | `/api/skills` | 返回技能列表 | `integration api skills` |
| POST | `/api/skill_state` | 切换技能目录状态 | `integration api skill-state` |
| GET | `/api/files` | 浏览文件列表 | `integration api files` |
| GET | `/api/data` | 读取文本文件 | `integration api data` |
| GET | `/api/workspace` | 返回 workspace 路径 | `integration api workspace` |
| GET | `/api/url_preview_probe` | 探测 URL 可达性 / iframe 预览可用性 | `integration api url-preview-probe` |
| POST | `/api/edit` | 写文件 | `integration api edit` |
| GET | `/api/del` | 删除文件/目录 | `integration api del` |
| GET | `/api/raw` | 读取 Base64 原文 | `integration api raw` |
| GET | `/file/lastUpdate` | 文件最后更新时间 | `integration api file-last-update` |
| GET | `/api/agent/init` | 初始化 Agent | `integration api agent init` |
| GET | `/api/copy` | 复制 Agent 受管内容 | `integration api agent copy` |
| GET | `/api/agent/delete` | 删除 Agent | `integration api agent delete` |
| GET | `/api/agent/export` | 导出 Agent zip | `integration api agent export` |
| POST | `/api/agent/import` | 导入 Agent | `integration api agent import` |
| GET | `/api/agent/create` | 在 Agent 下建文件/目录 | `integration api agent create` |
| POST | `/api/upload` | 上传文件到 Agent | `integration api upload` |
| GET | `/api/download` | 下载文件/目录 | `integration api download` |

### 运行时 / 配置 / 沙盒接口

| 方法 | 路径 | 功能 | CLI |
|---|---|---|---|
| POST | `/api/config` | 更新 Agent 配置 / 删除共享模型 | `integration api config` |
| GET, POST | `/api/token` | 读取/保存模型配置 | `integration api token get/set` |
| GET | `/api/consume` | 查询 token 消费 | `integration api consume` |
| POST | `/api/message_insert/add` | 新增插入消息 | `integration api message-insert add` |
| POST | `/api/message_insert/del` | 标记取消插入消息 | `integration api message-insert del` |
| POST | `/api/message_insert/delete` | 物理删除插入消息 | `integration api message-insert delete` |
| GET | `/api/message_insert/list` | 查询插入消息列表 | `integration api message-insert list` |
| GET, POST | `/api/sandbox` | 读写会话沙盒 | `integration api sandbox` |
| GET, POST | `/api/sandbox=off` | 关闭沙盒 | `integration api sandbox --sandbox off` |
| GET, POST | `/api/sandbox=filepick` | 设为 `filepick` | `integration api sandbox --sandbox filepick` |
| GET, POST | `/api/sandbox=net` | 设为 `net` | `integration api sandbox --sandbox net` |
| GET, POST | `/api/sandbox=filepick_net` | 设为 `filepick_net` | `integration api sandbox --sandbox filepick_net` |
| GET, POST | `/api/sandbox_status` | 查询沙盒状态 | `integration api sandbox` |
| GET, POST, PUT, DELETE | `/api/host` | 查询/修改运行时 host | `integration api host` |
| GET, POST, PUT, DELETE | `/api/standalone` | 查询/修改 standalone | `integration api standalone` |
| GET | `/api/standalone=true` | 快捷开启 standalone | `integration api standalone set --value true` |
| GET | `/api/standalone=false` | 快捷关闭 standalone | `integration api standalone set --value false` |
| GET | `/api/site/access` | 查询站点访问地址 | `integration api site-access` |

### 插件 / Connect / Cron 接口

| 方法 | 路径 | 功能 | CLI |
|---|---|---|---|
| GET | `/api/plugins/meta` | 插件元数据 | `integration api plugins meta` |
| GET | `/api/plugins/status` | 插件状态 | `integration api plugins status` |
| POST | `/api/plugins/config` | 保存插件配置 | `integration api plugins config` |
| POST | `/api/plugins/start` | 启动插件 | `integration api plugins start` |
| POST | `/api/plugins/stop` | 停止插件 | `integration api plugins stop` |
| GET | `/api/plugins/exec` | 执行插件命令 | `integration api plugins exec` |
| GET | `/api/plugins/log` | 流式查看插件日志 | `integration api plugins log` |
| GET, POST, PUT, DELETE | `/api/connect/meta` | Connect meta CRUD | `integration api connect ...` |
| GET, POST, PUT | `/api/connect/request` | Connect request 增查改状态 | `integration api connect ...` |
| GET, POST | `/api/connect/response` | Connect response 增查 | `integration api connect ...` |
| POST | `/api/cron/create` | 创建 cron 任务 | `integration api cron create` |
| POST | `/api/cron/detail/metadata` | 查 cron 元数据 | `integration api cron detail-metadata` |
| POST | `/api/cron/delete` | 删除 cron 元数据 | `integration api cron delete` |
| POST | `/api/cron/detail/delete` | 删除 cron 明细 | `integration api cron detail-delete` |
| POST | `/api/cron/detail/list` | 查 cron 明细 | `integration api cron detail-list` |
| POST | `/api/cron/detail/status` | 更新 cron 明细状态 | `integration api cron detail-status` |

### 辅助接口

| 方法 | 路径 | 功能 | CLI |
|---|---|---|---|
| GET | `/skills_warning` | 返回 skills warning | `integration api skills-warning` |
| GET | `/install_app` | 返回缺失依赖列表 | `integration api install-app` |
| GET | `/log_round` | 导出 round 日志 | `integration api log-round` |
| GET | `/log_skill` | 返回技能日志文件信息 | `integration api log-skill` |
| GET | `/log_skill_status` | 返回技能日志状态 | `integration api log-skill-status` |
| GET | `/knowledge` | knowledge 根目录树/文件 | `integration api knowledge` |
| GET | `/knowledge/...` | knowledge 子路径 | `integration api knowledge --path ...` |
| GET | `/knowledge_lastUpdate` | knowledge 更新时间文本 | `integration api knowledge-last-update` |
| GET | `/knowledge_path` | knowledge 目录路径 | `integration api knowledge-path` |
| GET, HEAD | `/launch` | 浏览器启动跳转页 | 无 CLI 包装 |
| GET, HEAD | `/mapping/...` | app 静态资源 | 无 CLI 包装 |
| 多方法 | `server.Register(mux, cfg.Site)` | site 静态文件服务 | 无 CLI 包装 |

---

## 详细用法

### 1. `POST /v1/chat/completions`

用途：把请求转发到上游补全接口，并由 integration 注入 metadata。

HTTP：

```bash
curl -X POST 'http://127.0.0.1:8080/v1/chat/completions' \
  -H 'Content-Type: application/json' \
  -d '{
    "model":"OpenAI",
    "stream":true,
    "messages":[{"role":"user","content":"hello"}],
    "metadata":{"agentId":"demo-agent","chat":"chat-001"}
  }'
```

CLI：

```bash
integration api chat-completions --body-file ./chat-completions.json
```

### 2. `GET /api/heartbeat`

用途：检查 cli-get 心跳。`curl 'http://127.0.0.1:8080/api/heartbeat'` ｜ `integration api heartbeat`

### 3. `POST /api/cancel`

用途：取消指定会话的流式响应。

HTTP：

```bash
curl -X POST 'http://127.0.0.1:8080/api/cancel?chat=chat-001'
```

CLI：

```bash
integration api cancel --chat chat-001
```

### 4. `POST /api/restore`

用途：按时间线恢复会话日志，或按历史分页向前翻页。

HTTP：

```bash
curl -X POST 'http://127.0.0.1:8080/api/restore?agentId=demo-agent&chat=chat-001&timeline=2026-07-06T10:00:00&lastId=100'
```

CLI：

```bash
integration api restore \
  --agentId demo-agent \
  --chat chat-001 \
  --timeline 2026-07-06T10:00:00 \
  --lastId 100
```

历史分页模式：

```bash
curl -X POST 'http://127.0.0.1:8080/api/restore?agentId=demo-agent&chat=chat-001&history=1&limit=120'

integration api restore \
  --agentId demo-agent \
  --chat chat-001 \
  --history true \
  --limit 120
```

继续翻上一页时，可继续携带上一页响应中的 `history.beforeTimeline` 和 `history.beforeId`：

```bash
curl -X POST 'http://127.0.0.1:8080/api/restore?agentId=demo-agent&chat=chat-001&history=1&beforeTimeline=2026-07-06T10:00:00&beforeId=321&limit=120'

integration api restore \
  --agentId demo-agent \
  --chat chat-001 \
  --history true \
  --beforeTimeline 2026-07-06T10:00:00 \
  --beforeId 321 \
  --limit 120
```

说明：

- 前向恢复仍使用 `timeline` + `lastId`
- 历史分页模式下，响应会额外返回 `history.hasMore`、`history.beforeTimeline`、`history.beforeId`

### 5. `GET /api/chat_session_log`

用途：查询会话日志窗口。

HTTP：

```bash
curl 'http://127.0.0.1:8080/api/chat_session_log?agentId=demo-agent&chatId=chat-001&limit=50'
```

CLI：

```bash
integration api chat-session-log --agentId demo-agent --chatId chat-001 --limit 50
```

### 6. `POST /api/cmd`

用途：为指定 `agentId + chatId` 执行命令。

HTTP：

```bash
curl -X POST 'http://127.0.0.1:8080/api/cmd' \
  -H 'Content-Type: application/json' \
  -d '{"agentId":"demo-agent","chatId":"chat-001","cmd":"pwd","timeout":30000}'
```

CLI：

```bash
integration api cmd --agentId demo-agent --chatId chat-001 --cmd 'pwd' --timeout 30000
```

### 7. `POST /api/kill`

用途：停止 `/api/cmd` 启动的活动命令。

HTTP：

```bash
curl -X POST 'http://127.0.0.1:8080/api/kill' \
  -H 'Content-Type: application/json' \
  -d '{"agentId":"demo-agent","chatId":"chat-001","cmd":"sleep 10"}'
```

CLI：

```bash
integration api kill --agentId demo-agent --chatId chat-001 --cmd 'sleep 10'
```

### 8-17. Agent / 文件快速示例

这些接口都属于“轻参数直接调用”；方法与 CLI 名见路由总览，这里只保留最短示例。

- `/api/agentId`：Agent 列表。`curl 'http://127.0.0.1:8080/api/agentId'` ｜ `integration api agent-id`
- `/api/swarm_agent`：swarm Agent 列表。`curl 'http://127.0.0.1:8080/api/swarm_agent'` ｜ `integration api swarm-agent`
- `/api/deviceId`：deviceId。`curl 'http://127.0.0.1:8080/api/deviceId'` ｜ `integration api device-id`
- `/api/folder`：打开 Agent workspace 或绝对路径。`curl 'http://127.0.0.1:8080/api/folder?agentId=demo-agent'` / `curl 'http://127.0.0.1:8080/api/folder?path=/Users/me/Desktop'` ｜ `integration api folder --agentId demo-agent` / `integration api folder --path /Users/me/Desktop`
- `/api/skills`：技能列表；常用参数 `agentId`、`chatId`。`curl 'http://127.0.0.1:8080/api/skills?agentId=demo-agent&chatId=chat-001'` ｜ `integration api skills --agentId demo-agent --chatId chat-001`
- `/api/skill_state`：切换技能目录禁用状态；HTTP body 形如 `{"chatId":"chat-001","path":"/abs/skills/demo","disabled":true}` ｜ `integration api skill-state --chatId chat-001 --path /abs/skills/demo --disabled true`
- `/api/files`：文件列表。`curl 'http://127.0.0.1:8080/api/files?path=/abs/workspace'` ｜ `integration api files --path /abs/workspace`
- `/api/data`：读取文本文件。`curl 'http://127.0.0.1:8080/api/data?path=/abs/workspace/USER.md'` ｜ `integration api data --path /abs/workspace/USER.md`
- `/api/workspace`：workspace 路径。`curl 'http://127.0.0.1:8080/api/workspace?agentId=demo-agent'` ｜ `integration api workspace --agentId demo-agent`
- `/api/url_preview_probe`：探测 URL 可达性 / iframe 预览可用性。普通用法：`curl --get 'http://127.0.0.1:8080/api/url_preview_probe' --data-urlencode 'url=https://example.com'` ｜ `integration api url-preview-probe --url https://example.com`

本地页面用法：若目标 URL 的 host 是 `localhost` / `127.0.0.1`，请把 `agentId`、`chatId`、`themeLabel` 放进 `url` 本身；integration 只探测最终 URL，目标本地页面再消费这些参数。例如：

```bash
curl --get 'http://127.0.0.1:8080/api/url_preview_probe' \
  --data-urlencode 'url=http://localhost:3000/demo?agentId=demo-agent&chatId=chat-001&themeLabel=深色模式'
```

### 18. `POST /api/edit`

用途：写文本文件或二进制文件。

HTTP：

```bash
curl -X POST 'http://127.0.0.1:8080/api/edit?agentId=demo-agent&path=USER.md' \
  -H 'Content-Type: application/json' \
  -d '{"content":"# hello"}'
```

CLI：

```bash
integration api edit --agentId demo-agent --path USER.md --content '# hello'
```

二进制示例：

```bash
integration api edit --agentId demo-agent --path app/logo.png --base64 "$(<logo.base64.txt)"
```

### 19. `GET /api/del`

HTTP：

```bash
curl 'http://127.0.0.1:8080/api/del?agentId=demo-agent&path=tmp/a.txt'
```

CLI：

```bash
integration api del --agentId demo-agent --path tmp/a.txt
```

### 20. `GET /api/raw`

HTTP：

```bash
curl 'http://127.0.0.1:8080/api/raw?agentId=demo-agent&path=app/logo.png'
```

CLI：

```bash
integration api raw --agentId demo-agent --path app/logo.png
```

### 21. `GET /file/lastUpdate`

HTTP：

```bash
curl 'http://127.0.0.1:8080/file/lastUpdate?agentId=demo-agent&file=USER.md'
```

CLI：

```bash
integration api file-last-update --agentId demo-agent --file USER.md
```

### 22. Agent 初始化 / 删除 / 创建 / 导入 / 导出 / 复制

初始化：

```bash
curl 'http://127.0.0.1:8080/api/agent/init?name=demo-agent'
integration api agent init --name demo-agent
```

删除：

```bash
curl 'http://127.0.0.1:8080/api/agent/delete?name=demo-agent'
integration api agent delete --name demo-agent
```

创建文件或目录：

```bash
curl 'http://127.0.0.1:8080/api/agent/create?agentId=demo-agent&name=docs/readme.md&type=1'
integration api agent create --agentId demo-agent --name docs/readme.md --type 1
```

导出：

```bash
curl -OJ 'http://127.0.0.1:8080/api/agent/export?agent_id=demo-agent'
integration api agent export --agent demo-agent --output ./demo-agent.zip
```

导入：

```bash
integration api agent import --input ./demo-agent.zip
```

复制：

```bash
curl 'http://127.0.0.1:8080/api/copy?source_agentId=src&target_agentId=dst'
integration api agent copy --source src --target dst
```

### 23. `POST /api/upload`

HTTP：

```bash
curl -X POST 'http://127.0.0.1:8080/api/upload?agentId=demo-agent' \
  -F 'files=@./local.txt'
```

CLI：

```bash
integration api upload --agentId demo-agent --file ./local.txt
```

### 24. `GET /api/download`

HTTP：

```bash
curl -o ./download.bin 'http://127.0.0.1:8080/api/download?path=/abs/workspace/USER.md'
```

CLI：

```bash
integration api download --path /abs/workspace/USER.md --output ./USER.md
```

### 25. `POST /api/config`

HTTP：

```bash
curl -X POST 'http://127.0.0.1:8080/api/config?agentId=demo-agent' \
  -H 'Content-Type: application/json' \
  -d @./agent-config.json
```

CLI：

```bash
integration api config --agentId demo-agent --body-file ./agent-config.json
```

删除共享模型配置：

```bash
curl -X POST 'http://127.0.0.1:8080/api/config' \
  -H 'Content-Type: application/json' \
  -d '{"action":"delete_model","model":"OpenAI"}'

integration api config --body '{"action":"delete_model","model":"OpenAI"}'
```

说明：

- 更新 Agent `config.json` 时需要传 `agentId`
- 执行 `delete_model` 动作时不需要 `agentId`

### 26. `GET/POST /api/token`

读取：

```bash
curl 'http://127.0.0.1:8080/api/token'
integration api token get
```

写入：

```bash
curl -X POST 'http://127.0.0.1:8080/api/token' \
  -H 'Content-Type: application/json' \
  -d @./token.json
integration api token set --body-file ./token.json
```

### 27. `GET /api/consume`

HTTP：

```bash
curl 'http://127.0.0.1:8080/api/consume?agentId=demo-agent&starTime=20260706-090000&closeTime=20260706-180000&limit=200'
```

CLI：

```bash
integration api consume --agentId demo-agent --start 20260706-090000 --close 20260706-180000 --limit 200
```

### 28. `message_insert` 系列

新增：

```bash
integration api message-insert add --agentId demo-agent --chatId chat-001 --tid 1718966400000 --message 'HELLO'
```

取消：

```bash
integration api message-insert del --chatId chat-001 --tid 1718966400000
```

物理删除：

```bash
integration api message-insert delete --chatId chat-001 --tid 1718966400000
```

列表：

```bash
integration api message-insert list --chatId chat-001
```

### 29. `sandbox` 系列

查询：

```bash
integration api sandbox --agentId demo-agent --chatId chat-001
```

设置：

```bash
integration api sandbox --agentId demo-agent --chatId chat-001 --sandbox filepick_net
integration api sandbox --agentId demo-agent --chatId chat-001 --sandbox off
```

### 30. `host` / `standalone`

Host：

```bash
integration api host get
integration api host set --value https://staging.deepright.cn
integration api host reset
```

Standalone：

```bash
integration api standalone get
integration api standalone set --value true
integration api standalone reset
```

### 31. `GET /api/site/access`

HTTP：

```bash
curl 'http://127.0.0.1:8080/api/site/access'
```

CLI：

```bash
integration api site-access
```

### 32. `plugins` 系列

插件列表：

```bash
curl 'http://127.0.0.1:8080/api/plugins/meta'
integration api plugins meta
```

状态：

```bash
integration api plugins status --key feishu
```

配置：

```bash
integration api plugins config --body-file ./plugin-config.json
```

启动 / 停止：

```bash
integration api plugins start --key feishu --connect-bin ./integration
integration api plugins stop --key feishu --connect-bin ./integration
```

执行：

```bash
integration api plugins exec --key browser --command 'instance init' --agentId demo-agent --chatId chat-001
```

日志：

```bash
curl 'http://127.0.0.1:8080/api/plugins/log?key=browser&last=50'
integration api plugins log --key browser --last 50
```

### 33. `connect` 系列

Meta：

```bash
integration api connect meta-create --key feishu --meta '{"token":"abc"}' --callback ignored --agent demo-agent --model OpenAI
integration api connect meta-get --key feishu
integration api connect meta-list
integration api connect meta-update --key feishu --meta '{"token":"def"}' --callback ignored
integration api connect meta-delete --key feishu
```

Request：

```bash
integration api connect add-request --key feishu --externalId ext-1 --content 'HELLO WORLD'
integration api connect request-list --key feishu --limit 20
```

Response：

```bash
integration api connect add-response --key feishu --request-id 1 --response '{"ok":true}'
integration api connect response-list --key feishu --request-id 1
```

### 34. `cron` 系列

创建一次性任务：

```bash
integration api cron create \
  --content '明早整理日报' \
  --model OpenAI \
  --rawTime '2026-07-07 09:00' \
  --cycle 0 \
  --agent demo-agent
```

创建 cron 表达式任务：

```bash
integration api cron create-cron \
  --content '工作日中午整理异常日志' \
  --model OpenAI \
  --cron '10 12 * * 1-5' \
  --agent demo-agent
```

查元数据：

```bash
integration api cron detail-metadata --agentId demo-agent --chatId chat-001
```

查明细：

```bash
integration api cron detail-list --metaId cron_1
```

删元数据 / 明细：

```bash
integration api cron delete --id meta_1
integration api cron detail-delete --detailId detail_1
```

更新明细状态：

```bash
curl -X POST 'http://127.0.0.1:8080/api/cron/detail/status?agentId=demo-agent&detailId=detail_1&status=3'
integration api cron detail-status --agentId demo-agent --detailId detail_1 --status 3
```

### 辅助查询与页面路由快速示例

- `/skills_warning`：`curl 'http://127.0.0.1:8080/skills_warning'` ｜ `integration api skills-warning`
- `/install_app`：`curl 'http://127.0.0.1:8080/install_app'` ｜ `integration api install-app`
- `/log_round`：`curl 'http://127.0.0.1:8080/log_round?chatId=chat-001&round=3'` ｜ `integration api log-round --chatId chat-001 --round 3`
- `/log_skill`：`curl 'http://127.0.0.1:8080/log_skill?chatId=chat-001&round=3'` ｜ `integration api log-skill --chatId chat-001 --round 3`
- `/log_skill_status`：`curl 'http://127.0.0.1:8080/log_skill_status?agentId=demo-agent&chatId=chat-001&round=3'` ｜ `integration api log-skill-status --agentId demo-agent --chatId chat-001 --round 3`
- `/api/log_cleanup_status`：`curl 'http://127.0.0.1:8080/api/log_cleanup_status'` ｜ `integration api log-cleanup-status`
- `knowledge`：目录树 `curl 'http://127.0.0.1:8080/knowledge'` ｜ `integration api knowledge`
- `knowledge` 子路径：`curl 'http://127.0.0.1:8080/knowledge/demo-agent'` ｜ `integration api knowledge --path demo-agent`
- `knowledge_lastUpdate`：`curl 'http://127.0.0.1:8080/knowledge_lastUpdate?agentId=demo-agent'` ｜ `integration api knowledge-last-update --agentId demo-agent`
- `knowledge_path`：`curl 'http://127.0.0.1:8080/knowledge_path?agentId=demo-agent'` ｜ `integration api knowledge-path --agentId demo-agent`
- `/launch`：浏览器启动跳转页，常用于本机拉起后等待 `/site/` 准备完成；示例：直接访问 `http://127.0.0.1:8080/launch`

### `/api/shutdown`

按需求，不提供 CLI 包装。

HTTP：

```bash
curl -X POST 'http://127.0.0.1:8080/api/shutdown'
curl 'http://127.0.0.1:8080/api/shutdown'
```

限制：

- 仅允许 `localhost` / `127.0.0.1` / `::1`
- 返回接受成功后，大约 5 秒后触发关闭

## 备注

- `/mapping/...` 与 `server.Register(mux, cfg.Site)` 提供的是静态资源/页面服务，不额外包装为 CLI。
- `integration api ...` 是主命令；`integration service ...` 为兼容别名。
- 对于 `cron / connect / plugins / agent / sandbox / message-insert / host / standalone` 这类已有成熟 CLI 的能力，`integration api ...` 会尽量复用原命令行为，以保证帮助信息和输出语义稳定。
