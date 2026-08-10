# remote 迭代 20260521-6 使用手册

## 本次完成内容

- 原业务命令 `command` 改为 `exec`
- `param` 返回值移除 `cookie_path`
- `param` 中 `cmd_timeout` 改为 `exec_timeout`
- 新增符合插件规范的 `command` 命令，用于返回插件支持的命令列表

## command 命令

执行：

```bash
./remote command
```

返回示例：

```json
["command","exec","help","name","param","scope","start","stop","create","shutdown","list","get","ssh","scp"]
```

说明：

- 该命令仅用于插件规范探测能力
- 不执行远程命令
- 不触发 daemon 启动

## exec 命令

执行：

```bash
./remote exec --session agent-a@chat-001 "uname -a"
```

说明：

- `exec` 是原来远程执行命令能力的新名称
- 行为与原业务 `command` 一致
- 仍然复用已缓存 SSH 主连接

## scp 命令

执行：

```bash
./remote scp ./local.txt ubuntu@1.2.3.4:/tmp/ --session agent-a@chat-001
```

说明：

- `scp` 也会通过 `--session` 复用已缓存 SSH 主连接
- `--session` 由 Remote 自己消费，不会透传给系统 `scp`

## param 命令

执行：

```bash
./remote param
```

返回：

```json
[{"exec_timeout":"选填。SSH执行超时。","scp_timeout":"选填。SCP执行超时。"}]
```

说明：

- `cookie_path` 已移除
- `exec_timeout` 控制 `remote exec`
- `scp_timeout` 控制 `remote scp`
- 两个字段现在通过带说明的对象数组暴露，方便调用方直接展示参数用途

## 对应需求

- [REQUIREMENT.md](REQUIREMENT.md)
