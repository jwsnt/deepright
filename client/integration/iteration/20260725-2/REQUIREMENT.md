### 第一性原则
+ 仅可以新增/更新/删除integration（../..）同目录及其子目录下的文件和文件夹

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
+ 新增同源媒体预览接口：`GET /api/media_preview?agentId=<agentId>&path=<relativePath>`，用于 Site 预览当前 Agent 工作目录中的音频和视频文件。
+ 接口仅接受 `GET` 与 `HEAD`，`agentId` 和 `path` 均为必填。
+ `path` 必须是当前 `agentId` 工作目录内的非空相对文件路径；拒绝绝对路径、`..` 逃逸路径、目录及不存在的文件。不得因为 `path` 参数读取其他 Agent 或任意本机路径。
+ 接口仅允许媒体扩展名：
    + 视频：`mp4`、`m4v`、`webm`、`ogv`、`mov`、`mkv`、`avi`、`mpg`、`mpeg`
    + 音频：`mp3`、`m4a`、`aac`、`wav`、`flac`、`ogg`、`opus`
+ 成功时按文件扩展名返回准确的媒体 `Content-Type`，设置 `Content-Disposition: inline` 与 `X-Content-Type-Options: nosniff`，并使用标准 HTTP 内容服务支持 `Range` 请求、`206 Partial Content`、`Content-Length`、`Last-Modified` 和 `HEAD`。
+ 非法方法返回 `405`；参数或路径不合法、非媒体文件及目录返回 `400`；不存在的 Agent 或文件返回 `404`；文件打开失败返回 `500`。错误响应不得泄露 Agent 工作目录的绝对路径。
+ 现有 `/api/raw`、`/api/download` 和其它文件接口行为保持不变；Site 的音视频预览改用新接口，避免将大媒体文件整体 Base64 编码到 JSON 响应。

### 编写代码
+ 在 Integration 主 HTTP 服务注册 `/api/media_preview`，复用现有 Agent 工作目录解析与根目录约束能力；媒体扩展名与 MIME 映射集中维护，避免 Site 与服务端对支持范围不一致。
+ 覆盖自动化测试：方法限制、缺少参数、相对路径读取、绝对路径和路径逃逸拒绝、其它 Agent 隔离、目录/非媒体拒绝、MIME 头、`Range` 的部分内容响应及 `HEAD`。
+ 最小范围更新，不新增外部依赖。

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`，说明媒体预览接口、路径范围、浏览器播放限制和 Range 支持。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求，不记录实现过程。
