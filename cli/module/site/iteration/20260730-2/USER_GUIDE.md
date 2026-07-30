# 迷你应用参考文档保护使用手册

`__internal_miniapp` 使用当前 Agent `app/` 目录中的 `API.md`、`CANVAS.md`、`DESIGN.md` 作为只读参考资料。制作或更新迷你应用时，不能新增、修改、覆盖、移动或删除这三份文件；HTML、CSS、JavaScript、图片等其它迷你应用资源仍按既有流程放在 `app/` 内。

页面访问 Integration API 时应优先写为同源相对路径，例如 `fetch('/api/files?...')`；如必须组合完整 URL，应基于 `location.origin`。不要从文档示例复制 `localhost:#port` 后改成固定端口。这样通过 `/mapping/<agentId>/...` 打开的页面会自动使用当前 Integration 实际端口。

文档的服务端恢复周期由主应用 `config/config.json.miniapp.recover` 控制。页面不显示或保存该周期；运行配置接口仍以只读方式向 Site 提供完整 `miniapp` 对象。
