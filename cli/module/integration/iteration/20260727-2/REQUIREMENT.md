### 第一性原则
+ 仅可以新增/更新/删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../build.sh`、`../../../build/install.ps1`、`../../../build/USER_GUIDE.md`、`../../../build/USER_GUIDE.txt` 与 `../../../config/app/API.md`、`../../../config/app/CANVAS.md`、`../../../config/app/DESIGN.md`。

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Integration介绍：../../REQUIREMENT.md
+ Integration手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ `POST /api/upload?agentId=...` 上传到指定 Agent 的 `tmp/` 后，成功响应中的 `dest` 必须是经本机路径解析后的绝对工作区路径。响应 `files` 继续保持上传相对路径，以保留文件夹结构；调用方通过 `dest + files` 得到稳定的附件绝对路径。
+ Integration 必须先按 `agentId` 解析并验证 Agent 工作区，再创建 `tmp/` 和写入上传文件。不得以进程当前目录、请求中的相对路径或前端当前 Agent 推断上传目标。
+ `GET /api/workspace?agentId=...` 继续返回该 Agent 的绝对工作区路径，供 Site 将绝对附件路径安全换算为工作区相对路径；Agent 无效时保持既有失败语义。
+ `GET/HEAD /api/media_preview?agentId=...&path=...` 继续只接受指定 Agent 工作区内的相对媒体文件路径，继续拒绝绝对路径、路径逃逸、跨 Agent 文件和解析后逃出工作区的符号链接。不得为修复跨 Agent 预览而放宽该边界。
+ 上传、工作区查询和媒体预览共同保证：同名媒体分别位于 Agent A、Agent B 时，使用 A 的 `agentId` 和 A 工作区相对路径只能读取 A 的文件。

### 编写代码
+ 最小范围更新，不新增外部依赖

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`，说明上传响应的绝对 `dest`、相对 `files`、工作区解析方式与媒体预览的 Agent 工作区安全边界。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求，不记录实现过程。
