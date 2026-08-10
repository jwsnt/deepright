# 20260503-6 使用手册

## 目标

本次迭代统一了 `integration` 模块的数据库路径规则：

- 所有数据库路径统一使用应用启动目录下的 `data`
- 数据库路径必须是绝对路径
- 启动 HTTP 服务后，会把实际数据库绝对路径写入 `runtime.json`

## 路径规则

假设应用启动目录为：

```text
/home/integration
```

那么数据库路径必须固定为：

```text
/home/integration/data
```

这里的“应用启动目录”指的是：

- `./integration`
- `./integration start`
- `./integration serve`
- `./integration restart`

这些命令真正启动应用时所在的目录。

## runtime.json

启动 HTTP 服务后，当前启动目录下会生成或覆盖 `runtime.json`。

现在其中会明确写入：

```json
{
  "app": "/home/integration/integration",
  "app-dir": "/home/integration",
  "db": "/home/integration/data"
}
```

说明：

- `app` 是实际执行的 `integration` 二进制绝对路径
- `app-dir` 是应用启动目录绝对路径
- `db` 是本次统一后的数据库绝对路径

## 验证方式

假设你在发布目录启动：

```bash
cd /path/to/deepright/cli/module/release
./integration start
```

则应满足：

```text
/path/to/deepright/cli/module/release/data
```

为实际数据库路径。

可以继续验证：

```bash
cd /path/to/deepright/cli/module/release
./integration connect meta-get --key feishu
```

以及从插件目录验证：

```bash
cd /path/to/deepright/cli/module/release/plugins
/path/to/deepright/cli/module/release/integration connect meta-get --key feishu
```

两次都应读取同一份：

```text
/path/to/deepright/cli/module/release/data
```

而不是误读当前工作目录下其他同名 `data` 文件。

## 兼容说明

- 本次迭代不改变数据库表结构
- 也不改变 `connect meta-get`、`plugins config`、cron 执行等业务语义
- 只修正所有数据库加载路径，统一到应用启动目录下的绝对 `data`
- 更完整的启动参数和 CLI 说明请查看上级手册 `../../USER_GUIDE.md`
