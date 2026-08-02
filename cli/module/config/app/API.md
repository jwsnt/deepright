# Integration API

`integration` 默认对外提供 HTTP 服务，默认地址为 `http://localhost:#port`。`#port` 是源码文档占位符，构建发布包时会替换为 `config/config.json` 的 `port` 属性；启动时仍可由 `--port` 覆盖端口。以下 URL 示例使用发布包配置的端口。

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
- `--port PORT`：指定 integration HTTP 端口；省略时使用当前运行端口（默认 `8080`）
- `--output PATH`：把响应体保存到文件

常用示例：

```bash
integration api heartbeat
integration api agent-id
integration api edit --agentId demo --path USER.md --content '# hello'
integration api whisper create --agentId demo --path audios/meeting.mp3 --scenario chinese_meeting
integration api rembg create --agentId demo --path images/product.jpg --model u2net --alpha-matting
integration api voxcpm create --agentId demo --textPath scripts/intro.txt --outputName intro.wav
integration service cancel --chat chat-001
```

## 本地页面 URL 约定

适用场景：site 侧 assistant URL 小气泡的 `预览` / `打开`，以及任何主动探测或跳转到本地页面的场景。

当目标 URL 的 host 为 `localhost` 或 `127.0.0.1` 时，建议把上下文放进目标 URL 自身：

| 参数 | 用途 |
|---|---|
| `agentId` | 当前 Agent ID，供目标本地页面恢复 Agent 上下文 |
| `chatId` | 当前会话 ID，供目标本地页面恢复会话上下文 |
| `themeLabel` | 当前主题标签；冷色传 `冷色模式`，暖色传 `暖色模式` |

说明：

- 这三个参数不是 integration 服务端统一强制消费的公共 query；integration 通常只透传、打开或探测最终 URL。
- 真正消费这些参数的是目标本地页面。
- 原 URL 已有 query 时继续追加；若已存在同名键，建议按当前上下文覆盖。

## 页面主题说明

`themeLabel` 只用于向外部本地页面传递当前页面上下文；DeepRight 首页不读取该 query 来切换主题。完整的配色令牌、视觉规则和 `light` / `dark` 实现约定见 [DESIGN.md](./DESIGN.md)。

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

### Whisper 音频文字提取接口

| 方法 | 路径 | 功能 | CLI |
|---|---|---|---|
| GET | `/api/whisper/check` | 检查受控环境能否运行 Whisper | `integration api whisper check` |
| GET | `/api/whisper/tasks` | 查询当前 Agent 的转写任务（固定每页 5 条） | `integration api whisper list` |
| POST | `/api/whisper/tasks` | 将一个或多个工作区音频加入转写队列 | `integration api whisper create` |
| POST | `/api/whisper/tasks/cancel` | 取消排队中或执行中的任务 | `integration api whisper cancel` |
| POST | `/api/whisper/tasks/restart` | 将已取消任务重新加入队列 | `integration api whisper restart` |
| POST | `/api/whisper/tasks/delete` | 删除失败任务的记录，不删除文件 | `integration api whisper delete` |
| GET | `/api/whisper/tasks/log` | 查询任务详情及持久化执行日志 | `integration api whisper log` |

### rembg 图片主体提取接口

| 方法 | 路径 | 功能 | CLI |
|---|---|---|---|
| GET | `/api/rembg/check` | 检查受控环境能否运行 rembg | `integration api rembg check` |
| GET | `/api/rembg/tasks` | 查询当前 Agent 的图片主体任务（固定每页 5 条） | `integration api rembg list` |
| POST | `/api/rembg/tasks` | 将一个或多个工作区图片加入提取队列 | `integration api rembg create` |
| POST | `/api/rembg/tasks/cancel` | 取消排队中或执行中的任务 | `integration api rembg cancel` |
| POST | `/api/rembg/tasks/restart` | 将已取消任务重新加入队列 | `integration api rembg restart` |
| POST | `/api/rembg/tasks/delete` | 删除失败任务的记录，不删除源图片或输出图片 | `integration api rembg delete` |
| GET | `/api/rembg/tasks/log` | 查询任务详情及持久化执行日志 | `integration api rembg log` |

### VoxCPM 文字转语音接口

