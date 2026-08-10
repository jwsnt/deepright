# 迭代 20260729-1：飞书插件凭证组与 MCP 启动

## 需求目录

- 本迭代前端需求：[REQUIREMENT.md](REQUIREMENT.md)
- Site 主需求：[../../REQUIREMENT.md](../../REQUIREMENT.md)
- Site 主手册：[../../USER_GUIDE.md](../../USER_GUIDE.md)
- 服务端需求：[../../connect/feishu/iteration/20260729-1/REQUIREMENT.md](../../connect/feishu/iteration/20260729-1/REQUIREMENT.md)

## 页面配置与启动

飞书插件参数保持为 `appId`、`appSecret`、`mcp_url`。页面始终通过运行时 `key=feishu` 识别并提交飞书插件，不会使用展示名称替代该 key。

- `appId` 与 `appSecret` 是同一凭证组，必须同时填写。
- 完整凭证组可单独启动，也可同时填写 `mcp_url`；服务端会建立飞书长连接。
- 仅填写 `mcp_url` 可以启动或重启。页面显示启动成功，但服务端不会建立飞书长连接。
- 仅填写一个凭证时，页面会提示“appId 和 appSecret 必须同时填写”。
- 三项都为空时，页面会提示填写完整凭证组或 `mcp_url`，且不会提交配置或启动请求。

其它插件不受此校验影响。飞书 MCP 文档图标、参数回填、会话绑定、日志查看和既有启动／关闭交互保持不变。
