# 迭代 20260724-1：按 Agent 注入 MCP 配置

Integration 会按 Agent 工作目录检查 MCP 配置，并在可用时向上游请求声明配置文件的系统绝对路径。MCP 文件内容不会离开本机。

## 配置

在应用资源目录的 `config/config.json` 中配置检查间隔和文件名：

```json
{
  "mcp": {
    "interval": 30,
    "file": "mcp.json"
  }
}
```

- `interval` 必须是正整数，单位为秒。
- `file` 必须是 Agent 工作目录内的相对路径；例如 `mcp.json` 或 `config/mcp.json`。
- Integration 首次运行和之后每个间隔都会检查全部 Agent 工作目录。配置变更会在下一次检查时生效。

例如 Agent `X` 的工作目录为 `/path/to/agents/X` 时，默认配置检查的是 `/path/to/agents/X/mcp.json`。

## 文件判定与转发

当当前请求属于 Agent `X`，且其 MCP 文件是含至少一个元素的 JSON 对象或数组时，Integration 会在最终上游请求中加入：

```json
{
  "metadata": {
    "mcp": "/path/to/agents/X/mcp.json"
  }
}
```

传递的是本机 MCP 文件的系统绝对路径，不会传递 JSON 文件内容。客户端自行提交的 `metadata.mcp` 会被丢弃，并由本机检查结果覆盖。页面错误提示仍只显示相对路径。

| MCP 文件状态 | `metadata.mcp` | 页面提示 |
| --- | --- | --- |
| 不存在、空白、`{}`、`[]` | 不添加 | 无 |
| 非空 JSON 对象或数组 | 添加系统绝对路径 | 无 |
| JSON 语法错误，或 `null`、字符串、数字、布尔值等 JSON 标量 | 不添加 | 显示 MCP 解析错误 |

MCP 解析错误在左侧工作区中以与 `SKILL.md` 解析错误相同的红色标记和错误浮层展示；它使用独立的 `/mcp_warning` 接口和状态，不会混入 `/skills_warning`。文件在下一轮检查中变为有效、空白、空对象/数组或被删除后，提示会自动消失。

## 覆盖请求

以下请求都会使用相同的 Agent MCP 检查结果：

- 普通页面对话的 `/v1/chat/completions`
- 备忘录定时任务
- 飞书插件任务
- 邮件插件任务
- Integration 主动发送的 `/cli/get`
