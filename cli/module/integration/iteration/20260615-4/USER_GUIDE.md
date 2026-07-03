# 20260615-4 WSL 默认运行目录

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
