# CLI 子面板实时与恢复展示

聊天 SSE 正常存活时，右侧 CLI 子面板只显示 SSE 中的实时 `cli/sub` 回显。切换回该会话只会重绘已接收的内存内容，不会为 CLI 子面板请求恢复数据。

页面刷新、重新进入会话等恢复场景中，非 `cli/sub` 的聊天 SSE 内容会继续填充居中对话；`cli/sub` 不会进入居中 SSE 或思考片段。子面板改由 `/api/restore` 返回的 `cli/get` 与 `cli/pub` 重建。

恢复时沿用实时 SSE 的数据包路由：`biz=cli && workflow=sub` 只属于 CLI 子面板；`workflow=__sub` 等思考片段仍显示在居中对话/思考展示中。两者不会相互覆盖或重复显示。

恢复任务按 `chatId + tid` 关联，并会记住该任务的显示决定以处理后续恢复批次。只有 `cli/get.subOps.echo` 明确为 `false` 时，该任务及其结果不显示在子面板；缺失、为空或为 `true` 时正常显示。该字段只影响显示：任务仍会执行，`cli/pub` 仍会回传并保存。无法找到同一 `chatId + tid` 任务的结果不会显示到其他命令。

## 相关文档

- [主手册](../../USER_GUIDE.md)
- [需求说明](REQUIREMENT.md)
