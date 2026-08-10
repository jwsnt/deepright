# remote 迭代 20260521-4 使用手册

## 本次完成内容

- `exec` 命令新增执行超时控制
- `scp` 命令新增执行超时控制
- 两个命令都支持显式 `--timeout`
- 超时配置统一按插件规范从 `integration connect meta-get --key remote` 读取
- 当未配置、配置为空或配置非法时，`exec` / `scp` 都默认使用 `30000` 毫秒
- 优先级固定为 `--timeout > meta-get > 默认值`

## 配置来源

Remote 按插件规范读取自身配置：

```bash
../integration connect meta-get --key remote
```

返回示例：

```json
{
  "key": "remote",
  "meta": {
    "exec_timeout": 60000,
    "scp_timeout": 90000
  }
}
```

说明：

- `exec_timeout` 单位为毫秒，控制 `remote exec`
- `scp_timeout` 单位为毫秒，控制 `remote scp`
- 不填写、空字符串、非正整数都视为未配置
- 未配置时默认值固定为 `30000` 毫秒

## exec 超时

当执行：

```bash
./remote exec --connect-bin /path/to/integration --session agent-a@chat-001 --timeout 1500 "sleep 10"
```

Remote 会先读取：

- `meta.exec_timeout`

然后把这个超时时间同时传递到：

- manager 到 daemon 的本地请求超时
- daemon 内部真实 `ssh` 子进程的执行超时

说明：

- `--timeout` 单位为毫秒，优先级高于插件配置
- 这样可以避免只有上层请求超时，而底层远程命令仍继续运行
- 当命令超过配置时间，命令会失败并返回明确的 timeout 错误
- 未传 `--connect-bin` 时，不读取 `meta-get`，直接回退默认超时

## scp 超时

当执行：

```bash
./remote scp --connect-bin /path/to/integration ./local.txt ubuntu@1.2.3.4:/tmp/ --session agent-a@chat-001 --timeout 5000
```

Remote 会先读取：

- `meta.scp_timeout`

然后使用该超时去限制本地 `scp` 进程的最长执行时间。

说明：

- `--timeout` 单位为毫秒，优先级高于插件配置
- `remote scp` 会通过 `--session` 复用已缓存 SSH 会话，并保持系统 `scp` 的参数语义
- `--connect-bin` 会被 Remote 自己消费，用于读取插件配置，不会透传给系统 `scp`
- `--timeout` 也会被 Remote 自己消费，不会透传给系统 `scp`
- `--session` 会被 Remote 自己消费，用于定位对应 SSH 主连接，不会透传给系统 `scp`
- 其他 `scp` 参数仍按原顺序原样透传
- 超时后命令会返回退出码 `124`

## 默认行为

如果以下任一情况出现：

- 没有配置 `exec_timeout`
- 没有配置 `scp_timeout`
- 配置为空字符串
- 配置不是正整数
- 当前环境无法成功读取 `meta-get`

则：

- `remote exec` 默认超时为 `30000` 毫秒
- `remote scp` 默认超时为 `30000` 毫秒

## 验收重点

- `remote exec --timeout 1234` 会优先使用 `1234ms`，即使 `meta-get` 里还有 `exec_timeout`
- `remote scp --timeout 1234` 会优先使用 `1234ms`，即使 `meta-get` 里还有 `scp_timeout`
- `integration connect meta-get --key remote` 中配置 `exec_timeout` 后，`remote exec` 按该毫秒数超时
- `integration connect meta-get --key remote` 中配置 `scp_timeout` 后，`remote scp` 按该毫秒数超时
- 未配置时，两个命令都回退为 `30000ms`
- `remote param` 会返回 `[{"exec_timeout":"选填。SSH执行超时。","scp_timeout":"选填。SCP执行超时。"}]`，用于暴露 Remote 运行时可配置项及用途说明

## 对应需求

- [REQUIREMENT.md](REQUIREMENT.md)
