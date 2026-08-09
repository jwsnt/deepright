# CLI-Get 回显元数据

`/cli/get` 任务的 `subOps.echo` 是页面恢复 CLI 子面板时使用的展示元数据。`false` 仅表示该任务在恢复面板中不显示；字段缺失时保持既有默认展示语义。

独立 cli-get 不会因 `echo=false` 改变任务队列、执行、沙盒、重试或 `cli/pub` 发布。发布结果继续携带原任务的 `chat` 与 `tid`，使页面能够按 `chatId + tid` 完成恢复关联。

## 相关文档

- [主手册](../../USER_GUIDE.md)
- [需求说明](REQUIREMENT.md)
