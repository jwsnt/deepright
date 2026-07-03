# Site 迭代 20260622-1 使用手册

## 本次更新

- 知识库 WIKI 改为 Agent 维度
- 切换 Agent 时，右侧 WIKI 会同步切换到对应 Agent 的知识库首页
- 当前 Agent 的首页文档固定为：
  - `knowledge/<agentId>/index.html`
- 知识库整理小灯泡的过期判断改为按 Agent 调用：
  - `/knowledge_lastUpdate?agentId=...`
- 知识库整理浮层读取的目录改为按 Agent 调用：
  - `/knowledge_path?agentId=...`

## 页面行为

- 页面会为每个 Agent 单独记录 `最近刷新` 与过期提醒关闭状态
- 当切换 Agent 时：
  - 会重新拉取该 Agent 的知识库目录
  - 会重新拉取该 Agent 的 `lastUpdate`
  - 当前文档会重置到该 Agent 的首页 `index.html`
- 通过知识库整理小灯泡触发的请求仍会写入 `metadata.knowledge_commit: true`

## 说明

- 这次变更只调整知识库 WIKI 的 Agent 维度，不改变现有页面技术栈和交互骨架
- 更完整说明请参考上级手册 `../../USER_GUIDE.md`
