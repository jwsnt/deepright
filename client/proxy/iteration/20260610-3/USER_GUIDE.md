# Proxy 迭代手册（20260610-3）

## 本次更新

- `GET /api/skills?agentId=xxx` 继续返回 Agent 自身技能名
- 返回结果现在固定追加 `__internal_cron`
- 当 `browser` 插件处于开启状态时，返回结果追加 `__internal_browser`
- 当 `remote` 插件处于开启状态时，返回结果追加 `__internal_remote`
- 已保持原有参数校验与 `404` 行为不变

## HTTP 用法

请求：

```text
GET /api/skills?agentId=A
```

返回示例：

```json
[
  "__internal_F",
  "__internal_cron",
  "__internal_browser",
  "__internal_remote"
]
```

说明：

- `agentId` 仍为必填
- `__internal_cron` 不依赖插件，始终存在
- `browser`、`remote` 两个内部技能只会在对应插件已开启时返回

## 同步结果

- `proxy/main.go` 已更新 `/api/skills` 返回逻辑
- `proxy/main_test.go` 已补充 cron、browser、remote 场景测试
- 本迭代手册对应当前目录下的 `REQUIREMENT.md`
