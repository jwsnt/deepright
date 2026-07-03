# 20260503-6 User Guide

## 目标

本次迭代为 `proxy` 增加了 cron 删除能力，支持通过 CLI 和 HTTP 删除任务元数据与任务明细。

## CLI 命令

删除任务元数据：

```bash
./proxy cron delete-meta --id meta_1
```

删除指定元数据下全部明细：

```bash
./proxy cron delete-detail --metaId meta_1
```

删除单条明细：

```bash
./proxy delete-detail --detailId detail_1
```

## 删除规则

- `delete-meta` 会同时删除匹配元数据下的全部任务明细
- `delete-detail` 支持 `detailId`、`metaId` 或与查询明细相同的过滤条件
- `metaId` 支持 `1`、`cron_1`、`meta_1`
- `detailId` 支持 `1`、`detail_1`
- 删除明细时，未指定时间条件不会默认限制为未来数据

## HTTP 接口

删除任务元数据：

```text
POST /api/cron/delete
```

删除任务明细：

```text
POST /api/cron/detail/delete
```

两个接口均支持与 CLI 一致的删除过滤参数。
