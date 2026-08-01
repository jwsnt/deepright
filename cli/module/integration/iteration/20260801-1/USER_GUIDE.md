# Integration 迭代 20260801-1 使用手册

## 默认行为

所有带 `chatId` 的技能读取链路均默认关闭技能。首次读取某个会话的技能目录或技能元数据时，Integration 会把已发现的直属 `SKILL.md` 目录记录为关闭；已有会话中此前未记录的目录也会在下一次读取时补齐。

显式启用会被持久化，因此刷新、重启和后续扫描不会将已启用目录重新关闭。之后新发现的目录仍采用默认关闭状态。状态继续按 `chatId` 隔离，不影响其它会话。

## 接口行为

- `GET /api/files?path=...&chatId=...`：首次返回技能目录时，`skillDisabled` 和 `skillDisabledSelf` 即为 `true`。`hasSkill`、继承关闭字段和原有排序保持不变。
- `GET /api/skills?agentId=...&chatId=...`：默认不返回未启用技能；`@ 技能` 缓存、聊天元数据和 CLI Agent 元数据使用同一过滤结果。
- `POST /api/skill_state`：请求格式不变。`disabled: false` 显式启用当前目录，`disabled: true` 关闭当前目录。父级关闭仍使子级继承关闭。
- 不传 `chatId` 的兼容 `GET /api/skills` 仍返回完整技能列表。

示例：启用当前会话的 `alpha` 技能目录：

```json
{
  "chatId": "chat-1",
  "path": "/Users/demo/agent/A/skills/alpha",
  "disabled": false
}
```
