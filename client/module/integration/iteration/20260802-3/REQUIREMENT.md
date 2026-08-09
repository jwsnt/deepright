### 第一性原则
+ 仅可以新增、更新或删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../site/index.html`、`../../../config/config.json` 与应用发布资源。

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
+ 不新增外部 Go 依赖，不改变既有 `/api/runtime_config`、普通消息发送、附件上传或 Agent 工作区访问协议。

### 需求介绍

- 主应用资源 `config/config.json` 必须提供：
  ```json
  "rvm": { "check": 24, "install": "请安装 robust video matting，模型与代码存放至 `$workspace/rvm` 目录下，检查安装是否成功的标准为`$workspace/rvm/weights/rvm_mobilenetv3.pth`是否存在" }
  ```
  `check` 必须是正整数小时，`install` 必须是非空字符串；其中 `$workspace` 是唯一允许的运行时占位符。`GET /api/rvm/check?agentId=<当前页面Agent>` 每次读取并校验此配置，服务端用该 Agent ID 解析其工作目录，再将所有 `$workspace` 替换为此已解析目录后返回安装请求；配置或 Agent 错误返回可展示错误且不得使用默认值。RVM 代码固定在 `$workspace/rvm/inference.py`，模型固定在 `$workspace/rvm/weights/rvm_mobilenetv3.pth`。
- 某个 Agent 的 RVM 仅在其 `$workspace/rvm` 目录同时存在非空 `$workspace/rvm/inference.py` 与非空 `$workspace/rvm/weights/rvm_mobilenetv3.pth`，且以实际选定的 Python 受控执行 `inference.py --help` 成功时，才算已安装；不得使用其它 Agent 的安装目录，也不得回退查找其它模型路径。成功结果仅在 Integration 进程内按 `rvm.check` 缓存，进程重启自然失效。服务启动时异步检查并缓存每个既有 Agent 的该标准目录；检查失败不得阻塞启动、执行安装或写入 Agent 工作区。
- 为兼容新版 PyAV，Integration 启动 RVM 推理时必须通过自身受控包装器将帧率转换为有理数，再执行未修改的 `$workspace/rvm/inference.py`。该包装器不得写入、补丁或以其他方式修改下载的 RVM 上游代码。
- 新增受控接口：`GET /api/rvm/check` 必须携带当前页面的 `agentId`，服务端只接受有效 Agent ID 并解析其工作目录；该接口不得读取或传递会话 UUID，也不得以 VFS 路径推断身份。`GET/POST /api/rvm/tasks`、`POST /api/rvm/tasks/cancel`、`POST /api/rvm/tasks/restart`、`POST /api/rvm/tasks/delete`、`GET /api/rvm/tasks/log` 只接受当前 Agent 的工作区相对路径与结构化参数，不接受自定义命令、模型参数、输出路径或其它 Agent 的文件信息。
- 启动时幂等创建共享 SQLite 的 `rvm_task` 表，持久化任务 ID、Agent ID、源/输出相对路径、输出绝对路径、场景、状态、进度、开始/创建/更新时间、取消标记及受限大小日志；已有表须安全迁移场景字段。状态固定为 `queued`、`running`、`completed`、`cancelled`、`failed`；列表支持状态筛选、固定每页 5 条。删除接口只允许删除当前 Agent 的失败或已取消记录，且不得删除源视频或输出文件。重启时未取消的运行任务恢复为排队中，完成、失败和取消记录保留。
- 创建任务时必须校验 Agent ID、路径边界和视频扩展名，并以 `ffprobe` 确认可读取的第一视频流；非视频、伪造后缀、目录、工作区外路径、符号链接逃逸及跨 Agent 路径均必须拒绝。队列在全部 Agent 间串行执行，取消排队任务或终止运行中的子进程后立即继续下一条。
- 输出固定为当前 Agent `videos/<源文件名>_subject.mov`，并额外生成同名 `.mp4` 预览副本。MOV 是任务的主产物和复制路径，保留透明通道；MP4 仅用于浏览器预览，不保留透明通道，且必须先把透明区域合成为纯黑背景。任一同名 MOV 或配套 MP4 冲突时，两个文件同时追加相同时间戳和必要序号，绝不覆盖源文件或已有输出；失败、取消或回退前必须清理临时文件。
- 每条任务创建时保存受控场景 ID；空值或既有任务按“标准主体提取”处理，未知值必须拒绝。场景固定为：标准主体提取（默认，4 Mbps）、精细边缘提取（`--downsample-ratio 1`，8 Mbps）和快速主体提取（`--downsample-ratio 0.25`，2 Mbps），均使用 `--seq-chunk 1`。每条任务先重新确认 `ffmpeg` 与 `ffprobe` 可用，再以其已保存场景的受控参数执行 RVM：`--variant mobilenetv3 --checkpoint <已发现的模型权重> --input-source <source> --output-type video --output-foreground <temp> --output-alpha <temp>`。macOS 先用 `mps`，其它系统先用 `cuda`；GPU/MPS 执行失败时记录原因、清理临时输出，并仅以 `cpu` 从头重试一次。成功后使用受控 FFmpeg `alphamerge` 生成带透明通道的 ProRes 4444 MOV，再将其透明区域合成纯黑背景，受控转码为 H.264 MP4 预览副本；子进程输出和失败原因写入任务日志。

### 编写代码
+ 保持实现最小化，避免为纯文本模板增加数据库、任务队列、子进程、文件系统写入或新的 HTTP 路由。

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求、边界和验收行为，不记录实现过程。
