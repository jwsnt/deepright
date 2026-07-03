# Cron 迭代手册（20260518-1）

## 本次变更

本次迭代为 cron 任务元数据与任务明细统一增加了 `response_schema` 字段。

字段类型为字符串，可为空字符串 `""`。

## 当前行为

1. `task_meta` 新增 `response_schema`
2. `task_detail` 新增 `response_schema`
3. 由周期任务元数据拆分出的任务明细，会自动继承所属 `task_meta.response_schema`
4. 一次性任务在创建时可以可选指定 `response_schema`
5. 不传时默认保存为空字符串，不影响旧任务

## 使用方式

创建一次性任务时，可选传入 `schema`：

```bash
./cron create --content "整理日报" --schema '{"type":"object","properties":{"summary":{"type":"string"}},"required":["summary"]}' --model OpenAI --rawTime "2026-05-08 09:00" --cycle 0 --agent demo-agent
```

创建周期任务时，可选传入 `schema`：

```bash
./cron create-cron --content "提取结构化日报" --schema '{"type":"object","properties":{"todo":{"type":"array"}},"required":["todo"]}' --model OpenAI --cron "0 18 * * 1-5" --agent demo-agent
```

## 继承规则

- 周期任务：
  - `task_meta.response_schema` 在拆分明细时直接复制到每条 `task_detail.response_schema`
- 一次性任务：
  - 创建时如果传了 `schema`，首条明细直接继承该值
- 空值：
  - 元数据和明细都允许为空字符串

## 兼容性说明

- 旧数据不传 `response_schema` 时，行为保持不变
- 本次新增字段不会改变既有 cron 拆分规则、查询能力和删除逻辑
- 字段设计保持为简单 string，便于 `integration` / `proxy` 继续透传

## 说明

- 本次迭代手册对应当前目录下的 `REQUIREMENT.md`
- 当前目录交付文件为：`REQUIREMENT.md`、`USER_GUIDE.md`
