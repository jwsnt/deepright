# 20260507-3 使用手册

## 目标

本次迭代补充 `connect` 模块的使用说明，重点明确两件事：

- 最终用户验收与日常使用优先走 `integration` 顶层入口
- `connect` 在独立运行或嵌入 `integration` / `proxy` 时，都会复用启动阶段初始化的数据库连接，不能在每次请求时单独打开和关闭 SQLite 文件

## 推荐入口

最终用户优先在 `/path/to/deepright/cli/module/integration` 目录使用顶层命令：

```bash
./integration connect meta-create --name "飞书" --meta '{"appId":"cli-app","appSecret":"cli-secret"}' --callback ../plugins/feishu --agent A --model OpenAI
./integration connect meta-list
./integration connect add-request --key feishu --content "HELLO WORLD"
./integration connect add-response --name feishu --request-id 1 --response "HELLO BACK"
```

`connect` 自身命令仍然保留，主要用于内部实现联调、兼容说明和排查：

```bash
./connect help
```

## 帮助命令

`./connect help` 会展示当前 `connect` 支持的主要命令，包括：

- `start` / `stop` / `serve`
- `meta-create` / `meta-update` / `meta-delete` / `meta-get` / `meta-list`
- `list-meta` / `list-plugins`
- `add-request` / `request-list`
- `add-response` / `response-list`

其中请求写入相关参数以当前实现为准，推荐优先使用：

```bash
./connect add-request --key feishu --externalId msg-1 --content "HELLO WORLD" --original '{"text":"HELLO WORLD"}'
```

## 连接复用说明

- `connect start` 启动后，会初始化并持有 SQLite 连接池
- 后续 HTTP 请求、`meta-create`、`add-request`、`add-response` 等操作都会复用这份已初始化的数据库连接
- `connect` 被 `integration` 或 `proxy` 以内嵌方式调用时，也复用进程启动阶段准备好的 `connectsvc.Service`
- 不允许为单次请求单独打开数据库文件，再在请求结束后立即关闭

这意味着：

- 请求链路更稳定
- 避免高频打开/关闭 `data` 文件
- 与 `cron` 共用 SQLite 时仍保持统一连接池语义

## 验证方式

可以用下面方式快速确认本次迭代关注点：

```bash
cd /path/to/deepright/cli/module/connect
./connect help
./connect start --foreground true --db ./data --agent-dir ../agent/test-case
```

另一个终端执行：

```bash
./connect list-meta
./connect add-request --key feishu --content "HELLO WORLD"
```

行为预期：

- 命令通过本地 HTTP 服务访问 `connect`
- 未直接操作 SQLite 文件
- 多次请求复用同一份启动后初始化的数据库连接

## 兼容说明

- `connect help` 是内部实现和兼容说明入口，不替代 `integration` 顶层文档
- 顶层完整说明仍以 `/path/to/deepright/cli/module/connect/USER_GUIDE.md` 和 `integration/USER_GUIDE.md` 为准
- 本迭代不改变既有命令语义，只补充连接复用与文档入口约定
