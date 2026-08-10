# Bubblewrap 依赖检查服务手册

Integration 提供 `GET /api/bubblewrap/check`，供 Site 在打开会话沙盒浮层前检查 WSL/Linux 的 `bwrap` 命令。接口不执行安装，也不变更任何会话沙盒状态。

Linux/WSL 环境读取主应用资源目录 `config/config.json` 中的 `bubblewrap` 配置：`check` 必须是正整数小时，`install` 必须是非空字符串。接口只缓存成功查找到 `bwrap` 的时间，缓存时长由 `check` 决定；失败结果不缓存，Integration 重启后成功缓存自动失效。

缺少 `bwrap` 时，接口返回 `available: false` 以及配置中的 `install` 文本。Site 在用户确认后将该文本发送至当前 Chat，由任务处理安装；Integration 和浏览器都不会直接运行安装命令。非 Linux 运行环境返回不需要 Bubblewrap，因此不读取该配置也不执行命令查找。
