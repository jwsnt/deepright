# Remote 迭代 20260521-2 使用手册

## 变更说明

本次迭代为 `remote` 增加空闲回收能力。

`remote` 的 manager daemon 会每分钟巡检一次全部受管 SSH 会话；如果某个连接最近 15 分钟都没有任何活动，则会主动关闭该连接，并把关闭记录写入 `remote.log`。

## 回收规则

- 巡检周期：每分钟一次
- 空闲阈值：15 分钟
- 活动来源：
  - `create` 创建或复用会话
  - `get` 获取指定会话
  - `exec` 通过缓存 SSH 会话执行远程命令
  - `scp` 通过 `--session` 复用缓存 SSH 会话执行文件传输
- 若超过阈值未活动：
  - 主动关闭对应 SSH 主连接
  - 清理相关 socket / control 运行时文件
  - 从 `remote.json` 中移除该会话
  - 在 `remote.log` 追加关闭日志

## 日志说明

日志写入文件：

```text
remote.log
```

典型日志会包含：

- `agentId`
- `chatId`
- `pid`
- `ssh`
- `idle`

用于说明是哪一条会话因空闲超时被关闭。

## 与主手册关系

- 主手册：[../../USER_GUIDE.md](../../USER_GUIDE.md)
- 主需求：[../../REQUIREMENT.md](../../REQUIREMENT.md)
- 当前迭代需求：[REQUIREMENT.md](REQUIREMENT.md)
