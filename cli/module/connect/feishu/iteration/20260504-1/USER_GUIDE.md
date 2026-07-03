# 20260504-1 User Guide

## 目标

本次迭代为 `connect` 增加了 `feishu` 长连接接入能力，并保持和 `connect` 当前服务化设计一致：

- `feishu` 通过独立 CLI 启动和停止
- `feishu` 从 `connect` 服务读取 `name=feishu` 的元数据配置
- `feishu` 收到消息后，通过 `connect` CLI 向 `connect` 服务推送请求
- `feishu` 不直接连接 SQLite，也不直接读取 Agent 目录

## 启动方式

在 `/path/to/deepright/cli/module/connect` 目录编译 `feishu` 时，产物默认放在 `../plugins/feishu`：

```bash
go build -o ../plugins/feishu ./feishu
```

先启动 `connect` 服务：

```bash
./connect start --db ./data --agent-dir ../agent/test-case
```

再启动 `feishu`：

```bash
../plugins/feishu start --connect-bin ./connect
```

停止 `feishu`：

```bash
../plugins/feishu stop --pid-file ../plugins/feishu.pid
```

## 默认行为

- `feishu` 默认读取的连接名称为 `feishu`
- 默认通过本地 `connect` 服务地址 `http://127.0.0.1:18080` 访问 Connect
- 每 60 秒检查一次心跳
- 如果心跳不存在或连接中断，会自动断开并重连
- 每次收到消息后，在调用 `connect` 推送前会先把消息追加写入同目录下的 `feishu.log`
- 日志格式为标准时间加消息内容
- 如果消息创建时间距离当前时间超过 30 分钟，则只记录日志并跳过
- 如果飞书原始报文无法解析出有效消息内容，则只记录日志并跳过
- 文本消息会优先从 `content.text` 中提取真正的消息内容

## 非默认地址

如果 `connect` 服务不在默认地址，可以显式指定：

```bash
../plugins/feishu start --connect-bin ./connect --addr http://127.0.0.1:18081
```

## Mock 验证

可以先在 `connect` 中写入 mock 配置：

```bash
./connect meta-create \
  --name feishu \
  --meta '{"mode":"mock","heartbeatIntervalSec":1,"heartbeatTimeoutSec":1,"reconnectDelayMs":50,"mockHeartbeatMs":100,"mockMessages":[{"delayMs":50,"messageId":"msg-1","chatId":"chat-1","content":"hello from mock"}]}' \
  --stream true \
  --callback ../plugins/feishu \
  --agent A \
  --model OpenAI
```

然后启动：

```bash
../plugins/feishu start --connect-bin ./connect
```

启动后，`feishu` 会从 `connect` 服务取配置、建立 mock 长连接，并把收到的消息重新写回 `connect add-request`。

## 验证链路

启动 `connect`：

```bash
./connect start --db ./data --agent-dir ../agent/test-case
```

注册 `feishu`：

```bash
./connect meta-create \
  --name feishu \
  --meta '{"appId":"aaa","appSecret":"bbb","mode":"feishu"}' \
  --stream true \
  --callback ../plugins/feishu \
  --agent a \
  --model deepseek
```

启动 `feishu`：

```bash
../plugins/feishu start --connect-bin ./connect
```

关闭 `feishu`：

```bash
../plugins/feishu stop --pid-file ../plugins/feishu.pid
```

关闭 `connect`：

```bash
./connect stop
```
