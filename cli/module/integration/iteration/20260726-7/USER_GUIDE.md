# Integration 迭代手册（20260726-7）

## 录音 WAV 保存接口

Site 通过以下同源接口保存“新增 → 音频”的最终录音：

```http
POST /api/agent/audio?agentId=AgentId&path=audios/note.wav
Content-Type: application/json
```

```json
{
  "content": "Base64 编码的 PCM WAV 内容"
}
```

- `path` 必须是当前 Agent 工作区内的 `audios/<文件名>.wav` 相对路径。前端负责为无后缀名称补 `.wav`，服务端只接受 `.wav`，不会接受或转换其它后缀；`audios` 不存在时会自动创建。
- 服务端严格检查 Base64 和 WAV 容器：必须包含 RIFF/WAVE、PCM 16-bit 的 `fmt ` 区块和非空 `data` 区块。非法或截断内容会返回错误且不写入文件。
- 同名文件会返回 HTTP `409`，不会覆盖原文件。录音仅在最终保存时写入；写入先进入工作区内临时文件，再原子创建最终文件。
- 工作区外路径、`audios` 以外目录、绝对路径、`~`、路径逃逸、目录、跨 Agent 访问和符号链接逃逸均会被拒绝。接口没有任意本机文件写入能力。
- Integration 不依赖 FFmpeg，也不会转码、重采样、降噪或改写录音字节。
