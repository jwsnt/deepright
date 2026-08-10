# CLI 恢复记录与回显元数据

`/api/restore` 保留 `cli/get` 与 `cli/pub` 的原始记录，供页面恢复 CLI 子面板。任务的 `chatId`、`tid` 及 `subOps.echo` 均随既有协议保留。

`subOps.echo=false` 只表示恢复态页面不显示该任务；Integration 仍照常调度任务、接收 `cli/pub`、写入日志并转发执行结果。页面使用 `chatId + tid` 关联任务与结果，服务端不会跨会话合并结果。

## 相关文档

- [主手册](../../USER_GUIDE.md)
- [需求说明](REQUIREMENT.md)
