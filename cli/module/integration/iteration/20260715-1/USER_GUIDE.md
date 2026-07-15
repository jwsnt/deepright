# Integration 迭代手册（20260711-2）

## 本次更新

- `integration` 在转发 `/v1/chat/completions` 时，会统一补充顶层 `metadata.plugins_dir`
- `integration` 发往上游的 `/cli/get` 心跳请求，也会补充顶层 `metadata.plugins_dir`
- 内部 `memo`、`email`、`feishu` 等任务最终转发到上游 `/v1/chat/completions` 时，同样会补充顶层 `metadata.plugins_dir`
- 字段值优先按 integration 当前运行时目录规则解析真实插件目录；只有运行时目录不可得时，才回退到进程环境变量 `DEEPRIGHT_PLUGIN_DIR`
- 外部请求里的旧插件目录字段不会继续向下游透传；最终只保留 `metadata.plugins_dir`

## 字段说明

- `metadata.plugins_dir` 表示当前 `integration` 运行时使用的插件二进制目录绝对路径
- 该字段位于顶层 `metadata`，与 `knowledge`、`sandbox_path` 同层
- 该字段不进入 `metadata.agent` 或 `metadata.agents[]`

## 示例

转发到上游的请求体 `metadata` 片段示例：

```json
{
  "metadata": {
    "agentId": "A",
    "chat": "chat-001",
    "plugins_dir": "/Users/demo/Library/Application Support/deepright/plugins"
  }
}
```

## 兼容性说明

- 原有 `metadata.plugins` 仍表示“当前检测到正在运行的插件 key 列表”，不变
- 本次新增的 `metadata.plugins_dir` 表示“当前 integration 使用的插件目录路径”，用于让下游服务拿到本机插件运行时目录
