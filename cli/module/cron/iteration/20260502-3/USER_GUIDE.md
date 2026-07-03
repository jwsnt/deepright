# 20260502-3 使用手册

## 简介

本迭代为 Cron 模块补充了可直接创建任务元数据的 CLI（命令行）工具，并完善了 `--help`、`CHAT_ID` 透传以及高频周期任务的明细生成逻辑。

## CLI 命令

查看帮助：

```bash
./cron --help
```

创建周期任务：

```bash
./cron create --agent-dir ../agent/test-case --cycle=0 --time='2026-04-30 12:10' --agent=A --chat=chat-001 --model=OpenAI --thinking --content='查看天气'
```

需求案例风格也已兼容：

```bash
./cron create --agent-dir ../agent/test-case --content "每15分钟检查一次上游接口健康" --model "OpenAI" --thinking true --rawTime "2026-05-03 10:00" --cycle 4 --chatId "chat-001" --agent "A"
```

创建自定义 Cron 任务：

```bash
./cron create-cron --cron='10 12 * * 1-5' --agent=A --chat=chat-001 --model=OpenAI --thinking --content='查看天气'
```

兼容旧命令：

- `submit` 等同 `create`
- `submit-cron` 等同 `create-cron`

## 参数说明

- `--cycle=INT`
  - `0=仅一次`
  - `1=工作日`
  - `2=自然日`
  - `3=每小时`
  - `4=每15分钟`
  - `5=每30分钟`
- `--time='YYYY-MM-DD HH:MM'`
  - 首次开始时间
  - `create` 必填
- `--rawTime='YYYY-MM-DD HH:MM'`
  - 与 `--time` 等价
- `--agent=ID`
  - 绑定的 AgentId
  - 必填
- `--agentId=ID`
  - 与 `--agent` 等价
- `--chat=ID` / `--chatId=ID`
  - 绑定的会话 ID（CHAT_ID）
  - 可为空
- `--model=NAME`
  - 选择的模型
  - 必填
- `--thinking`
  - 带上即表示深度思考为 `true`
- `--thinking true` / `--thinking false`
  - 也支持显式传布尔值
- `--cron='EXPR'`
  - 自定义 Cron 表达式
  - `create-cron` 必填
- `--content='TEXT'`
  - 任务内容
  - 必填

## 行为说明

### CHAT_ID

- CLI 创建任务元数据时支持显式传入 `CHAT_ID`
- `chatId` 会写入 `task_meta`
- 后续自动创建的 `task_detail` 会继承同一个 `chatId`
- 创建时会检查指定 Agent 是否存在
- `--agent-dir` 既支持传 Agent 根目录，也支持直接传某个具体 Agent 目录
- 创建时会检查指定模型是否已在共享 SQLite `token_store` 中注册，且 token 非空

### 明细生成

- `仅一次`：立即创建一个任务明细
- `工作日 / 自然日`：立即创建后 5 天窗口内的任务明细
- `每小时 / 每15分钟 / 每30分钟`：立即创建后 5 天窗口内按间隔展开的全部任务明细
- `自定义 Cron`：提交时不立即创建，由每分钟周期检查补齐后续 5 天窗口内的明细

### 周期检查

- 每分钟仍会扫描所有非一次性任务
- 如果后续 5 天窗口内有尚未创建的任务明细，则自动补齐
- 如果 `(meta_id, exec_time)` 已存在，则跳过，避免重复创建

## 对齐说明

- CLI 创建出来的任务元数据与其他创建方式使用同一套 Cron 表达式生成和任务明细展开逻辑
- `CHAT_ID` 行为与其他模块链路保持一致
- 同时兼容 `--key=value` 与 `--key value` 两种命令行传参方式
- Agent 根目录优先取 `--agent-dir`，其次取环境变量 `AGENT_DIR`，最后回退当前目录下的 `./agents`
