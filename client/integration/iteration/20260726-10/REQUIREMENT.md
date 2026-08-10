### 第一性原则
+ 仅可以新增/更新/删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../build.sh` 与 `../../../config/app/API.md`、`../../../config/app/CANVAS.md`、`../../../config/app/DESIGN.md`。

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
+ 从主应用资源目录的 `config/config.json` 读取必填 `ffmpeg` 对象：`ffmpeg.check` 为正整数，单位为小时；`ffmpeg.install` 为非空字符串，内容由 Site 在用户确认后发送到当前 Chat。不得读取 Agent 工作目录中的 `config.json`，也不得提供默认值。
+ 新增同源只读依赖检查接口，供视频预览编辑入口调用。该接口只能接受 `GET`，每次请求先校验上述配置；配置缺失或无效时返回可直接展示的配置错误。
+ 接口应通过受控的本机可执行文件查找同时检查 `ffmpeg` 和 `ffprobe`。仅在两者都存在时，将成功时间保存在 Integration 进程内存，并在 `ffmpeg.check` 指定的小时数内直接返回可用；进程重启后缓存自然失效。未安装、只安装其中一个或查找异常时不得写入成功缓存。
+ 依赖缺失时接口返回可展示的未安装状态及经过配置校验的 `ffmpeg.install` 文本；不得执行安装命令、不得修改配置文件、不得写入 Agent 文件或视频文件。缓存命中也必须使用当前有效配置的周期判断是否过期。
+ 现有 `/api/video_trim` 的受控转码、二进制与 CLI 收口原则保持不变；检查接口只负责预先决定能否进入视频编辑界面，不能取代实际保存时的依赖检查。

### 编写代码
+ 最小范围更新，不新增外部依赖

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`，说明 `ffmpeg` 配置、预检接口、仅成功结果的进程内缓存、重启失效与依赖缺失时的安装请求行为。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求，不记录实现过程。
