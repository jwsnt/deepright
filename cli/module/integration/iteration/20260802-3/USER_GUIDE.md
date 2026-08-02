# RVM 视频主体提取

发布环境须在 `config/config.json` 配置 `rvm.check` 与 `rvm.install`。安装请求中的 `$workspace` 会由服务端替换为当前 Agent 的实际工作目录。RVM 仅安装在该 Agent 的下列固定位置，检查与任务执行均使用这两个固定地址，不回退查找其它目录；无需、也不得设置任何 RVM 环境变量：

```bash
$workspace/rvm/inference.py
$workspace/rvm/weights/rvm_mobilenetv3.pth
```

`GET /api/rvm/check?agentId=<当前页面Agent>` 使用实际发现的 Python 执行 `$workspace/rvm/inference.py --help` 预检该 Agent 的 RVM。检查过程只使用当前页面已选择的 Agent ID，不使用会话 UUID 或 VFS 路径。缺少固定脚本、固定模型权重或可用 Python 时，接口返回 `available: false` 与已经展开 `$workspace` 的配置安装请求。

任务接口为 `GET/POST /api/rvm/tasks`、`POST /api/rvm/tasks/cancel`、`/restart`、`/delete` 与 `GET /api/rvm/tasks/log`。创建请求只能引用当前 Agent 工作区中的视频，并为每项任务保存一个场景：标准主体提取（默认）、精细边缘提取或快速主体提取；旧任务自动作为标准主体提取处理。服务用 FFprobe 再次确认视频流。结果固定为透明的 `videos/<原文件名>_subject.mov`，并生成同名 `.mp4` 预览副本；预览副本不含透明通道，原透明区域会填充为纯黑色。复制路径仍为 MOV。任一配对文件冲突时都会追加同一时间戳，且不覆盖已有文件。

处理要求同时具备 FFmpeg、FFprobe 与 RVM。标准主体提取使用平衡的 4 Mbps 输出；精细边缘提取使用完整分辨率与 8 Mbps；快速主体提取使用 0.25 下采样与 2 Mbps。RVM 先生成前景和 alpha 临时视频，再由 FFmpeg 合成为透明 ProRes 4444 MOV，随后将其透明区域合成纯黑背景，生成不含透明通道的 H.264 MP4 预览副本。macOS 优先 Apple MPS，其他系统优先 CUDA；GPU/MPS 失败时会删除临时输出、记录原因并用 CPU 从头重试一次。任务场景、状态、日志和取消信息持久化于共享 SQLite，服务重启后未取消任务会重新排队。
