# Proxy 迭代手册（20260606-1）

## 本次更新

- 新增 `GET /api/plugins/exec`
- 新增 `proxy plugins exec` CLI 子命令
- `command` 支持多级子命令文本，例如 `instance init`
- 除 `key`、`command` 之外的 query / CLI 参数都会转成 `--flag value` 透传给插件
- 如果没有显式传 `connect-bin`，proxy 会自动补齐当前 `proxy` 二进制路径

## HTTP 用法

请求：

```text
GET /api/plugins/exec?key=browser&command=instance%20init&agentId=A&chatId=chat-001
```

说明：

- 只支持 `GET`
- `key` 必填
- `command` 必填
- `command` 里的空格需要按 URL 规则转义
- 其他参数数量不限，会按名字排序后透传给插件

## CLI 用法

```bash
./proxy plugins exec --key browser --command 'instance init' --agentId A --chatId chat-001
```

## 同步结果

- `proxy/main.go` 已新增 `/api/plugins/exec` 路由与 `plugins exec` CLI
- `proxy/USER_GUIDE.md` 已同步补充接口与 CLI 说明
- 本迭代手册对应当前目录下的 `REQUIREMENT.md`
