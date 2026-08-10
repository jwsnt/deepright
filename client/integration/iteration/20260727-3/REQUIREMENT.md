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
+ 新增同源 `POST /api/video_record_convert` 受限接口，用于将 Site 已写入当前 Agent 工作区的临时画面录制 WebM 转为 MP4。请求仅含 `agentId`、工作区相对 `path` 与仅含文件名的 `outputName`；不接收浏览器二进制流、绝对路径、命令、编解码参数或输出目录。
+ `path` 必须严格匹配当前 Agent 工作区 `tmp/.screen-recording-<随机值>.webm` 的临时文件命名空间，拒绝非 POST、缺失字段、绝对路径、`~`、`..`、目录、非 WebM、非临时命名、跨 Agent 文件、符号链接逃逸和无效/过大请求体。接口不接受普通工作区视频作为源，防止它成为通用转码器。
+ `outputName` 必须是安全且非空的单个 `.mp4` 文件名；输出固定为当前 Agent 工作区 `videos/`，目录不存在时创建，且始终新建不覆盖。若 `videos/<outputName>` 已存在，或最终原子创建时发生同名并发冲突，服务端必须在扩展名前追加时间戳并重试，成功响应通过 `savedAs` 返回实际最终绝对路径；路径字符、非 MP4 后缀及工作区边界外目标都必须拒绝。
+ 转码必须处理输入 WebM 的完整时间线与全部可读取视频帧，不得通过时长取整、0.5 秒切分或页面时间戳裁剪减少内容。服务端以受控 FFmpeg 参数重编码为 H.264 视频、AAC 音频与 MP4 封装；音频流缺失时仍须生成可播放的视频。若共享画面的宽或高为奇数，必须在右侧或底部补齐至最近偶数尺寸，不能裁掉原始画面，以满足 H.264 4:2:0 的编码约束。源文件、输出路径、临时路径和命令参数均由服务端生成，调用设置超时。
+ 转码先在同一 `videos/` 目录生成不可见临时 MP4，成功后原子改名为最终名称；任何失败、超时、依赖缺失、校验错误或清理错误都不得留下最终 MP4。成功后服务端删除输入临时 WebM；失败后也删除该临时 WebM 和临时 MP4，确保 `videos/` 只包含用户最终 MP4。
+ FFmpeg 与 FFprobe 只可从受控运行环境查找；接口应复用现有依赖检查与工作区安全解析原则，不得执行安装命令、不新增外部依赖、不放宽 `/api/edit` 的二进制写入边界，也不得暴露任意文件转码能力。

### 编写代码
+ 最小范围更新，不新增外部依赖

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`，说明临时 WebM 输入范围、全量 MP4/H.264/AAC 转码、`videos/` 输出、清理规则、工作区安全边界及 FFmpeg/FFprobe 依赖。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求，不记录实现过程。
