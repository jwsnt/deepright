# Integration：上游客户端版本与页面主题

Integration 启动时从实际生效的主 `config/config.json` 读取并缓存 `version`。内置 `/cli/get` 心跳，以及普通会话、备忘录/Connect 任务和设置页模型测试发出的上游 `/v1/chat/completions` 请求，都会在顶层 metadata 中携带该客户端版本：

```json
{
  "metadata": {
    "version": "0.1"
  }
}
```

该字段由 Integration 转发前写入并覆盖调用方同名值，不会读取 Agent 工作目录的 `config.json`。版本在进程启动后保持缓存；若主配置随后修改，需要重启 Integration 才会使用新版本。`/cli/pub` 与独立 `cli-get` 程序不携带此字段。

普通页面会话与设置页模型测试还会在每次请求的顶层 metadata 携带页面参数：冷色模式为 `theme: "cold"`，暖色模式为 `theme: "warm"`。它不缓存、不持久化；Integration 只原样转发普通会话的主题。模型测试除主题外还必须携带非空的 `device`，且主题只能是 `cold` 或 `warm`。备忘录、Connect 任务、内置 `/cli/get` 和 `/cli/pub` 不携带 `theme`。

上游返回 HTTP 或 SSE 业务码 `400` 时，普通会话和模型测试会显示“服务商请求无效（400），请检查模型地址、模型名称与请求参数”，而不会直接展示服务商原始错误内容。

对应需求见 [REQUIREMENT.md](REQUIREMENT.md)。
