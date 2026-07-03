# Cron 迭代手册（20260524-1）

## 本次变更

本次迭代为 cron 任务元数据与任务明细统一增加了 `swarm` 字段。

字段类型为布尔值，不可为空，默认值为 `false`。

## 当前行为

1. `task_meta` 新增 `swarm`
2. `task_detail` 新增 `swarm`
3. 由周期任务元数据拆分出的任务明细，会自动继承所属 `task_meta.swarm`
4. 一次性任务创建时可选指定 `swarm`
5. 不传时统一保存为 `false`，不影响旧任务

## 使用方式

创建一次性任务时，可选传入 `swarm`：

```bash
./cron create --content "整理日报" --swarm --model OpenAI --rawTime "2026-05-24 09:00" --cycle 0 --agent demo-agent
```

创建周期任务时，也可显式传入 `swarm=true`：

```bash
./cron create-cron --content "提取结构化日报" --swarm true --model OpenAI --cron "0 18 * * 1-5" --agent demo-agent
```

## 继承规则

- 周期任务：
  - `task_meta.swarm` 在拆分明细时直接复制到每条 `task_detail.swarm`
- 一次性任务：
  - 创建时如果传了 `--swarm`，首条明细直接继承该值
- 默认值：
  - 元数据和明细都默认为 `false`

## 兼容性说明

- 旧数据不传 `swarm` 时，行为保持不变
- 历史库升级时会自动补齐 `swarm` 列，默认值为 `0`
- 本次新增字段不会改变既有 cron 拆分规则、查询能力和删除逻辑

## 说明

- 本次迭代手册对应当前目录下的 `REQUIREMENT.md`
- 当前目录交付文件为：`REQUIREMENT.md`、`USER_GUIDE.md`
# 2026-05-24 修订

- 本迭代里原先说明的 `swarm` 现统一改为 `router_disable`。
- `router_disable=true` 表示关闭 SWARM，`router_disable=false` 表示开启 SWARM。
- CLI 仍保留 `--swarm` 名称，但它会反向映射到 `router_disable`。
- 库表与接口都以 `router_disable` 为准；旧 `swarm` 仅用于兼容迁移。
