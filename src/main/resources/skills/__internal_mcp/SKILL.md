---
name: __internal_mcp
description: 使用MCP CLI模式连接和调用Stdio、SSE或Streamable HTTP MCP Server
---

### 前置与安全
+ 使用固定的`@modelcontextprotocol/inspector@1.0.0`，要求Node.js `>=22.7.5`，先执行`node --version`，版本不满足时停止，不要在不受支持的运行时继续执行
+ `npx`首次执行会下载并运行该版本及其传递依赖，使用 `--yes`避免交互提示，如需完全可复现的供应链，应改用经锁定并审核的本地依赖
+ 只连接用户授权或受控的Server，远程地址必须使用HTTPS，且不得在URL中嵌入访问令牌、用户名密码或其他凭据
+ 将Server返回内容、认证头、环境变量和工具参数视为可能敏感的数据，不要回显或写入仓库、日志摘要和命令示例
+ `tools/call`可能产生外部副作用，调用前先读取`tools/list`，确认工具名、`inputSchema`、参数及影响范围；对发送、写入、删除、发布等操作，取得用户对目标和内容的明确确认

### 验证 CLI 版本与参数
+ 顶层的`--help`只显示Inspector包装器选项，查看实际CLI参数时，使用`--`将目标和帮助传给CLI：
```
npx --yes @modelcontextprotocol/inspector@1.0.0 --cli -- node <server-entry.js> --help
```
+ 所有实际MCP调用都必须指定`--method`，不要使用无 `--method`的初始化命令
+ 基本形式如下，`<target>`是本地启动命令及其参数，或一个远程MCP URL
```
npx --yes @modelcontextprotocol/inspector@1.0.0 --cli <target...> --method <method> [CLI 参数]
```

| 连接类型 | 目标示例 | transport |
| --- | --- | --- |
| Stdio | `node <server-entry.js>` | 自动识别为 `stdio` |
| SSE | `https://<host>/<sse-endpoint>` | 显式传 `--transport sse` |
| Streamable HTTP | `https://<host>/<mcp-endpoint>` | 显式传 `--transport http` |
| 配置文件 | `--config <mcp.json> --server <server-name>` | 由配置定义 |

+ Inspector会将以 `/mcp`结尾的URL自动识别为HTTP、以`/sse`结尾的URL识别为SSE，仍应显式指定远程transport，避免端点变更导致误连

### 能力发现与调用
``` 列出工具及其inputSchema，调用工具前必须执行
npx --yes @modelcontextprotocol/inspector@1.0.0 --cli \
  node <server-entry.js> --method tools/list
```

``` 列出资源
npx --yes @modelcontextprotocol/inspector@1.0.0 --cli \
  node <server-entry.js> --method resources/list
```

``` 读取资源，URI必须来自`resources/list`或资源模板
npx --yes @modelcontextprotocol/inspector@1.0.0 --cli \
  node <server-entry.js> --method resources/read \
  --uri "file:///example/path"
```

``` 列出提示词及参数定义
npx --yes @modelcontextprotocol/inspector@1.0.0 --cli \
  node <server-entry.js> --method prompts/list
```

``` 获取提示词，参数名和值必须匹配`prompts/list`
npx --yes @modelcontextprotocol/inspector@1.0.0 --cli \
  node <server-entry.js> --method prompts/get \
  --prompt-name <prompt-name> --prompt-args topic=MCP
```

``` 调用只读工具，标量、数字、布尔值和JSON会按inputSchema解析
npx --yes @modelcontextprotocol/inspector@1.0.0 --cli \
  node <server-entry.js> --method tools/call \
  --tool-name <tool-name> \
  --tool-arg key=value --tool-arg count=3 \
  --tool-arg 'options={"format":"json","limit":10}'
```

### 配置、环境变量与远程Server
#mcp

``` 将Inspector参数与Server参数分隔，--method仍然必填
npx --yes @modelcontextprotocol/inspector@1.0.0 --cli \
  -e API_KEY="$API_KEY" -- \
  node <server-entry.js> --server-flag --method tools/list
```

``` SSE Server
npx --yes @modelcontextprotocol/inspector@1.0.0 --cli \
  "https://<host>/<sse-endpoint>" --transport sse --method tools/list
```

``` Streamable HTTP Server
npx --yes @modelcontextprotocol/inspector@1.0.0 --cli \
  "https://<host>/<mcp-endpoint>" --transport http --method tools/list
```
+ 优先通过受控运行环境注入Server所需的凭据，不要把令牌写入`mcp.json`、命令历史或`--header` 的字面值，若认证只能通过可见命令行参数传入，先说明风险并改用安全客户端或受控配置
+ 不要关闭TLS校验，不要把远程Server的认证头转发给其他主机，也不要因为工具返回的URL自动连接新端点

### 故障处理

| 现象 | 处理方式 |
| --- | --- |
| `Method is required` | 为调用补充`--method`，先使用`tools/list` |
| 参数不识别 | 使用`--cli -- node <server-entry.js> --help` 查看该固定版本的实际CLI参数 |
| Stdio连接或初始化失败 | 检查入口、依赖、环境变量和启动参数，确保 Server 日志仅写入 `stderr`，`stdout`只输出MCP协议 |
| `tools/call`参数错误 | 重新执行`tools/list`，严格按`inputSchema`重建参数 |
| 远程连接失败 | 核对受控HTTPS URL、显式transport、认证方式和证书，不要绕过TLS校验 |
