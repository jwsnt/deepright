### 第一性原则

+ 仅可以新增、更新或删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../site/index.html`、`../../../config/config.json` 与应用发布资源。

### 技术规范

+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
+ 不新增外部 Go 依赖，不改变既有运行时配置、普通消息发送、附件上传或 Agent 工作区访问协议。
+ 下载的 Wav2Lip 代码、模型和全部第三方包均视为只读；不得补丁、写入或修改它们。所有兼容、设备选择及参数适配只能位于 Integration 自身代码。

### 需求介绍

- 主应用资源 `config/config.json` 必须提供 `wav2lip.check`（正整数小时）和 `wav2lip.install`（非空安装请求）。安装请求中的唯一运行时占位符 `$workspace` 由服务端替换为当前 Agent 的已解析工作目录。
- Wav2Lip 固定安装于当前 Agent 的 `$workspace/wav2lip`：上游入口固定为 `$workspace/wav2lip/inference.py`，默认模型固定为 `$workspace/wav2lip/checkpoints/wav2lip_gan.pth`。只有两个文件均存在且非空，并且可用 Python 实际执行 `inference.py --help` 成功时才视为可用；不得回退搜索其它目录、模型、环境变量或其它 Agent。成功检查仅在进程内按 `wav2lip.check` 缓存，缓存键必须隔离 Agent；进程重启后自然失效。启动时异步预检每个既有 Agent，不阻塞启动、不安装、不写入工作区。
- 增加 `GET /api/wav2lip/check?agentId=<当前页面Agent>`，以及受控的任务列表、创建、取消、重新开始、删除和日志接口。接口只接受当前 Agent ID、工作区相对路径和结构化参数；不得接受会话 UUID、自定义命令、模型路径、输出路径或其它 Agent 的文件。
- 创建和执行任务前均须确认 FFmpeg 与 FFprobe 可用；缺失时沿用既有 FFmpeg 安装确认流程。Wav2Lip 缺失时返回已展开 `$workspace` 的 `wav2lip.install`，服务端和浏览器都不得自行安装。
- 每条任务恰有一个工作区内的视频和一个工作区内的音频。视频仅接受受控视频扩展名并以 FFprobe 验证第一视频流；音频仅接受 `wav`、`mp3`、`m4a`、`aac`、`flac`、`ogg`，并以 FFprobe 验证音频流。前端从本机选择的音频上传到当前 Agent `tmp/` 后再创建任务。所有路径均须拒绝目录、符号链接逃逸、伪造后缀、工作区外与跨 Agent 路径。
- 任务状态固定为 `queued`、`running`、`completed`、`cancelled`、`failed`，持久化任务 ID、Agent ID、视频与音频相对/绝对路径、输出路径、状态、进度、时间、取消标记及受限大小日志。全局串行执行；重启时未取消的运行任务恢复排队。失败或已取消任务可删除记录，且不得删除源文件或已生成输出。
- 输出固定为 `videos/<视频名>_lip_sync.mp4`；同名文件、既有任务或并发竞争时追加同一规则的时间戳及必要序号，绝不覆盖任何已有文件。完成任务提供输出 MP4 的复制和预览。
- 执行仅可调用受控的上游语义：`inference.py --checkpoint_path <固定GAN模型> --face <视频> --audio <音频> --outfile <输出>`。必须优先使用 macOS MPS、其次 CUDA、最后 CPU；设备执行失败时写入原因、清理临时输出，并仅以 CPU 从头重试一次。上游原脚本不原生支持 MPS 时，Integration 可使用自身的受控运行时包装器适配设备，但不得修改上游文件。日志须记录实际设备、回退原因、受控命令失败输出及最终结果。
- 上游 `inference.py` 最终调用 FFmpeg 时未引用音频和输出路径。Integration 必须在自身管理的不含空白字符的临时目录中映射视频、音频和输出路径后再调用上游，并以固定的 `$workspace/wav2lip` 作为整个子进程树的工作目录；成功后才将结果安全写入既定的 Agent 输出路径。不得修改、补丁或复制覆盖下载的 Wav2Lip 文件。
- 对可识别的上游模型缓存损坏错误必须转换为明确原因并持久化，例如 PyTorch `unexpected EOF` 必须提示 Wav2Lip 人脸检测模型缓存不完整或损坏、给出 `~/.cache/torch/hub/checkpoints/s3fd-619a316812.pth` 的清理与重新下载指引；不得只向用户暴露 Python 堆栈。

### 编写代码

+ 保持实现最小化，复用既有受控依赖检查、任务队列、路径验证、FFmpeg、日志和 Agent 隔离能力。

### 撰写手册

+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`。

### 其他要求

+ `REQUIREMENT.md` 仅描述需求、边界和验收行为，不记录实现过程。
