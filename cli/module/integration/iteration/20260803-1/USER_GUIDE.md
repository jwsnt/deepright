# 媒体任务重新开始与日志求助

五类媒体任务的重启接口现在同时接受 `failed` 与 `cancelled` 记录。成功重新开始时，任务保留原有受控输入与参数、重新分配安全输出路径并清空原 `logs`；新的日志会在任务实际开始执行时出现。

`POST /api/task_help` 接受当前 Agent 的 `agentId`、受限任务类型 `voxcpm`、`whisper`、`rembg`、`wav2lip` 或 `rvm`，以及任务 `id`。服务端验证任务归属后，把当前任务日志写入该 Agent 工作目录的 `tmp/`，文件名为 `<功能名>_YYYYMMDD_HHMMSS.log`；同秒冲突时自动追加序号且不覆盖已有文件。

接口每次读取 `config/config.json.seekHelp`。该配置必须同时含有 `$function` 和 `$log`：服务端将前者替换为固定中文功能名，将后者替换为 `[FILE:<日志绝对路径>]`，并以 `content` 返回生成消息。接口不会自行发送消息，浏览器使用当前会话的普通消息发送链路完成发送。

五类媒体任务的每次执行日志还会记录任务目的、实际启动的 Python/CPython 解释器和执行入口/脚本、服务端确认的输入文件绝对路径、最终输出文件绝对路径、实际模型/权重及具体参数。模型或运行时资源需要下载时，日志会给出下载源、准备写入的绝对路径、进度和校验结果；失败的最终记录会包含具体原因及详细错误输出，便于判断输入、网络、模型或输出写入问题。

五类媒体任务共用一个模型执行队列。服务启动时读取 `config/config.json` 的 `modelTask.concurrence`，该正整数决定文字转语音、音频转写、图片主体提取、人物视频对口型和视频主体提取合计可同时执行的任务数；默认配置为 `1`。没有公共名额的任务继续显示为“排队中”，获得名额才进入“执行中”。修改该值后重启 Integration 生效；值缺失时使用 `1`，非法值同样安全回退为 `1` 并记录服务日志。

五类功能均可由 CLI 管理：`integration api whisper`、`rembg`、`voxcpm`、`wav2lip` 与 `rvm`，每组均支持 `check`、`list`、`create`、`cancel`、`restart`、`delete` 和 `log`。执行 `integration api <功能> --help` 可查看受控参数、状态边界和案例；`restart` 适用于失败或已取消任务，并会清空旧日志。RVM 的 `--path` 可配合 `--scenario standard|quality|fast` 使用；Wav2Lip 则以 `--videoPath` 和 `--audioPath` 的顺序组成视频、音频配对。
