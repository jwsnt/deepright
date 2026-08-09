# Chrome 录屏 MP4 转码服务使用手册

Integration 提供同源 `POST /api/video_record_convert`，把 Site 上传的临时画面录制 WebM 转为最终 MP4：

```json
{
  "agentId": "AgentId",
  "path": "tmp/.screen-recording-<随机值>.webm",
  "outputName": "recording.mp4"
}
```

输入路径只能是当前 Agent 工作区 `tmp/` 下符合专用命名规则的临时 `.webm`；普通工作区视频、绝对路径、路径逃逸、目录、符号链接和跨 Agent 路径均会拒绝。输出名称只能是安全的单个 `.mp4` 文件名，服务始终在当前 Agent 的 `videos/` 新建 MP4；若同名或原子创建遇到并发冲突，会自动在扩展名前追加时间戳，绝不覆盖已有文件。

转码覆盖输入的完整可读取时间线，不使用视频切分接口的 0.5 秒截断规则。服务端以受控 FFmpeg 参数输出 H.264 视频、AAC 音频和 MP4 容器；输入没有音频时仍可生成视频。若录制源的宽或高为奇数，服务会在右侧或底部补一个像素以满足 H.264 编码要求，不会裁掉原始内容。FFprobe 可用时会读取源视频目标码率。

服务先在 `videos/` 内生成临时 MP4，再以原子方式创建最终文件。成功响应的 `savedAs` 提供最终绝对路径。成功、失败、超时、冲突或清理错误都会清除临时 WebM 和临时 MP4，失败不会留下最终输出。

服务要求受控运行环境同时提供 `ffmpeg` 与 `ffprobe`，不会执行安装命令，也不提供任意文件转码能力。
