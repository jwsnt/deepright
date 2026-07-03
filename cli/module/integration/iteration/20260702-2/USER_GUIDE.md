# 迭代说明

本次迭代为 `integration` 增加了每个 Agent 工作目录下 `Knowledge.md` / `knowledge.md` 的透传能力。

## 新增行为

- 当 `integration` 转发 `/v1/chat/completions`、`/cli/get`，以及 integration 内部发起 cron 聊天请求时，会实时读取每个 Agent 工作目录下同级的 `Knowledge.md`
- 如果找不到 `Knowledge.md`，则会回退读取 `knowledge.md`
- 如果该文件存在，则会把文件内容写入对应的 `metadata.agents[].knowledge`
- 每个 Agent 只读取自己工作目录下的 `Knowledge.md` / `knowledge.md`，不会复用其他 Agent 的文件内容

## 目录约定

- `Knowledge.md` 与 Agent 工作目录下的 `SOUL.md`、`USER.md` 同级；兼容回退读取 `knowledge.md`
- 文件不存在时，不会输出空的 `knowledge` 字段
- 文件内容按原样透传，不额外做格式转换
