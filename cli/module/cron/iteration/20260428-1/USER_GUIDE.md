# 迭代 20260428-1：启动定时器与最后执行时间

## 功能说明

应用启动时即启动定时器，每分钟执行一次周期检查，将后 5 天内非一次性的任务元数据拆分为任务明细。已存在的明细不会重复创建。每次检查记录最后执行时间。

## Cron 模块

- `CronService.Start()` — 启动后台定时器，立即执行一次检查，之后每分钟执行
- `CronService.LastCheckTime` — 记录最后一次检查的时间
- CLI 新增 `start` 命令：`./cron start`（阻塞运行，持续执行周期检查）

## Integration 同步

Integration 启动时自动启动 `startCronCheck()` 后台协程，逻辑与 cron 模块一致：
- 每分钟查询 `task_meta` 中 `cycle != 0` 的记录
- 为后 5 天内的执行时间点创建 `task_detail`
- 已存在则跳过（`INSERT OR IGNORE`）
