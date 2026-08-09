# Integration 迭代手册（20260719-1）

## 本次更新

Integration 在 macOS 服务运行期间会自动持有一个独立的 `caffeinate` 子进程，防止空闲熄屏或系统空闲睡眠影响任务执行。

## 行为

- 子进程使用 `caffeinate -d -i -m -s`，持续到当前 Integration 进程结束；不使用定时租约或续租。
- `integration stop`、`restart` 的停止阶段、`SIGINT`、`SIGTERM` 及本机 `/api/shutdown` 都会终止该子进程，不留下防睡眠进程。
- 仅 macOS 启用。Linux、WSL、Windows 及其他平台保持原有启动行为。
- 若系统无法启动 `caffeinate`，Integration 会记录日志但仍继续启动服务。

## 范围

此功能仅防止空闲熄屏和系统睡眠；不会修改屏幕保护程序或锁屏设置，也不支持合上笔记本盖子后继续运行。
