# WAV 混音服务使用手册

Integration 提供同源 `POST /api/audio_mix`，将当前 Agent 工作区中的音频源按结构化工程合成为双声道 WAV：

```json
{
  "agentId": "AgentId",
  "outputName": "final.wav",
  "sampleRate": 44100,
  "bitDepth": 16,
  "tracks": [
    {
      "path": "assets/voice.flac",
      "trimStart": 0,
      "trimEnd": 12.5,
      "start": 3,
      "speed": 1,
      "loop": false,
      "duration": 12.5,
      "volume": 1,
      "leftVolume": 1,
      "rightVolume": 1,
      "muted": false,
      "fadeIn": 0,
      "fadeOut": 0,
      "effects": [{"type": "reverb", "amount": 0.3}]
    }
  ]
}
```

每条 `path` 必须是当前 Agent 工作区内的普通相对文件，且 FFprobe 能检测到音频流。接口拒绝绝对路径、`~`、`..`、目录、符号链接逃逸、跨 Agent 路径及不含音频流的文件；因此它不能用作任意文件读取器或通用转码器。循环、裁剪、速率、淡入淡出、音量和效果器均为受限结构化参数，效果器仅支持 `equalizer`、`compressor`、`reverb` 与 `delay`。

导出采样率只允许 `44100` 或 `48000`，位深只允许 `16` 或 `24`；服务固定输出两声道 PCM WAV。输出名称只能是安全的单个 `.wav` 文件名，文件固定新建于当前 Agent 的 `audios/`。同名文件或原子提交期间的并发冲突会返回冲突错误，绝不覆盖或自动改名。

服务先在 `audios/` 内写入临时 WAV，成功后原子创建最终文件，并在成功响应的 `savedAs` 中返回绝对路径。解析失败、校验失败、执行超时、依赖缺失、FFmpeg 失败或保存冲突都会清理临时文件且不留下最终 WAV；源音频始终只读。运行环境必须能从受控命令搜索路径找到 `ffmpeg` 与 `ffprobe`，接口不会执行安装命令。
