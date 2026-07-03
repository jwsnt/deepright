# FEISHU CLI

`feishu` 是 `connect` 模块里的独立编译 CLI，用来读取 `name=FEISHU` 的 Connect 元数据，建立飞书长连接并把收到的消息推回 Connect。

## 独立编译

在 `/Users/shenjiawei/DEV/code_gen/cli/module/connect` 目录执行：

```bash
go build -o ../plugins/feishu ./feishu
```

也可以进入当前目录直接编译：

```bash
cd /Users/shenjiawei/DEV/code_gen/cli/module/connect/feishu
go build -o ../../plugins/feishu .
```

## 最小启动示例

先启动 `connect` 服务：

```bash
../connect start --db ../data --agent-dir ../../agent/test-case
```

再启动 `feishu`：

```bash
../plugins/feishu start \
  --connect-bin ./connect
```

`feishu` 运行时只会通过 `connect` CLI 去读取 `name=FEISHU` 的配置并推送消息，不会直接读取数据库或访问 Agent 目录。

如果需要从上层 HTTP 服务读取插件元数据，也可以通过：

- `proxy`：`GET /api/plugins/meta`
- `integration`：`GET /api/plugins/meta`

这两个接口底层都会复用 `connect list-plugins` 的插件发现逻辑，因此会把 `feishu` 暴露为可用插件。

## 停止示例

```bash
../plugins/feishu stop --pid-file ../plugins/feishu.pid
```
