# SAM2 视频物体提取服务

`GET /api/sam2/check?agentId=<Agent>` 检查当前 Agent 工作区的 SAM2 运行环境和基础 tiny 模型。`config/config.json.sam2.install` 中的 `$workspace` 会由服务端替换为当前 Agent 的工作目录，前端不会自行安装。

任务接口为 `/api/sam2/tasks`、`/cancel`、`/restart`、`/delete` 和 `/log`。任务保存受控模型 ID、可选的首帧点/框/掩码提示、状态、输出路径与日志；模型只能为 tiny、small、base、large。完全未标注时，服务使用 SAM2 自动分割首帧并选择最显著主体作为视频跟踪目标。执行日志记录模型、权重路径、提示或自动选择、设备、下载过程和最终输出。

模型缺失时，任务运行时从官方源下载对应 checkpoint，失败后清除未完成断点并切换备用源。透明 MOV 输出和黑底 MP4 预览均在当前 Agent 的 `videos/` 中生成，取消或失败会清理临时文件。
