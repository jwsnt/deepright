# 20260503-3 User Guide

## 目标

本次迭代为 `integration` 增加了 cron 查询能力，支持在未启动独立 HTTP 服务的情况下通过 CLI 查询任务元数据与任务明细，并保持与 HTTP 接口一致的过滤语义。

## CLI 命令

查询任务元数据：

```bash
./integration cron find-meta --content "每15分钟检查一次上游接口健康" --model "OpenAI" --chatId "chat-001"
```

查询任务明细：

```bash
./integration cron find-detail --metaId cron_1 --content "每15分钟检查一次上游接口健康" --model "OpenAI" --cycle 4 --chatId "chat-001"
```

按日期查看某个 Agent 的未来或指定日明细：

```bash
./integration cron find-detail --agent demo-agent --date "2026-05-03"
```

## 查询规则

- 元数据支持 `agentId`、`chatId`、`model`、`content`、`cycle`、开始执行时间范围
- 明细支持 `metaId`、`agentId`、`chatId`、`model`、`content`、`cycle`、执行时间范围
- 未指定的维度表示全部匹配
- 明细未指定时间条件时，默认仅查询当前时间之后的数据
- `metaId` 支持 `1` 和 `cron_1`

## HTTP 接口

任务元数据查询：

```text
POST /api/cron/detail/metadata
```

任务明细查询：

```text
POST /api/cron/detail/list
```

两个接口均支持与 CLI 一致的过滤参数。
