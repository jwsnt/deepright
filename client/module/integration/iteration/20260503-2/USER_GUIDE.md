# 20260503-1 使用手册

## 简介

本次迭代为 Integration 模块增加了 `cron create` CLI 命令，使其在统一命令上同时兼容：

- 启动 HTTP 服务
- 通过 CLI 创建备忘录（定时任务）元数据

## 命令示例

启动 HTTP 服务：

```bash
./integration --agent-dir ./agents --port 8080 --host http://127.0.0.1:9998
```

创建周期任务：

```bash
./integration cron create --content "每15分钟检查一次上游接口健康" --model "OpenAI" --thinking true --rawTime "2026-05-03 10:00" --cycle 4 --chatId "chat-001" --agent "demo-agent"
```

创建自定义 Cron 任务：

```bash
./integration cron create-cron --content "工作日中午整理异常日志" --model "OpenAI" --thinking true --cron "10 12 * * 1-5" --chatId "chat-001" --agent "demo-agent"
```

## 参数说明

### `integration cron create`

- `--content TEXT`
  - 任务内容
  - 必填
- `--model NAME`
  - 模型名称
  - 必填
- `--thinking`
  - 不带值时等于 `true`
- `--thinking true` / `--thinking false`
  - 也支持显式布尔值
- `--rawTime 'YYYY-MM-DD HH:MM'`
  - 首次开始时间
  - 必填
  - 也支持 `--time`
- `--cycle INT`
  - `0=仅一次`
  - `1=工作日`
  - `2=自然日`
  - `3=每小时`
  - `4=每15分钟`
  - `5=每30分钟`
- `--chatId ID`
  - 会话 ID
  - 可为空
  - 也支持 `--chat`
- `--agent ID`
  - AgentId
  - 必填
  - 也支持 `--agentId`
- `--agent-dir DIR`
  - Agent 根目录
  - 优先级高于环境变量 `AGENT_DIR` 和默认 `./agents`

### `integration cron create-cron`

- `--content TEXT`
  - 任务内容
  - 必填
- `--model NAME`
  - 模型名称
  - 必填
- `--thinking`
  - 不带值时等于 `true`
- `--thinking true` / `--thinking false`
  - 也支持显式布尔值
- `--cron EXPR`
  - 自定义 Cron 表达式
  - 必填
- `--chatId ID`
  - 会话 ID
  - 可为空
  - 也支持 `--chat`
- `--agent ID`
  - AgentId
  - 必填
  - 也支持 `--agentId`
- `--agent-dir DIR`
  - Agent 根目录
  - 优先级高于环境变量 `AGENT_DIR` 和默认 `./agents`

## 行为说明

- `integration` 默认不带子命令时，仍按原方式启动统一 HTTP 服务
- `integration cron create` 与 `POST /api/cron/create` 共用同一套任务元数据创建逻辑
- `chatId` 会写入 `task_meta`，并传递到 `task_detail`
- 创建时会检查指定 Agent 是否存在
- `--agent-dir` 既支持传 Agent 根目录，也支持直接传某个具体 Agent 目录
- 未显式传入的非 cron 模块必填参数会优先从启动目录下的 `runtime.json` 读取，当前主要用于补全 `agent-dir` 与 `device`
- 创建时还会检查指定模型是否已在 `/api/token` 注册，且 token 非空
- 高频周期 `cycle=3/4/5` 会在创建时立即展开未来 5 天窗口内的任务明细
- 自定义 Cron 使用 `cycle=-1 + cron`，创建元数据时不立即展开首批明细
- 命令行参数同时兼容 `--key=value` 与 `--key value`

## 输出结果

成功时输出 JSON：

```json
{
  "status": 0,
  "id": 1,
  "cron": "*/15 * * * *",
  "agentId": "demo-agent"
}
```

## 编译

```bash
cd cli/module/integration
/opt/homebrew/bin/go build -o integration ./
```