| 方法 | 路径 | 功能 | CLI |
|---|---|---|---|
| GET | `/api/voxcpm/check` | 检查受控环境能否运行 VoxCPM | `integration api voxcpm check` |
| GET | `/api/voxcpm/tasks` | 查询当前 Agent 的文字转语音任务（固定每页 5 条） | `integration api voxcpm list` |
| POST | `/api/voxcpm/tasks` | 将一至五条工作区文字任务加入队列 | `integration api voxcpm create` |
| POST | `/api/voxcpm/tasks/cancel` | 取消排队中或执行中的任务 | `integration api voxcpm cancel` |
| POST | `/api/voxcpm/tasks/restart` | 将已取消任务重新加入队列 | `integration api voxcpm restart` |
| POST | `/api/voxcpm/tasks/delete` | 删除失败任务的记录，不删除源文件或输出文件 | `integration api voxcpm delete` |
| GET | `/api/voxcpm/tasks/log` | 查询任务详情及持久化执行日志 | `integration api voxcpm log` |

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
curl -X POST 'http://localhost:#port/v1/chat/completions' \
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

用途：检查 cli-get 心跳。`curl 'http://localhost:#port/api/heartbeat'` ｜ `integration api heartbeat`

### 3. `POST /api/cancel`

用途：取消指定会话的流式响应。

HTTP：

```bash
curl -X POST 'http://localhost:#port/api/cancel?chat=chat-001'
```

CLI：

```bash
integration api cancel --chat chat-001
```

### 4. `POST /api/restore`

用途：按时间线恢复会话日志，或按历史分页向前翻页。

HTTP：

```bash
curl -X POST 'http://localhost:#port/api/restore?agentId=demo-agent&chat=chat-001&timeline=2026-07-06T10:00:00&lastId=100'
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
curl -X POST 'http://localhost:#port/api/restore?agentId=demo-agent&chat=chat-001&history=1&limit=120'

integration api restore \
  --agentId demo-agent \
  --chat chat-001 \
  --history true \
  --limit 120
```

继续翻上一页时，可继续携带上一页响应中的 `history.beforeTimeline` 和 `history.beforeId`：

```bash
curl -X POST 'http://localhost:#port/api/restore?agentId=demo-agent&chat=chat-001&history=1&beforeTimeline=2026-07-06T10:00:00&beforeId=321&limit=120'

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
- `history=1` 仅返回中心会话正文所需的 `Q`、`A`、`X`；`cli/get`、`cli/pub` 只在前向恢复中按实时会话需要合并返回
- 历史分页按 `chatId + createdAt + id` 游标读取最近记录；若页面从一轮回答中间开始，服务会向前补到该轮对应的 `Q` 后再按正序返回

### 5. `GET /api/chat_session_log`

用途：查询会话日志窗口。

HTTP：

```bash
curl 'http://localhost:#port/api/chat_session_log?agentId=demo-agent&chatId=chat-001&limit=50'
```

CLI：

```bash
integration api chat-session-log --agentId demo-agent --chatId chat-001 --limit 50
```

### 6. `POST /api/cmd`

用途：为指定 `agentId + chatId` 执行命令。

HTTP：

```bash
curl -X POST 'http://localhost:#port/api/cmd' \
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
curl -X POST 'http://localhost:#port/api/kill' \
  -H 'Content-Type: application/json' \
  -d '{"agentId":"demo-agent","chatId":"chat-001","cmd":"sleep 10"}'
```

CLI：

```bash
integration api kill --agentId demo-agent --chatId chat-001 --cmd 'sleep 10'
```

### 8-17. Agent / 文件快速示例

这些接口都属于“轻参数直接调用”；方法与 CLI 名见路由总览，这里只保留最短示例。

- `/api/agentId`：Agent 列表。`curl 'http://localhost:#port/api/agentId'` ｜ `integration api agent-id`
- `/api/swarm_agent`：swarm Agent 列表。`curl 'http://localhost:#port/api/swarm_agent'` ｜ `integration api swarm-agent`
- `/api/deviceId`：deviceId。`curl 'http://localhost:#port/api/deviceId'` ｜ `integration api device-id`
- `/api/folder`：打开 Agent workspace 或绝对路径。`curl 'http://localhost:#port/api/folder?agentId=demo-agent'` / `curl 'http://localhost:#port/api/folder?path=/Users/me/Desktop'` ｜ `integration api folder --agentId demo-agent` / `integration api folder --path /Users/me/Desktop`
- `/api/skills`：技能列表；常用参数 `agentId`、`chatId`。`curl 'http://localhost:#port/api/skills?agentId=demo-agent&chatId=chat-001'` ｜ `integration api skills --agentId demo-agent --chatId chat-001`
- `/api/skill_state`：切换技能目录禁用状态；HTTP body 形如 `{"chatId":"chat-001","path":"/abs/skills/demo","disabled":true}` ｜ `integration api skill-state --chatId chat-001 --path /abs/skills/demo --disabled true`
- `/api/files`：文件列表。`curl 'http://localhost:#port/api/files?path=/abs/workspace'` ｜ `integration api files --path /abs/workspace`
- `/api/data`：读取文本文件。`curl 'http://localhost:#port/api/data?path=/abs/workspace/USER.md'` ｜ `integration api data --path /abs/workspace/USER.md`
- `/api/workspace`：workspace 路径。`curl 'http://localhost:#port/api/workspace?agentId=demo-agent'` ｜ `integration api workspace --agentId demo-agent`
- `/api/url_preview_probe`：探测 URL 可达性 / iframe 预览可用性。普通用法：`curl --get 'http://localhost:#port/api/url_preview_probe' --data-urlencode 'url=https://example.com'` ｜ `integration api url-preview-probe --url https://example.com`

