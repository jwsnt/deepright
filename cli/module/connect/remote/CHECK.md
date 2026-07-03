# Remote Check

以下集成验证案例必须通过：

1. `./remote create --agentId Agent-A --chatId Chat-001 --remote ubuntu@1.2.3.4 --password secret`
   返回 JSON，且 `agentId/chatId` 已归一化为小写。
2. 同一个 `agentId/chatId` 可以分别为 `ubuntu@1.2.3.4` 和 `ubuntu@1.2.3.5` 创建两条独立连接，互不复用。
3. 父命令退出后，`remote` daemon 仍然存活；随后执行 `./remote get --agentId agent-a --chatId chat-001 --remote ubuntu@1.2.3.4` 仍能成功。
4. `./remote exec --session agent-a@chat-001 --remote ubuntu@1.2.3.4 "echo ok"` 能复用现有连接并返回远程输出。
5. `./remote scp ./local.txt ubuntu@1.2.3.4:/tmp/ --session agent-a@chat-001` 能自动提取 `--remote` 并复用正确连接。
6. `./remote shutdown --agentId agent-a --chatId chat-001 --remote ubuntu@1.2.3.4` 只关闭指定主机连接。
7. `./remote shutdown --agentId agent-a --chatId chat-001` 会关闭该 `agentId/chatId` 下剩余全部连接。
8. 替换 `remote` 二进制后，旧 daemon 不会被误判为当前版本有效连接。
9. `./remote list` 会自动剔除失效 daemon，不返回脏数据。
