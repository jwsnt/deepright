# Remote User Guide

`remote` 是交付给最终用户的唯一二进制文件，用来管理按 `agentId + chatId + remote` 维度缓存的 SSH 长连接，并通过该连接执行远程命令。

## 需求目录

- 主需求文档：[REQUIREMENT.md](REQUIREMENT.md)
- 当前手册：[USER_GUIDE.md](USER_GUIDE.md)
- 迭代需求目录：[/path/to/deepright/cli/module/connect/remote/iteration](iteration)

当前迭代需求清单：

- [20260521-1/REQUIREMENT.md](iteration/20260521-1/REQUIREMENT.md)
- [20260521-2/REQUIREMENT.md](iteration/20260521-2/REQUIREMENT.md)
- [20260521-3/REQUIREMENT.md](iteration/20260521-3/REQUIREMENT.md)
- [20260521-4/REQUIREMENT.md](iteration/20260521-4/REQUIREMENT.md)
- [20260521-5/REQUIREMENT.md](iteration/20260521-5/REQUIREMENT.md)
- [20260521-6/REQUIREMENT.md](iteration/20260521-6/REQUIREMENT.md)
- [20260530-1/REQUIREMENT.md](iteration/20260530-1/REQUIREMENT.md)
- [20260610-1/REQUIREMENT.md](iteration/20260610-1/REQUIREMENT.md)

当前迭代手册清单：

- [20260521-1/USER_GUIDE.md](iteration/20260521-1/USER_GUIDE.md)
- [20260521-2/USER_GUIDE.md](iteration/20260521-2/USER_GUIDE.md)
- [20260521-3/USER_GUIDE.md](iteration/20260521-3/USER_GUIDE.md)
- [20260521-4/USER_GUIDE.md](iteration/20260521-4/USER_GUIDE.md)
- [20260521-5/USER_GUIDE.md](iteration/20260521-5/USER_GUIDE.md)
- [20260521-6/USER_GUIDE.md](iteration/20260521-6/USER_GUIDE.md)
- [20260530-1/USER_GUIDE.md](iteration/20260530-1/USER_GUIDE.md)
- [20260610-1/USER_GUIDE.md](iteration/20260610-1/USER_GUIDE.md)

需求文件结构：

```text
remote/
├── REQUIREMENT.md
├── USER_GUIDE.md
└── iteration/
    ├── 20260521-1/
    │   ├── REQUIREMENT.md
    │   └── USER_GUIDE.md
    ├── 20260521-2/
    │   ├── REQUIREMENT.md
    │   └── USER_GUIDE.md
    ├── 20260521-3/
    │   ├── REQUIREMENT.md
    │   └── USER_GUIDE.md
    ├── 20260521-4/
    │   ├── REQUIREMENT.md
    │   └── USER_GUIDE.md
    ├── 20260521-5/
    │   ├── REQUIREMENT.md
    │   └── USER_GUIDE.md
    ├── 20260521-6/
    │   ├── REQUIREMENT.md
    │   └── USER_GUIDE.md
    ├── 20260530-1/
    │   ├── REQUIREMENT.md
    │   └── USER_GUIDE.md
    └── 20260610-1/
        ├── REQUIREMENT.md
        └── USER_GUIDE.md
```

需求职责说明：

- [REQUIREMENT.md](REQUIREMENT.md)
  负责定义 `remote` 插件的主能力边界、CLI 收口、生命周期、日志和构建交付要求。
- [20260521-1/REQUIREMENT.md](iteration/20260521-1/REQUIREMENT.md)
  负责新增 `scp` 命令，支持本地上传和远程下载。
- [20260521-1/USER_GUIDE.md](iteration/20260521-1/USER_GUIDE.md)
  负责说明 `scp` 命令的用法、方向和与系统 `scp` 的一致性。