本地页面用法：若目标 URL 的 host 是 `localhost` / `127.0.0.1`，请把 `agentId`、`chatId`、`themeLabel` 放进 `url` 本身；integration 只探测最终 URL，目标本地页面再消费这些参数。例如：

```bash
curl --get 'http://localhost:#port/api/url_preview_probe' \
  --data-urlencode 'url=http://localhost:3000/demo?agentId=demo-agent&chatId=chat-001&themeLabel=冷色模式'
```

### 18. `POST /api/edit`

用途：写文本文件或二进制文件。

HTTP：

```bash
curl -X POST 'http://localhost:#port/api/edit?agentId=demo-agent&path=USER.md' \
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
curl 'http://localhost:#port/api/del?agentId=demo-agent&path=tmp/a.txt'
```

CLI：

```bash
integration api del --agentId demo-agent --path tmp/a.txt
```

### 20. `GET /api/raw`

HTTP：

```bash
curl 'http://localhost:#port/api/raw?agentId=demo-agent&path=app/logo.png'
```

CLI：

```bash
integration api raw --agentId demo-agent --path app/logo.png
```

### 21. `GET /file/lastUpdate`

HTTP：

```bash
curl 'http://localhost:#port/file/lastUpdate?agentId=demo-agent&file=USER.md'
```

CLI：

```bash
integration api file-last-update --agentId demo-agent --file USER.md
```

### 22. Agent 初始化 / 删除 / 创建 / 导入 / 导出 / 复制

初始化：

```bash
curl 'http://localhost:#port/api/agent/init?name=demo-agent'
integration api agent init --name demo-agent
```

删除：

```bash
curl 'http://localhost:#port/api/agent/delete?name=demo-agent'
integration api agent delete --name demo-agent
```

创建文件或目录：

```bash
curl 'http://localhost:#port/api/agent/create?agentId=demo-agent&name=docs/readme.md&type=1'
integration api agent create --agentId demo-agent --name docs/readme.md --type 1
```

导出：

```bash
curl -OJ 'http://localhost:#port/api/agent/export?agent_id=demo-agent'
integration api agent export --agent demo-agent --output ./demo-agent.zip
```

导入：

```bash
integration api agent import --input ./demo-agent.zip
```

复制：

```bash
curl 'http://localhost:#port/api/copy?source_agentId=src&target_agentId=dst'
integration api agent copy --source src --target dst
```

### 23. `POST /api/upload`

HTTP：

```bash
curl -X POST 'http://localhost:#port/api/upload?agentId=demo-agent' \
  -F 'files=@./local.txt'
```

CLI：

```bash
integration api upload --agentId demo-agent --file ./local.txt
```

### 24. `GET /api/download`

HTTP：

```bash
curl -o ./download.bin 'http://localhost:#port/api/download?path=/abs/workspace/USER.md'
```

CLI：

```bash
integration api download --path /abs/workspace/USER.md --output ./USER.md
```

### 25. `POST /api/config`

HTTP：

```bash
curl -X POST 'http://localhost:#port/api/config?agentId=demo-agent' \
  -H 'Content-Type: application/json' \
  -d @./agent-config.json
```

CLI：

```bash
integration api config --agentId demo-agent --body-file ./agent-config.json
```

删除共享模型配置：

```bash
curl -X POST 'http://localhost:#port/api/config' \
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
curl 'http://localhost:#port/api/token'
integration api token get
```

写入：

