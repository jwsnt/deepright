# Remote 迭代 20260530-1 使用手册

## 变更说明

本次迭代把 `remote` 的会话缓存维度从 `agentId + chatId` 扩展为 `agentId + chatId + remote`。

这意味着同一个 `agentId/chatId` 现在可以同时维护多个远程主机连接，不会再因为命中了旧缓存而错误复用到别的主机。

## 命令用法

创建或复用指定主机连接：

```bash
./remote create --agentId xxx --chatId yyy --remote ubuntu@1.2.3.4 --password xxx --port 10086
```

获取指定主机连接：

```bash
./remote get --agentId xxx --chatId yyy --remote ubuntu@1.2.3.4
```

执行指定主机命令：

```bash
./remote exec --session xxx@yyy --remote ubuntu@1.2.3.4 "ls -l /a"
```

关闭指定主机连接：

```bash
./remote shutdown --agentId xxx --chatId yyy --remote ubuntu@1.2.3.4
```

关闭同一 `agentId/chatId` 下全部连接：

```bash
./remote shutdown --agentId xxx --chatId yyy
```

通过 `scp` 自动识别远程主机并复用会话：

```bash
./remote scp /local/path/file.txt ubuntu@43.155.234.33:/remote/path/ --session xxx@yyy
./remote scp ubuntu@43.155.234.33:/remote/path/file.txt . --session xxx@yyy
```

## 行为说明

- `agentId` 和 `chatId` 仍会先归一化为小写
- `remote` 现在参与缓存键、socket/control 文件名和会话查找
- `create` 只会复用同一个 `agentId + chatId + remote` 的存活连接
- `get` 和 `exec` 在同一个 `agentId/chatId` 下存在多个远程主机时，必须带 `--remote`
- `shutdown` 带 `--remote` 时关闭单个连接，不带 `--remote` 时关闭该 `agentId/chatId` 下所有连接
- `scp` 不要求手工再传一次 `--remote`，会从上传或下载参数里的远程路径自动提取

## 与主手册关系

- 主手册：[../../USER_GUIDE.md](../../USER_GUIDE.md)
- 主需求：[../../REQUIREMENT.md](../../REQUIREMENT.md)
- 当前迭代需求：[REQUIREMENT.md](REQUIREMENT.md)
