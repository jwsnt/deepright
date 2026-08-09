# 视频音频剪辑服务使用手册

`POST /api/video_audio_edit` 用于将一个受限多轨音频工程合并到当前 Agent 工作区中的视频，并另存为 MP4。请求包含 `agentId`、视频相对 `path`、可选 `outputName` 与 `tracks`。

每段 `tracks` 可使用既有混音字段：`trimStart`、`trimEnd`、`start`、`speed`、`loop`、`duration`、`volume`、`leftVolume`、`rightVolume`、`fadeIn`、`fadeOut`、`effects`。原视频第一音轨使用 `original: true`，且不能带 `path`；新增轨道必须引用同一 Agent 工作区内的普通音频文件。没有原音轨的视频可只使用新增轨道。

接口只接受结构化参数。它不接受 FFmpeg 命令、滤镜字符串、绝对路径、跨 Agent 路径或输出目录。音频轨道经白名单 FFmpeg filter graph 混合后编码为 AAC，视频画面映射自源视频并保持不变。

`POST /api/video_audio_edit_preview` 接受相同的受限工程，但仅生成临时双声道 WAV 并以 `audio/wav` 返回，供前端试听。它不创建或覆盖 `videos/` 文件；响应结束、取消或失败时删除临时 WAV。

`POST /api/video_audio_extract_to_audio` 从当前 Agent 工作区的可预览视频中固定提取第一音轨（`0:a:0`），以 MP3 写入 `audios/`。请求只包含 `agentId` 与视频相对 `path`；接口不接受自定义输出路径、文件名或 FFmpeg 参数。输出重名或并发竞争时自动追加高精度时间戳，`savedAs` 返回最终绝对路径；无音轨、依赖缺失或提取失败不会创建文件。

默认输出名称为 `<源文件名>_edit.mp4`，固定写入 `videos/`。同名或并发竞争时服务端自动追加高精度时间戳，并在成功响应中设置 `renamed: true`；`savedAs` 返回最终绝对路径。临时 MP4 会在成功、失败或超时后清理，既有文件和源视频不会被覆盖或修改。运行环境必须提供 `ffmpeg` 与 `ffprobe`。
