# Integration 迭代 20260801-3 使用手册

## rembg 依赖检查

活动 `config/config.json` 需要提供图片主体提取配置：

```json
{
  "rembg": {
    "check": 24,
    "install": "请安装 rembg"
  }
}
```

`check` 是正整数小时，控制成功依赖检查的缓存时间；`install` 是缺少依赖时返回给页面、经用户确认后发送到当前 Chat 的安装请求。配置缺失或格式无效会返回错误，不会使用默认值，也不会由服务端执行安装。

`GET /api/rembg/check` 仅检查受控运行环境中的 `rembg` 命令能否启动。除当前 `PATH` 外，服务会发现用户的 Python Framework、pyenv、conda、venv、Poetry、asdf、mise 与常规用户安装目录；WSL 隔离任务会将相应 Python 运行时以只读方式继承。命令不可用时，页面收到安装请求而不会打开任务浮层。

## 图片主体任务

- `GET /api/rembg/tasks?agentId=<AgentId>&status=<状态>&page=<页码>` 查询当前 Agent 的任务。状态可为 `queued`、`running`、`completed`、`cancelled`、`failed` 或省略；每页固定 5 条。
- `POST /api/rembg/tasks` 接受 `{ "agentId": "...", "tasks": [{ "path": "images/photo.jpg", "model": "u2net", "alphaMatting": false }] }`。模型为空时默认通用模型 `u2net`；旧版 `paths` 请求继续按默认模型处理。路径必须是当前 Agent 工作区内经服务端验证的图片相对路径。
- `POST /api/rembg/tasks/cancel`、`/restart` 和 `/delete` 接受 `{ "agentId": "...", "id": 1 }`，分别用于取消排队或运行任务、重启已取消任务、删除失败或已取消任务记录。
- `GET /api/rembg/tasks/log?agentId=<AgentId>&id=1` 返回指定任务及其持久化执行日志。

任务记录存储在共享 SQLite 的 `rembg_task` 表中，按创建顺序在全部 Agent 间串行执行。服务重启时，未取消的运行任务会恢复为排队中；已完成、失败和取消任务继续保留历史记录。

服务端只会调用受控的 `rembg i -m <模型> [--alpha-matting] <source> <output>`。默认模型为通用的 `u2net`，也可选人物、轻量、服饰或动漫模型。模型与 alpha matting 开关会保存到每条任务；首次使用尚未缓存的模型时，rembg 会在任务过程中自动下载，下载的实时进度、失败原因都会写入任务日志。源图片不会被覆盖或删除；成功结果固定保存到对应 Agent 工作目录的 `images/`，文件名为 `<原文件名>_subject.png`。如存在同名文件或已分配的任务输出，系统会自动追加时间戳与序号，不覆盖现有文件。失败或取消时会清理尚未完成的输出文件。

任务执行日志、状态、进度与取消标记均会持久化。任务开始时进度为 `0`，仅在生成有效 PNG 后为 `100`；列表、日志、重试和失败删除均限于发起请求的 Agent。