```bash
curl -X POST 'http://localhost:#port/api/token' \
  -H 'Content-Type: application/json' \
  -d @./token.json
integration api token set --body-file ./token.json
```

### 27. `GET /api/consume`

HTTP：

```bash
curl 'http://localhost:#port/api/consume?agentId=demo-agent&starTime=20260706-090000&closeTime=20260706-180000&limit=200'
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
curl 'http://localhost:#port/api/site/access'
```

CLI：

```bash
integration api site-access
```

### 32. `plugins` 系列

插件列表：

```bash
curl 'http://localhost:#port/api/plugins/meta'
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
curl 'http://localhost:#port/api/plugins/log?key=browser&last=50'
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
curl -X POST 'http://localhost:#port/api/cron/detail/status?agentId=demo-agent&detailId=detail_1&status=3'
integration api cron detail-status --agentId demo-agent --detailId detail_1 --status 3
```

### 辅助查询与页面路由快速示例

- `/skills_warning`：`curl 'http://localhost:#port/skills_warning'` ｜ `integration api skills-warning`
- `/install_app`：`curl 'http://localhost:#port/install_app'` ｜ `integration api install-app`
- `/log_round`：`curl 'http://localhost:#port/log_round?chatId=chat-001&round=3'` ｜ `integration api log-round --chatId chat-001 --round 3`
- `/log_skill`：`curl 'http://localhost:#port/log_skill?chatId=chat-001&round=3'` ｜ `integration api log-skill --chatId chat-001 --round 3`
- `/log_skill_status`：`curl 'http://localhost:#port/log_skill_status?agentId=demo-agent&chatId=chat-001&round=3'` ｜ `integration api log-skill-status --agentId demo-agent --chatId chat-001 --round 3`
- `/api/log_cleanup_status`：`curl 'http://localhost:#port/api/log_cleanup_status'` ｜ `integration api log-cleanup-status`
- `knowledge`：目录树 `curl 'http://localhost:#port/knowledge'` ｜ `integration api knowledge`
- `knowledge` 子路径：`curl 'http://localhost:#port/knowledge/demo-agent'` ｜ `integration api knowledge --path demo-agent`
- `knowledge_lastUpdate`：`curl 'http://localhost:#port/knowledge_lastUpdate?agentId=demo-agent'` ｜ `integration api knowledge-last-update --agentId demo-agent`
- `knowledge_path`：`curl 'http://localhost:#port/knowledge_path?agentId=demo-agent'` ｜ `integration api knowledge-path --agentId demo-agent`
- `/launch`：浏览器启动跳转页，常用于本机拉起后等待 `/site/` 准备完成；示例：直接访问 `http://localhost:#port/launch`

### Whisper 音频文字提取

适用范围：所有请求都由 Integration 的受控运行环境执行。每个任务使用服务端白名单中的转写场景，默认 `chinese_meeting`（中文会议）；任务在全部 Agent 间串行调度，输出写入对应 Agent 工作目录的 `whisper/`。任务状态为 `queued`、`running`、`completed`、`cancelled` 或 `failed`。

| 场景 | 标识 | 适用情况 |
|---|---|---|
| 中文会议 | `chinese_meeting` | 默认；中文会议、访谈，准确率与速度均衡 |
| 专业转写 | `chinese_accurate` | 专业术语或正式字幕，优先准确率 |
| 低延迟 | `realtime` | 实时预览，优先响应速度 |
| 批量处理 | `batch` | 长音频和批量任务 |
| CPU 轻量 | `cpu` | 无独显或资源受限环境 |
| 中英技术 | `mixed_technical` | 中英混说、产品名、代码和缩写 |

CLI 入口：

```bash
integration api whisper --help
```

通用 CLI 参数为 `--addr URL`、`--port PORT`、`--output PATH` 和 `--pretty=true|false`。`create` 的 `--path` 可重复传入多个工作区相对路径。

#### `GET /api/whisper/check`

检查当前受控环境中的 `whisper` 命令是否可启动。成功时返回 `available: true`；缺失或不可用时返回 `available: false`、用于向当前会话发起安装请求的 `install`，以及说明文本 `content`。该接口不执行安装。

```bash
curl 'http://localhost:#port/api/whisper/check'
integration api whisper check
```

成功响应：

```json
{"available":true,"status":0}
```

#### `GET /api/whisper/tasks`

查询一个 Agent 的转写任务。`agentId` 必填；`status` 可选，支持 `all`（或省略）、`queued`、`running`、`completed`、`cancelled`、`failed`；`page` 从 1 开始。服务端固定每页返回 5 条，列表不含完整日志。

