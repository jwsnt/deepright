# 20260503-1 使用手册

## 简介

本次迭代为 Proxy 模块增加了直接创建任务元数据的 CLI 能力，并要求与原有 HTTP 服务模式兼容。

本次完成后，`proxy` 既可以：

- 直接启动 HTTP 代理服务
- 通过命令行创建周期任务
- 通过命令行创建自定义 Cron 任务

## 命令总览

查看帮助：

```bash
./proxy --help
```

默认服务模式：

```bash
./proxy --agent-dir ./agents
```

显式服务模式：

```bash
./proxy serve --agent-dir ./agents --port 9876
```

创建周期任务：

```bash
./proxy create --content "每15分钟检查一次上游接口健康" --model "OpenAI" --thinking true --rawTime "2026-05-03 10:00" --cycle 4 --chatId "chat-001" --agent "A"
```

创建自定义 Cron 任务：

```bash
./proxy create-cron --cron='10 12 * * 1-5' --agent=A --chat=chat-001 --model=OpenAI --content='查看天气'
```

兼容旧命令：

- `submit` 等同 `create`
- `submit-cron` 等同 `create-cron`

## 参数说明

### 服务参数

- `--agent-dir=DIR`
  - Agent 根目录
  - 服务启动必填
- `--port=INT`
  - 监听端口
  - 默认 `9876`
- `--host=URL`
  - 上游服务地址
  - 默认 `http://127.0.0.1:9998`
- `--device=ID`
  - 设备 ID
  - 可为空
- `--agent-cache=MS`
  - Agent 元数据缓存毫秒数
- `--site=DIR`
  - 静态站点目录
  - 默认当前目录下 `./site`
- `--connect_timeout=MS`
  - 上游连接超时毫秒数

### `create`

- `--cycle=INT`
  - `0=仅一次`
  - `1=工作日`
  - `2=自然日`
  - `3=每小时`
  - `4=每15分钟`
  - `5=每30分钟`
- `--time='YYYY-MM-DD HH:MM'`
  - 首次开始时间
  - 必填
- `--agent=ID`
  - AgentId
  - 必填
- `--chat=ID` / `--chatId=ID`
  - 会话 ID
  - 可为空
- `--model=NAME`
  - 模型名称
  - 必填
- `--thinking`
  - 带上即表示 `true`
- `--content='TEXT'`
  - 任务内容
  - 必填

### `create-cron`

- `--cron='EXPR'`
  - 自定义 Cron 表达式
  - 必填
- `--agent=ID`
  - AgentId
  - 必填
- `--chat=ID` / `--chatId=ID`
  - 会话 ID
  - 可为空
- `--model=NAME`
  - 模型名称
  - 必填
- `--thinking`
  - 带上即表示 `true`
- `--content='TEXT'`
  - 任务内容
  - 必填

## 行为说明

### HTTP 与 CLI 共用逻辑

- `proxy create`
- `proxy create-cron`
- `POST /api/cron/create`

以上三种入口最终都复用同一套任务元数据创建逻辑，保证：

- 参数校验一致
- `task_meta` 写入一致
- `chatId` 写入一致
- 高频周期任务的明细展开一致
- Agent 存在性校验一致
- 模型是否已在 `/api/token` 注册且 token 非空的校验一致
- 未显式传入的非 cron 模块必填参数会优先从启动目录下的 `runtime.json` 读取

### CHAT_ID

- 支持 `--chat` 和 `--chatId`
- `chatId` 会写入 `task_meta`
- 自动创建的 `task_detail` 会继承相同 `chatId`
- `agent-dir` 与 `device` 等非 cron 模块参数，在未显式传入时会优先读取 `runtime.json`

### 高频周期

- `cycle=3`：按小时展开未来 5 天窗口
- `cycle=4`：按 15 分钟展开未来 5 天窗口
- `cycle=5`：按 30 分钟展开未来 5 天窗口

### 自定义 Cron

- 使用 `cycle=-1 + cron`
- 创建元数据时不立即展开任务明细
- 后续由检查逻辑继续补齐未来时间窗口

## 输出结果

创建成功时会输出 JSON：

```json
{
  "status": 0,
  "id": 1,
  "cron": "10 12 * * 1-5",
  "agentId": "A"
}
```

## 编译

```bash
cd cli/module/proxy
/opt/homebrew/bin/go build -o proxy ./
```

## 对齐说明

- CLI 名称使用 `proxy`，未复用 `cron` 命令名
- 原有 HTTP 服务能力保留，没有破坏旧启动方式
- 新 CLI 和 HTTP 接口兼容共存
