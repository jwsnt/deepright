# 20260503-8 使用手册

## 目标

本次迭代补充 proxy 侧备忘录（cron）删除与清理规则，并与 integration 保持一致。

## 行为变化

- 删除 Agent 时，会同步删除该 Agent 关联的全部任务元数据和全部任务明细
- 通过 `/api/cron/delete` 删除任务元数据时，只会联动删除尚未完成的任务明细
- `started = 3` 的已完成任务明细会保留，便于保留历史执行记录
- 任务元数据与任务明细的删除行为都会写入 `cron_meta_log` / `cron_detail_log`

## 日志说明

- 元数据日志表：`cron_meta_log`
- 明细日志表：`cron_detail_log`
- 删除 Agent 触发的联动删除也会写入审计日志

## 接口提示

- `GET /api/agent/delete?name=<agent>`
- `POST /api/cron/delete`
- `POST /api/cron/detail/delete`

## 编译

```bash
cd /path/to/deepright/cli/module/proxy
/opt/homebrew/bin/go build -o proxy ./
```
