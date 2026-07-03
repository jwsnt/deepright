# Integration 迭代手册（20260610-3）

## 本次更新

- `integration` 收口后的 `GET /api/skills?agentId=xxx` 已同步 Proxy 行为
- 返回结果现在固定追加 `__internal_cron`
- 当 `browser` 插件处于开启状态时，返回结果追加 `__internal_browser`
- 当 `remote` 插件处于开启状态时，返回结果追加 `__internal_remote`
- 现有 `agentId` 参数校验与未命中 Agent 的错误行为保持不变

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

- `__internal_cron` 始终返回
- `__internal_browser`、`__internal_remote` 取决于对应插件是否已经开启
- 该行为与本次 Proxy 迭代保持一致

## 同步结果

- `integration/main.go` 已同步 `/api/skills` 返回逻辑
- `integration/main_test.go` 已补充 cron、browser、remote 场景测试
- 本迭代手册对应当前目录下的 `REQUIREMENT.md`
