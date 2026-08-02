# Wav2Lip 人物视频对口型

发布包的 `config/config.json` 使用 `wav2lip.check` 控制成功检查缓存时长，使用 `wav2lip.install` 作为缺失依赖时发送至当前 Chat 的安装请求。服务端会将文案中的 `$workspace` 替换为当前页面 Agent 的实际工作目录。每个 Agent 只能在其自己的固定目录使用 Wav2Lip：

```text
$workspace/wav2lip/inference.py
$workspace/wav2lip/checkpoints/wav2lip_gan.pth
```

两个文件都非空且可用 Python 能执行 `inference.py --help` 才算安装成功。成功结果仅在当前 Integration 进程内按 Agent 工作目录缓存；启动时会异步预热现有 Agent 的检查，不阻塞启动。检查和执行不搜索环境变量、其它目录、其它模型或其它 Agent，也不修改下载的 Wav2Lip 代码、模型或第三方包。

接口为 `GET /api/wav2lip/check?agentId=...`、`GET/POST /api/wav2lip/tasks`、`POST /api/wav2lip/tasks/cancel`、`/restart`、`/delete` 与 `GET /api/wav2lip/tasks/log`。创建和执行都重新确认 FFmpeg、FFprobe、视频流和音频流。每个任务仅接受同一 Agent 工作区内的一条视频与一条音频；音频格式为 WAV、MP3、M4A、AAC、FLAC 或 OGG。输出固定写到 `videos/<视频名>_lip_sync.mp4`，冲突时追加时间戳和必要序号，绝不覆盖已有文件。

任务使用受控的上游参数：固定 GAN 模型、视频、音频和输出路径。macOS 优先 Apple MPS，其他系统优先 CUDA；首次 GPU/MPS 执行失败时清理临时文件、记录失败原因，并以 CPU 从头重试一次。为让原始 Wav2Lip 在 MPS 上运行，Integration 仅在自身启动的子进程内选择设备，绝不写入上游目录。任务、状态、日志和取消标记持久化；服务重启后未取消的运行任务会重新排队。失败或已取消记录可以删除，删除不会影响源文件或输出文件。

原始 Wav2Lip 在最终合成时没有为音频和输出路径加引号。Integration 会在自身的无空格临时目录中映射任务文件，并固定以 `$workspace/wav2lip` 作为子进程工作目录后再调用它；成功后才将 MP4 写入 Agent 的 `videos/`。因此 macOS 容器路径中的 `Application Support` 不会导致 FFmpeg 截断路径或临时目录清理后的 `getcwd` 警告。该过程不修改下载的 Wav2Lip 代码、模型或第三方依赖。

如果 PyTorch 报出 `unexpected EOF`，服务会将其识别为人脸检测模型缓存不完整或损坏，并提示删除 `~/.cache/torch/hub/checkpoints/s3fd-619a316812.pth` 后重新执行；下一次执行会重新下载完整模型。原始堆栈仍保留在任务日志中。
