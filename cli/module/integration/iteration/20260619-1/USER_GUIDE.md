# 迭代 20260619-1 使用手册

本次迭代把本地通知能力收口进了 `integration`，并同时提供了可复用子模块与独立 CLI。

## 自动通知

- 普通对话的 `/v1/chat/completions` SSE 整体结束后，会触发一条系统通知
- 备忘录任务的内部 SSE 执行链路整体结束后，也会触发一条系统通知
- 正常结束和异常结束都会通知
- 通知标题固定为 `DeepRight通知`
- 普通对话通知内容会直接显示 `用户最后一条问题摘要`；正常完成与异常结束都会沿用同一摘要，超过 `10` 个字符会自动截断为 `...`
- macOS 下通过系统原生通知显示，并跟随当前 `integration` 进程所属应用图标
- Windows 与 WSL 下通知使用系统信息图标
- 当前支持平台为 macOS、Windows 与 WSL

## CLI

```bash
./integration notify --title "DeepRight通知" --message "普通对话已完成"
```

返回示例：

```json
{
  "status": 0,
  "supported": true,
  "title": "DeepRight通知",
  "message": "普通对话已完成"
}
```

说明：

- `supported=true` 表示当前运行环境具备实际通知能力
- 在非 macOS / Windows / WSL，或当前环境缺少本地通知依赖时，命令仍可执行，但 `supported` 会返回 `false`

## 复用方式

代码侧可以直接复用：

- 包：`integration/notification`
- 入口：`notification.Notify(notification.Options{Title: "...", Message: "..."})`

当前 `integration` 已经在两条主链路接入：

- 对外 `/v1/chat/completions`
- 内部备忘录 `cronExecuteOnce`
