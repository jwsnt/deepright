# Proxy 迭代手册（20260614-3）

## 本次更新

- `GET /api/skills?agentId=xxx` 仍然先返回 Agent 自身的技能名
- 主应用 `config/config.json` 中新增的 `skills` 数组会按顺序追加到返回结果
- `__internal_cron` 不再由接口硬编码追加，改为完全由主应用 `config/config.json.skills` 控制
- 当 `browser` 插件处于开启状态时，接口会追加 `__internal_browser`
- 当 `remote` 插件处于开启状态时，接口会追加 `__internal_remote`
- 返回结果会自动去重，保留首次出现的顺序

## 配置方式

主应用 `config/config.json` 示例：

```json
{
  "skills": [
    "__internal_cron",
    "__internal_demo"
  ]
}
```

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
  "__internal_demo",
  "__internal_browser",
  "__internal_remote"
]
```

说明：

- `agentId` 仍为必填
- `config/config.json.skills` 中声明了什么，接口就追加什么
- `browser`、`remote` 两个内部技能只会在对应插件实际开启时追加
- 如果 `skills`、Agent 自身技能、插件内部技能之间出现重名，只保留一份

## 同步结果

- `proxy/main.go` 已改为本地组装 `/api/skills` 返回结果
- `proxy/main_test.go` 已改为覆盖 config 驱动技能和运行中插件技能场景
- 本迭代手册对应当前目录下的 `REQUIREMENT.md`
