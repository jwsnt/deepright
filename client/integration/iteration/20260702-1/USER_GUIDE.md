# 迭代说明

本次迭代为 `integration` 增加了 Agent 复制能力，用于把一个已存在 Agent 的受管目录同步到另一个已创建的 Agent。

## 新增能力

- 新增 `GET /api/copy?source_agentId=xxx&target_agentId=yyy`
- 新增 CLI 收口：`integration agent copy --source SOURCE --target TARGET`
- 复制范围固定为：
  - Agent 工作目录下的 `app/`、`data/`、`skills/`、`SOUL.md`、`USER.md`、`Knowledge.md` / `knowledge.md`
  - 同级知识库目录下的 `knowledge/<agentId>`

## 复制规则

- `target_agentId` 必须已经存在；接口不会自动创建 target Agent
- 如果 source Agent 同时存在 `Knowledge.md` 和 `knowledge.md`，则优先同步 `Knowledge.md`
- target 上同名受管路径会先按 source 状态重建
- 如果 source 缺失某个受管路径，对应 target 路径会被删除
- target Agent 的 `config.json` 不会被覆盖

## 适用场景

- Site 先调用 `/api/agent/init` 创建 target Agent，再调用 `/api/copy`
- CLI 场景可直接用 `integration agent copy` 复用同一套逻辑
- 其他模块也可以直接复用 `integration/agentcopy` 子包
