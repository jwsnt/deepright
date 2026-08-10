# 20260613-2 USER_GUIDE

本次迭代收紧了模型与密钥接口的远程读取行为，避免非本机访问时直接看到真实密钥。

## `/api/token` 读取规则

- 当请求来源是 `localhost` 或 `127.0.0.1` 时，`GET /api/token` 仍返回真实 `token`
- 当请求来源不是 `localhost` 且不是 `127.0.0.1` 时，`GET /api/token` 返回中的每个 `token` 都会被替换成固定的 `**********`
- 这种替换只发生在接口读取阶段，不会修改共享 SQLite `token_store` 中真实保存的密钥

## 新增与使用

- `POST /api/token` 在新增或更新模型时，仍按原样接收并返回真实 `token`
- integration 在执行模型请求、查询已保存模型配置、按模型注入运行时 metadata 时，仍使用数据库中的真实密钥，不会把 `**********` 当成实际 token

## 影响范围

- 本次迭代除了调整模型与密钥的 HTTP 读取返回值外，也收紧了远程插件管理接口
- 不改变模型密钥的存储结构
- 不改变 CLI `integration token` 的输出行为

## 远程插件接口限制

- 这里按访问入口 Host 判断远程；通过非 `localhost` / `127.0.0.1` / `::1` 的域名、别名、LAN IP 或反代入口访问时，也会按远程模式处理
- 远程请求仍可读取：
- `GET /api/plugins/meta`
- `GET /api/plugins/status`
- `GET /api/plugins/log`
- 其中 `/api/plugins/meta` 只返回脱敏后的插件列表，不暴露已配置参数等运行期信息
- 会直接返回 `403` 的接口包括：
- `POST /api/plugins/config`
- `GET /api/plugins/exec`
- `POST /api/plugins/start`
- `POST /api/plugins/stop`
- 这意味着远程访问者仍能看插件列表、状态和日志，但不能读取敏感配置、执行插件命令、启动插件或关闭插件