- [20260521-2/REQUIREMENT.md](iteration/20260521-2/REQUIREMENT.md)
  负责新增空闲回收机制：每分钟巡检一次，15 分钟无活动的 SSH 连接会被主动关闭，并记录到 `remote.log`。
- [20260521-2/USER_GUIDE.md](iteration/20260521-2/USER_GUIDE.md)
  负责说明空闲回收的巡检周期、关闭条件和日志行为。
- [20260521-3/REQUIREMENT.md](iteration/20260521-3/REQUIREMENT.md)
  负责为 `create` 增加 `--certificate` 参数，支持通过 PEM 证书创建 SSH 连接。
- [20260521-3/USER_GUIDE.md](iteration/20260521-3/USER_GUIDE.md)
  负责说明 `--certificate` 的命令写法、证书认证模式和与密码模式的关系。
- [20260521-4/REQUIREMENT.md](iteration/20260521-4/REQUIREMENT.md)
  负责为 `exec` 和 `scp` 增加毫秒级超时控制，支持 `--timeout > meta-get > 默认值` 的优先级。
- [20260521-4/USER_GUIDE.md](iteration/20260521-4/USER_GUIDE.md)
  负责说明 `exec_timeout`、`scp_timeout`、显式 `--timeout` 和默认 `30000ms` 的行为。
- [20260521-5/REQUIREMENT.md](iteration/20260521-5/REQUIREMENT.md)
  负责为 `start` / `stop` 增加生命周期日志，并将 `scope` 命令固定为返回空数组。
- [20260521-5/USER_GUIDE.md](iteration/20260521-5/USER_GUIDE.md)
  负责说明 `scope=[]` 的契约，以及 `start` / `stop` 在 `remote.log` 中的日志行为。
- [20260521-6/REQUIREMENT.md](iteration/20260521-6/REQUIREMENT.md)
  负责将远程执行命令从 `command` 改为 `exec`，并新增符合插件规范的 `command` 命令。
- [20260521-6/USER_GUIDE.md](iteration/20260521-6/USER_GUIDE.md)
  负责说明 `exec`、插件规范 `command` 以及 `exec_timeout` 的行为。
- [20260530-1/REQUIREMENT.md](iteration/20260530-1/REQUIREMENT.md)
  负责把会话缓存维度从 `agentId + chatId` 扩展为 `agentId + chatId + remote`，并让 `scp` 自动从命令参数中提取远程主机。
- [20260530-1/USER_GUIDE.md](iteration/20260530-1/USER_GUIDE.md)
  负责说明多远程主机共用同一 `agentId/chatId` 时的 `create/get/exec/shutdown/scp` 用法。
- [20260610-1/REQUIREMENT.md](iteration/20260610-1/REQUIREMENT.md)
  负责把 `param` 命令改为固定返回带中文说明的参数描述对象。
- [20260610-1/USER_GUIDE.md](iteration/20260610-1/USER_GUIDE.md)
  负责说明新的 `param` 固定返回格式及字段含义。

## 核心能力

- `name`：返回插件标识 `{"key":"remote","name":"远程"}`
- `param`：返回插件参数说明 `[{"exec_timeout":"选填。SSH执行超时。","scp_timeout":"选填。SCP执行超时。"}]`
- `command`：返回符合插件规范的能力列表
  只暴露对外命令，不包含 `__daemon`、`__manager` 这类内部运行态命令
- `scope`：返回空数组 `[]`
- `start`：启动 `remote` 管理进程并写入 `remote.pid`
- `stop`：停止管理进程并清理所有 SSH 子进程
- `create`：创建或复用一个远程 SSH 连接
  支持 `--password` 或 `--certificate /path/to/id.pem`
- `get`：获取指定会话的连接信息
- `list`：列出当前所有有效连接
- `exec`：通过已缓存连接执行远程系统命令
- `shutdown`：关闭指定会话连接
- `ssh`：直通本机 `ssh` 命令
- `scp`：直通本机 `scp` 命令，支持本地上传和远程下载
  需要通过 `--session` 复用已缓存 SSH 会话
