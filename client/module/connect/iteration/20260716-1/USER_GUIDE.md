# 关联插件与 Remote 配置手册

## 关联插件生命周期

在 Integration 的 `config/config.json` 中配置 `associated` 数组后，列出的插件会随 Integration 启动：

```json
{
  "associated": [
    "remote"
  ]
}
```

- Integration 在 HTTP 服务启动流程中异步执行这些插件的 `start`，不会等待插件进程完成启动。
- 每次关联插件的 `start` 与 `stop` 均会传入 `--connect-bin <integration-bin>`。
- Integration 收到退出信号、调用关闭接口或执行 `integration stop` 时，会停止关联插件；插件失败仅记录日志，不会阻止 Integration 退出。
- 转发请求和 `cli/get` 的 `metadata.plugins` 始终包含 `remote`，并合并当前实时运行的 Connect Meta 插件。例如运行 A 时为 `[A, remote]`，运行 A、B 时为 `[A, B, remote]`。
- `remote` 不再出现在右上角插件扇形菜单，也无法从该入口打开插件浮层；远程技能和 CLI 命令仍可使用。

## Remote 超时

`remote` 从同一份 `config/config.json` 读取以下配置：

```json
{
  "remote": {
    "exec_timeout": 300,
    "scp_timeout": 300
  }
}
```

- `exec_timeout`：SSH `remote exec` 的超时，单位为秒。
- `scp_timeout`：`remote scp` 的超时，单位为秒。
- 字段缺失、为空、无效或非正数时，分别默认 300 秒。
- 命令行 `--timeout` 仍以毫秒为单位，并优先于配置文件。
