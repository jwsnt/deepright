# Integration 迭代 20260704-1 使用手册

## 变更说明

- `integration` 新增了虚拟文件系统技能目录的会话级禁用能力
- 技能目录以直属包含 `SKILL.md` 的目录为单位；禁用后，该目录及其子孙技能都会在当前 `chatId` 下失效
- 这组状态会持久化到共享 sqlite，刷新页面或重新进入同一会话后仍然保留，但不会影响其他会话
- `GET /api/skills?agentId=xxx&chatId=yyy` 会按当前会话禁用状态过滤技能列表
- `GET /api/files?path=xxx&chatId=yyy` 会返回技能目录禁用状态字段
- `POST /api/skill_state` 用于切换某个技能目录在当前会话中的禁用或恢复状态

## `/api/files`

请求示例：

```text
GET /api/files?path=/Users/demo/agent/A/skills&chatId=chat-1
```

目录项中的新增字段：

- `hasSkill`：当前目录直属是否存在 `SKILL.md`
- `skillDisabled`：当前目录是否处于禁用态
- `skillDisabledSelf`：当前目录是否为自身禁用
- `skillDisabledInherited`：当前目录是否因父级目录被禁用而继承禁用

## `/api/skill_state`

请求示例：

```json
{
  "chatId": "chat-1",
  "path": "/Users/demo/agent/A/skills/alpha",
  "disabled": true
}
```

返回示例：

```json
{
  "status": 0,
  "chatId": "chat-1",
  "path": "/Users/demo/agent/A/skills/alpha",
  "disabled": true,
  "disabledSelf": true,
  "disabledInherited": false,
  "disabledPaths": [
    "/Users/demo/agent/A/skills/alpha"
  ]
}
```

## 作用范围

- 仅传入 `chatId` 的读取链路会应用这组禁用状态
- 不传 `chatId` 时，`/api/skills` 仍保持原有完整返回
- 如果父级技能目录已禁用，子级目录不会重复写入状态表，而是自动表现为继承禁用