- 空闲回收：后台 manager 每分钟巡检一次，连续 15 分钟无活动的 SSH 连接会被自动关闭

## 常用命令

```bash
./remote start
./remote param
./remote create --agentId Agent-A --chatId Chat-001 --remote ubuntu@1.2.3.4 --password secret --port 22
./remote create --agentId Agent-A --chatId Chat-002 --remote ubuntu@1.2.3.4 --certificate /path/to/id.pem --port 22
./remote create --agentId Agent-A --chatId Chat-001 --remote ubuntu@1.2.3.5 --password secret --port 22
./remote get --agentId agent-a --chatId chat-001 --remote ubuntu@1.2.3.4
./remote list
./remote command
./remote exec --session agent-a@chat-001 --remote ubuntu@1.2.3.4 "uname -a"
./remote scp ./local.txt ubuntu@1.2.3.4:/tmp/ --session agent-a@chat-001
./remote scp ubuntu@1.2.3.4:/tmp/local.txt . --session agent-a@chat-001
./remote shutdown --agentId agent-a --chatId chat-001 --remote ubuntu@1.2.3.4
./remote shutdown --agentId agent-a --chatId chat-001
./remote stop
```

## 运行时文件

- `remote.log`：固定写在 `remote` 二进制同目录
- `remote.json`：固定写在 `remote` 二进制同目录
- `remote.pid`：固定写在插件运行根目录；如果通过 `integration` 触发，则固定落到 `integration/plugins/remote.pid`
- `.remote/`：内部 daemon socket、control socket、临时密钥文件目录
- `plugins/`：固定为启动目录下的插件目录，不做候选回退

## 行为说明

- `agentId` 和 `chatId` 在写入、查找、执行前都会转为小写
- `remote` 会参与会话缓存键；同一个 `agentId/chatId` 现在可以同时缓存多个远程主机
- `create/list/get/shutdown/exec` 会通过后台管理进程执行；如果管理进程未启动，会先自动拉起
- 已存在且仍归属当前 `remote` 二进制的有效 daemon 会直接复用，不会重复创建
- 会话有效性不只依赖 PID，还会校验 daemon socket 返回的会话信息和二进制指纹，避免误连旧版本残留进程
- `command` 固定返回插件支持的命令列表，供 `integration` 等容器按插件规范探测能力
- `param` 固定返回一个只含单个对象的数组，用中文说明 `exec_timeout` 与 `scp_timeout` 都是选填项
- `get` 和 `exec` 在同一个 `agentId/chatId` 下存在多个远程主机时，需要显式传入 `--remote`
- `exec` 通过缓存的 SSH 主连接执行命令，不要求用户再次输入密码
- `create` 既支持密码认证，也支持 `--certificate` 指定的 PEM 证书认证；证书模式等价于系统 `ssh -i /path/to/id.pem user@host -p PORT`
- `shutdown` 传 `--remote` 时只关闭对应主机；不传 `--remote` 时会关闭该 `agentId/chatId` 下的全部远程连接
- `scp` 通过 `--session` 复用已缓存 SSH 主连接，会自动从上传/下载参数里提取远程主机，并保持系统 `scp` 的参数和方向语义
- `scope` 固定返回 `[]`，表示 Remote 当前不暴露容器通用配置项
- manager daemon 会每分钟检查全部 SSH 会话；若某连接最近 15 分钟都没有任何活动，则会主动关闭并将关闭记录写入 `remote.log`
- daemon 以真正脱离前台命令生命周期的独立进程方式启动，避免父进程 stdout/stderr 关闭后连带退出
- `start` 和 `stop` 会把命令生命周期写入 `remote.log`

## 构建

```bash
./build.sh
```

构建结果输出到 `./release/remote`，并会清理历史运行态文件。