```bash
curl 'http://localhost:#port/api/whisper/tasks?agentId=demo-agent&status=running&page=1'
integration api whisper list --agentId demo-agent --status running --page 1
```

响应字段包括 `tasks`、`total`、`page`、`pageSize`（固定 `5`）和 `taskStatus`。每项任务包含 `id`、`sourcePath`、`outputPath`、`savedAs`（文字文件绝对路径）、`scenario`、`status`、`progress`、`startedAt`、`createdAt`、`updatedAt` 和 `cancelRequested`。

#### `POST /api/whisper/tasks`

向队列添加 1 至 64 个音频。每个路径必须是当前 Agent 工作目录内的音频文件；输出路径由服务端生成，已存在同名文字文件或任务输出时自动追加时间戳，绝不覆盖。推荐使用 `tasks`，让每个音频单独指定 `scenario`；未传场景时默认为 `chinese_meeting`。旧的 `paths` 请求仍可用，并可通过顶层 `scenario` 将同一场景应用到全部路径。

```bash
curl -X POST 'http://localhost:#port/api/whisper/tasks' \
  -H 'Content-Type: application/json' \
  -d '{"agentId":"demo-agent","tasks":[{"path":"audios/meeting.mp3","scenario":"chinese_meeting"},{"path":"audios/intro.m4a","scenario":"mixed_technical"}]}'

integration api whisper create \
  --agentId demo-agent \
  --path audios/meeting.mp3 \
  --scenario chinese_meeting
```

成功响应为 `{"status":0,"tasks":[...]}`。创建即进入串行队列；该接口不会等待转写完成。

#### 取消、重新开始与删除

这三个接口的 JSON 请求体相同：

```json
{"agentId":"demo-agent","id":12}
```

| 操作 | HTTP | 前置条件 | CLI |
|---|---|---|---|
| 取消 | `POST /api/whisper/tasks/cancel` | 任务为 `queued` 或 `running` | `integration api whisper cancel --agentId demo-agent --id 12` |
| 重新开始 | `POST /api/whisper/tasks/restart` | 任务为 `cancelled` | `integration api whisper restart --agentId demo-agent --id 12` |
| 删除记录 | `POST /api/whisper/tasks/delete` | 任务为 `failed`；不会删除音频或文字文件 | `integration api whisper delete --agentId demo-agent --id 12` |

取消正在执行的任务会终止对应 Whisper 子进程，并立即让队列调度下一条任务。重新开始会重新分配输出文件路径，并保留该任务原有的场景。执行日志会记录场景及本次实际使用的模型、语言、温度、搜索宽度、上下文和 `fp16` 参数。

HTTP 示例：

```bash
curl -X POST 'http://localhost:#port/api/whisper/tasks/cancel' \
  -H 'Content-Type: application/json' \
  -d '{"agentId":"demo-agent","id":12}'
```

成功时均返回：

```json
{"status":0}
```

#### `GET /api/whisper/tasks/log`

读取单个任务及其持久化执行日志。`agentId` 和正整数 `id` 均必填。正在执行的任务可轮询此接口获取最新日志。

```bash
curl 'http://localhost:#port/api/whisper/tasks/log?agentId=demo-agent&id=12'
integration api whisper log --agentId demo-agent --id 12
```

成功响应为 `{"status":0,"task":{...}}`；其中 `task.logs` 为按时间追加、且有大小上限的执行日志。

### rembg 图片主体提取

受控环境通过 `rembg i <source> <output>` 串行处理图片。任务、状态和日志保存在 Integration 数据库中；每个输出都使用透明背景 PNG，并写入相应 Agent 工作目录的 `images/`。图片须位于该 Agent 工作目录且经服务端图片检测通过，输出名称为 `<原文件名>_subject.png`，冲突时追加时间戳，绝不覆盖既有文件。

`GET /api/rembg/check` 从 `config/config.json.rembg` 读取检查缓存时长与安装请求文案；未找到可执行的 `rembg` 时返回该安装请求。任务接口与 Whisper 队列使用相同的 `status` / `content` 响应约定、状态值、分页规则及取消、重试、失败删除和日志查询语义。

CLI 入口：

```bash
integration api rembg --help
```

`create` 需要 `--agentId` 与至少一个可重复的 `--path`，并把同一个 `--model` 和 `--alpha-matting` 选择应用到本次提交的全部图片。支持的模型为 `u2net`（默认）、`u2net_human_seg`、`u2netp`、`u2net_cloth_seg`、`silueta`、`isnet-general-use` 和 `isnet-anime`。

