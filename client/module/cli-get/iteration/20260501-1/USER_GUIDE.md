# 迭代 20260501-1：cli/get 命令注册与终止捕捉

## 功能说明

1. cli/get 执行的命令在开始时注册到活跃命令列表，结束时注销，确保 `/api/kill` 能找到并终止
2. 命令被 `/api/kill` 终止时，捕捉 context.Canceled 和 SIGKILL 信号，返回"命令被终止"而非挂起

## 终止检测

- `context.Canceled`：kill handler 调用 `cancel()` 触发
- `ExitCode == -1`：进程被信号杀死（SIGKILL）
- 超时：`context.DeadlineExceeded` → "命令执行超时"
