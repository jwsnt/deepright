# 迭代说明

本次迭代为 `cli-get` 与 `integration` 内置 `cli-get` 增加了本地有界任务队列、独立发布队列、`ddl` 出队校验，以及 `/cli/pub` 失败重试能力。

## 行为变更

- `cli-get` 主链路从原先的：

```text
cli/get -> worker execute -> cli/pub
```

改为：

```text
cli/get -> taskQueue -> execute workers -> publishQueue -> cli/pub
```

- 心跳线程不会再因为执行 Worker 不够而阻塞
- 每次发 `/cli/get` 前会先检查本地 `taskQueue` 是否还有空位
- 当 `taskQueue` 已满时：
  - 不会继续发 `/cli/get`
  - 会按 `--sleep` 等待后重试
- 当 `/cli/get` 返回任务时：
  - 任务会立刻进入本地 `taskQueue`
  - 心跳线程马上继续下一轮，不等待执行完成

## 新增参数

- `--queue`
  - 本地任务队列容量
  - 默认 `1000`
- `--retry_interval`
  - `/cli/pub` 失败后的下一次重试等待时间（毫秒）
  - 默认 `10000`
- `--retry_times`
  - `/cli/pub` 首次失败后允许额外重试的次数
  - 默认 `1`

示例：

```bash
./cli-get --agent-dir ./agents --thread 20 --queue 1000 --retry_interval 10000 --retry_times 1
```

## `ddl` 过期处理

- 任务响应中的 `ddl` 表示执行截止时间戳（毫秒）
- `cli-get` 不会只在收到任务时校验一次 `ddl`
- Worker 真正从 `taskQueue` 取任务准备执行前，会再次检查当前时间是否已经超过 `ddl`
- 如果已经超过：
  - 任务直接丢弃
  - 不执行命令
  - 不进入 `/cli/pub`
  - 日志会打印 `agentId`、`chat`、`tid`、`ddl`、当前时间和 `cmd`

## 发布重试

- 执行结果会先进入本地 `publishQueue`
- 独立发布 Worker 负责提交 `/cli/pub`
- 如果出现以下情况，会进入重试：
  - 网络错误
  - 请求超时
  - HTTP Status Code 非 `200`
  - 响应解析失败
  - 响应中的 `code != 200`
- 重试时会等待 `--retry_interval`
- 超过 `--retry_times` 后会放弃，并打印失败日志

## 语义说明

- `taskQueue` 与 `publishQueue` 都是纯内存队列，不做持久化恢复
- 进程退出、崩溃、重启后，尚未执行或尚未发布的任务允许丢失
- 发布重试允许同一 `tid` 因超时或网络异常而重复推送
- 排队中的任务不支持取消；只有已经开始执行并注册到活跃命令表的任务仍可被 `/api/kill` 终止

## 已验证

已执行：

```bash
cd /path/to/deepright/cli/module/cli-get
go test ./...
```

以及：

```bash
cd /path/to/deepright/cli/module/integration
go build ./...
```
