# 视频编辑依赖检查服务手册

Integration 提供 `GET /api/ffmpeg/check`，供 Site 在显示视频切分界面前检查依赖。接口仅接受 `GET`，并在每次调用时读取主应用资源目录 `config/config.json` 的 `ffmpeg` 配置：

```json
{
  "ffmpeg": {
    "check": 120,
    "install": "请安装 ffmpeg 和 ffprobe"
  }
}
```

`check` 为正整数小时，`install` 为非空字符串；任一配置缺失或无效会返回 HTTP 400 和可展示的配置错误。接口检查 `ffmpeg` 与 `ffprobe` 是否都能从受控环境的命令搜索路径找到。两个命令都存在时返回 `available: true`，并仅在 Integration 当前进程的内存中缓存成功时间至多 `check` 小时；重启后缓存自动失效。

若缺少任一命令，接口返回 `available: false` 和 `install` 文本，不写入成功缓存，也不会执行该文本、安装软件、转码或写入文件。实际 `POST /api/video_trim` 在保存时仍会再次检查依赖并负责转码。
