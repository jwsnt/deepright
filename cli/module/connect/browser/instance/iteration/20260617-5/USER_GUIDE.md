# 20260617-5 使用手册

## 目标

本次迭代补充 `browser_instance stop` 在 Windows WSL / WSL2 下的收尾清理语义，并同步完善 `help` 对外手册。

核心变化：

- `stop` 在原有实例关闭流程结束后，会继续清理 `C:\ProgramData\deepright` 下全部 `chrome*` 目录
- 目录删除改为并发执行，任何单个删除失败都不会阻断命令返回
- 每个目录的删除成功和失败都必须记录到 `browser.log`
- `help` / `USER_GUIDE.md` 需要明确说明上述行为

## help

```bash
./browser_instance help
./browser help
./browser instance help
```

说明：

- `help` 需要覆盖 `create / init / restart / stop / shutdown / list / get` 的完整使用手册
- 手册里需要明确说明 WSL 下实例目录固定落在 `C:\ProgramData\deepright\chrome_<随机后缀>`
- 手册里需要明确说明 WSL 下 `stop` 会在原流程结束后并发清理 `C:\ProgramData\deepright` 下全部 `chrome*` 目录
- 手册里需要明确说明删除成功和失败都会记录日志，且删除失败不会让 `stop` 失败

## stop

```bash
./browser_instance stop --agentId agent-a --chatId chat-001
```

行为：

- 仍然要求显式传入 `--agentId` 和 `--chatId`
- 仍然先按原有流程关闭目标实例并移除状态记录
- 如果系统为 Windows WSL / WSL2，则在原流程结束后，并发删除 `C:\ProgramData\deepright` 下全部 `chrome*` 目录，包括 `chrome_def`
- 每个目录的删除成功和失败都会写入 `browser.log`
- 任意目录删除失败都不会阻断命令，成功路径仍返回 `OK`

## 总结

- 本次迭代新增的是 WSL 下 `stop` 的 best-effort 目录回收语义
- `stop` 的收尾清理范围是 `C:\ProgramData\deepright` 下全部 `chrome*` 目录，而不只是当前实例目录
- 对外 help 和模块手册都需要把这部分行为说明清楚，避免调用方误判
