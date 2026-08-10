# Integration 迭代手册（20260725-2）

## 媒体预览接口

Integration 新增只读媒体流接口：

```text
GET /api/media_preview?agentId=<agentId>&path=<relativePath>
```

`agentId` 是当前 Agent 标识；`path` 是其工作目录内的相对文件路径，例如：

```text
/api/media_preview?agentId=demo&path=outputs/recording.mp3
```

接口只接受以下类型：

| 类别 | 扩展名 |
| --- | --- |
| 视频 | `mp4`、`m4v`、`webm`、`ogv`、`mov`、`mkv`、`avi`、`mpg`、`mpeg` |
| 音频 | `mp3`、`m4a`、`aac`、`wav`、`flac`、`ogg`、`opus` |

## 安全范围与响应

- 只允许读取 `agentId` 对应工作目录内的文件。绝对路径、`..` 逃逸、目录、非媒体文件、其它 Agent 的文件都会被拒绝。
- `GET` 返回内联媒体内容，`HEAD` 返回相同元数据但不返回响应体。
- 服务会设置匹配扩展名的媒体 `Content-Type`、`Content-Disposition: inline` 和 `X-Content-Type-Options: nosniff`。
- HTTP Range 请求受到支持，播放器可以只读取需要的片段，完成进度条拖动和 15 秒前进/后退；源文件不会被改写。

常见响应状态：

| 状态 | 含义 |
| --- | --- |
| `200` | 成功返回完整媒体内容 |
| `206` | 成功返回 Range 请求的部分内容 |
| `400` | 参数、路径或文件类型不符合预览范围 |
| `404` | Agent 或媒体文件不存在 |
| `405` | 非 `GET` / `HEAD` 请求 |