```bash
# 检查依赖，以及按状态查询任务
integration api rembg check
integration api rembg list --agentId demo-agent --status running --page 1

# 提取单个主体并启用边缘优化
integration api rembg create \
  --agentId demo-agent \
  --path images/product.jpg \
  --model u2net \
  --alpha-matting

# 同一模型批量提交图片
integration api rembg create \
  --agentId demo-agent \
  --path images/portrait.jpg \
  --path images/logo.png \
  --model isnet-general-use

# 管理或查看单条任务
integration api rembg cancel --agentId demo-agent --id 12
integration api rembg restart --agentId demo-agent --id 12
integration api rembg delete --agentId demo-agent --id 12
integration api rembg log --agentId demo-agent --id 12
```

`cancel` 仅适用于 `queued` 或 `running`，`restart` 仅适用于 `cancelled`，`delete` 仅删除 `failed` 的任务记录；三者均不会删除源图片或已生成的 PNG。`list` 与 `log` 的筛选、分页和日志语义与 Whisper 对应命令相同。

### VoxCPM 文字转语音

VoxCPM 任务仅接受当前 Agent 工作区中的非空 UTF-8 文字文件；参考音色和表达风格均可不填。未提供参考音色时使用 `design` 自由生成声音，提供后使用 `clone` 克隆音色。输出固定写入当前 Agent 的 `audios/`；单条任务使用请求名称，批量任务和任何命名冲突会自动追加时间戳，不覆盖既有文件。

CLI 入口：

```bash
integration api voxcpm --help
```

`create` 需要 `--agentId`、一至五个可重复的 `--textPath` 和 `--outputName`。`--referenceAudioPath`、`--scenario` 与 `--control` 可以省略；提供一次时会应用到全部文字任务，提供与文字任务数相同的次数时则按顺序逐条对应。场景为 `balanced`（默认）、`quality`、`fast`、`clean`、`warm_narration` 或 `lively`。`--control` 优先于场景的内置表达风格。

```bash
# 检查依赖，以及按状态查询任务
integration api voxcpm check
integration api voxcpm list --agentId demo-agent --status running --page 1

# 不使用参考音色的自由生成
integration api voxcpm create \
  --agentId demo-agent \
  --textPath scripts/intro.txt \
  --outputName intro.wav \
  --scenario warm_narration

# 同一参考音色批量配音
integration api voxcpm create \
  --agentId demo-agent \
  --textPath scripts/part-1.txt \
  --textPath scripts/part-2.txt \
  --referenceAudioPath audios/voice.wav \
  --outputName narration.wav \
  --scenario quality \
  --control '温和、平稳、自然的叙述语气'

# 为两条任务分别指定场景和表达风格
integration api voxcpm create \
  --agentId demo-agent \
  --textPath scripts/ad.txt \
  --textPath scripts/news.txt \
  --outputName mixed.wav \
  --scenario lively \
  --scenario clean \
  --control '轻快、有活力' \
  --control '清晰、平稳'

# 管理或查看单条任务
integration api voxcpm cancel --agentId demo-agent --id 12
integration api voxcpm restart --agentId demo-agent --id 12
integration api voxcpm delete --agentId demo-agent --id 12
integration api voxcpm log --agentId demo-agent --id 12
```

`cancel` 仅适用于 `queued` 或 `running`，`restart` 仅适用于 `cancelled`，`delete` 仅删除 `failed` 的任务记录；三者都不会删除文字、参考音色或输出 WAV。`log` 返回包括实际 `design`/`clone` 模式、场景参数、表达风格和受控设备回退信息在内的持久化日志。

### `/api/shutdown`

按需求，不提供 CLI 包装。

HTTP：

```bash
curl -X POST 'http://localhost:#port/api/shutdown'
curl 'http://localhost:#port/api/shutdown'
```

限制：

- 仅允许 `localhost` / `127.0.0.1` / `::1`
- 返回接受成功后，大约 5 秒后触发关闭

## 备注

- `/mapping/...` 与 `server.Register(mux, cfg.Site)` 提供的是静态资源/页面服务，不额外包装为 CLI。
- `integration api ...` 是主命令；`integration service ...` 为兼容别名。
- 对于 `cron / connect / plugins / agent / sandbox / message-insert / host / standalone` 这类已有成熟 CLI 的能力，`integration api ...` 会尽量复用原命令行为，以保证帮助信息和输出语义稳定。
