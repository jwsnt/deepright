# 20260508-1 使用手册

## 目标

本次迭代修正了 `integration stop` 的关闭策略：

- 停止 integration 时，仍会优先尝试停止已配置插件
- 如果插件、子命令或其他内部步骤报错，不再阻断 integration 主进程关闭
- 失败信息只会输出到控制台和生命周期日志，便于排查

## 行为说明

执行：

```bash
./integration stop
```

当前关闭顺序为：

1. 读取 `connect meta-list` 返回的已配置插件
2. 逐个调用插件 `stop`
3. 无论插件是否报错，都继续向 integration 主进程发送退出信号
4. 等待 PID 退出，必要时升级为强制结束

## 失败处理

- 插件不可执行
- 插件 `stop` 返回非零
- 插件桥接命令执行失败
- `meta-list` 读取失败

以上问题都会记为 warning，并输出到：

- `stderr`
- `integration.log`

但 `integration stop` 仍会继续关闭主服务，不会因为单个插件异常而卡住整个关闭流程。

## 示例

当某个插件不可执行时，控制台可能看到：

```text
stop plugin feishu failed: plugin is not executable: /path/to/plugins/feishu
```

此时 integration 仍会继续退出；如果 PID 正常结束，`stop` 命令整体仍然返回成功。

## 兼容说明

- 本次迭代不改变 `integration start`、`serve`、`restart` 的参数格式
- 不改变插件 `start` / `stop` / `status` 的 HTTP API
- 只调整 `integration stop` 在内部错误场景下的收口策略，使其变为“尽力关闭”
