# 跨 Agent 上传与媒体预览服务手册

`POST /api/upload?agentId=<agentId>` 始终将文件写入指定 Agent 工作区的 `tmp/`。成功响应中的 `dest` 是该 `tmp/` 目录的绝对路径，`files` 则保留每个上传文件相对于 `tmp/` 的路径；调用方可组合两者得到稳定的附件绝对路径，同时保留上传文件夹的层级。

`GET /api/workspace?agentId=<agentId>` 返回该 Agent 的绝对工作区路径。Site 使用该信息将属于该工作区的附件绝对路径转换为相对路径后，再调用 `GET/HEAD /api/media_preview?agentId=<agentId>&path=<relativePath>`。

媒体预览接口仍只允许指定 Agent 工作区内的相对音视频文件。绝对路径、`..` 逃逸、跨 Agent 路径、目录、非媒体文件和逃出工作区的符号链接都会被拒绝；接口不会因跨 Agent 预览而获得读取任意本机文件的权限。
