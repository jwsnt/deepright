# Integration 迭代 20260509-2 使用手册

## 变更说明

本次迭代补充了 `integration` 生命周期中的 PID 清理规则，用于避免异常退出后残留的 `*.pid` 污染后续启动与关闭流程。

## 行为规则

- 每次执行 `integration start` 前，都会先清理应用启动目录下残留的 `*.pid` 文件，再继续启动
- 每次执行 `integration stop` 完成关闭后，也会再次清理应用启动目录下残留的 `*.pid` 文件
- 清理范围是当前应用启动目录下的同层 `*.pid`
- `integration.pid` 仍然由当前生命周期流程正常创建与删除，不会在启动前被误删

## 适用场景

如果上一次运行因为异常退出、强制杀进程或插件崩溃，导致启动目录下遗留：

- `browser.pid`
- `feishu.pid`
- `email.pid`

下一次执行 `start` 或完成一次 `stop` 后，这些残留 PID 文件会被自动清理，避免后续误判插件状态或污染运行环境。

## 补充说明

- 这里的“应用启动目录”优先取 `runtime.json` 中的 `app-dir`
- 如果取不到 `runtime.json`，则回退到 `integration.pid` 所在目录
