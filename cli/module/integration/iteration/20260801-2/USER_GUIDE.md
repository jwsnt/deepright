# Integration 迭代 20260801-2 使用手册

## Whisper 依赖检查

主应用活动 `config/config.json` 需要配置：

```json
{
  "whisper": {
    "check": 24,
    "install": "请安装 openai-whisper"
  }
}
```

`GET /api/whisper/check` 会验证受控运行环境中的 `whisper` 命令存在且可启动；macOS 同时会识别当前用户的 `~/Library/Python/*/bin/whisper` 安装位置，避免桌面应用未继承终端 `PATH` 时误报未安装。成功检测会按 `check` 的小时数缓存；命令缺失、失效、配置无效或检查失败不会缓存，接口会返回安装请求，且不会在服务端执行安装。

Integration 在启动时会构建统一的命令环境，并由 Whisper/FFmpeg 检查、Integration 的 `cmd` 请求、`CLI_SANDBOX` 和 `cli/get` 任务执行共同继承。这个 `PATH` 会动态合并系统、登录 Shell、启动服务、常见用户目录以及 macOS 当前用户已存在的 `~/Library/Python/*/bin`，因此用户通过 `pip --user` 安装的 `whisper` 不需要写死用户目录或 Python 版本；依赖检查成功后，实际任务会在同一环境中查找该命令。

在 WSL/Linux 中，用户级 Python 命令的标准位置为 `~/.local/bin`，同样会加入规范化 `PATH`。WSL `CLI_SANDBOX` 会清空宿主环境以保持隔离，因此只读挂载该目录与实际存在的 `.local/lib/python*/site-packages`，并设置对应 `PYTHONPATH`；不会把整个用户目录或 `.local` 数据目录暴露给沙箱。

## 任务接口与队列

- `GET /api/whisper/tasks?agentId=<AgentId>&status=<状态>&page=<页码>` 返回当前 Agent 的任务。`status` 可为 `queued`、`running`、`completed`、`cancelled`、`failed` 或省略（全部）；每页固定 5 条，并返回总数、当前页和页大小。
- `POST /api/whisper/tasks` 接受 `{ "agentId": "...", "paths": ["audios/demo.mp3"] }`，仅允许当前 Agent 工作区内的受支持音频相对路径。
- `POST /api/whisper/tasks/cancel` 接受 `{ "agentId": "...", "id": 1 }`，可取消排队或正在执行的任务；`POST /api/whisper/tasks/restart` 可将已取消任务重新加入队列。
- `POST /api/whisper/tasks/delete` 接受 `{ "agentId": "...", "id": 1 }`，仅可删除当前 Agent 的失败任务记录；不会删除音频或已生成文字文件。
- `GET /api/whisper/tasks/log?agentId=<AgentId>&id=1` 返回指定任务的持久化执行日志。

任务记录保存在共享 SQLite 的 `whisper_task` 表。任务使用固定 `base` 模型，按创建顺序全局串行执行；当前任务完成、失败或被取消后，队列会立刻开始下一个任务。服务重启时，未取消的执行中任务会恢复为排队中并保留恢复日志。

已取消任务可通过重启接口重新加入队列。系统会重新分配不冲突的文字输出路径；如重新开始失败，原因会写入该任务的执行日志。

每项任务的输出固定在当前 Agent 工作目录的 `whisper/` 中，优先使用与源文件同名的 `.txt` 文件，例如 `audios/demo.mp3` 输出到 `whisper/demo.txt`。已有同名文本或任务输出不会被覆盖，系统会自动追加时间戳（必要时追加序号）生成新文件名。Whisper 输出会尽可能解析为百分比进度，无法识别时仍会保持“执行中”。

执行时会使用运行同一 Whisper Python 环境探测 CUDA 与 Apple MPS；验证到 CUDA 时优先 CUDA，验证到 MPS 时优先 MPS，不能验证 GPU 时自动使用 CPU，并在任务日志中记录选择原因。若实际加载模型或转写时 GPU 后端不受当前 Whisper 环境支持，系统会保留该次输出并自动用 CPU 重试一次；仅当 CPU 重试也失败时才将任务标记为失败。
