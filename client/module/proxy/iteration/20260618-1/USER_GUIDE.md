# Proxy 迭代手册（20260618-1）

## 本次更新

- `GET /api/skills?agentId=xxx` 会先返回 Agent 自身技能名
- 主应用 `config/config.json.skills` 会按顺序追加到结果
- `__internal_cron` 不再硬编码，改为完全由 `config/config.json.skills` 控制
- `browser` 插件处于已开启状态时，结果追加 `__internal_browser`
- `remote` 插件处于已开启状态时，结果追加 `__internal_remote`
- `/api/cmd` 的沙盒 helper 路径解析新增 WSL/Linux 产物支持，mac 路径保持不变

## 示例

主应用 `config/config.json`：

```json
{
  "skills": [
    "__internal_cron",
    "__internal_demo"
  ]
}
```

请求：

```text
GET /api/skills?agentId=A
```

当 `browser`、`remote` 插件均已启动时，返回示例：

```json
[
  "__internal_F",
  "__internal_cron",
  "__internal_demo",
  "__internal_browser",
  "__internal_remote"
]
```

## 同步结果

- `proxy/main.go` 继续保持按插件运行状态动态追加内部技能
- `proxy/main.go` 的沙盒 helper 路径解析同时支持：
  - mac `.app/Contents/MacOS/CLI_SANDBOX`
  - WSL/Linux `helpers/<mode>/CLI_SANDBOX`
- `proxy/main_test.go` 已补充 WSL/Linux helper 路径解析测试
