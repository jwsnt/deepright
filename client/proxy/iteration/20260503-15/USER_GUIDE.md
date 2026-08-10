# 20260503-15 使用手册

## 目标

本次迭代为 `proxy` 补齐了三方待处理消息到一次性任务的自动桥接能力：

- 每30秒扫描 `connect add-request` 产生的待处理消息
- 按插件 `name/key` 聚合同一插件的全部待处理消息
- 生成一条立即执行的一次性任务明细
- 同时把对应三方消息状态更新为 `已启动`
- 在任务执行前校验 Agent 是否存在、模型是否已注册且 Token 非空

## 聚合规则

同一个插件当前所有 `status=0` 的待处理消息会被拼接成一条任务内容。

示例：

```text
你好 [image]/a/b/c.png [file]/d/e/f.csv 天气怎么样
```

来源消息可以是：

```text
消息1：你好
消息2：[image]
消息3：[file]
消息4：天气怎么样
```

如果第 2、3 条消息带有 `artifacts`，会自动拼成：

- 图片：`[image]绝对路径`
- 文件：`[file]绝对路径`

## 字段映射

- 任务内容：该插件当前全部待处理消息的拼接字符串
- 模型：来自 `connect meta-create` / `meta-update` 注册的 `model`
- AgentID：来自 `connect meta-create` / `meta-update` 注册的 `agentId`
- 思考模式：来自 `connect meta-create` / `meta-update` 注册的 `thinking`
- 会话ID：优先复用 connect 元数据的 `chatId`
- 执行时间：生成时立即执行
- 任务类型：插件 `key/name`
- `meta_ref`：任务明细中保留本轮被聚合的全部 `add-request` 记录 ID，使用英文逗号拼接
- `META_ID`：执行请求 metadata 中仅写入本轮最后一条 `add-request` 记录 ID

## 执行流程

1. 每30秒扫描 connect 元数据
2. 读取每个插件的 `status=0` 待处理消息
3. 用 CAS 方式把这些消息更新为 `status=1`
4. 生成一条 `cycle=0` 的一次性任务元数据和一条立即执行的任务明细
5. 如果本轮生成了待执行明细，则按插件调用一次 `send --content "<开始执行>可通过新消息更新任务"`
   这里不会写死插件二进制路径，而是先通过 `connect meta-list` 等价配置视图读取该插件的 `callback` 绝对路径，再执行 `<callback> send ...`
6. 执行任务前检查 Agent 和模型 Token
7. 发送到 `/v1/chat/completions`
8. 请求和 SSE 响应写入 `chat_log`
9. 执行完成后把任务明细更新为 `started=3`

## 状态说明

### connect_request.status

- `0`：待处理
- `1`：已启动
- `2`：已完成

### task_detail.started

- `0`：待启动
- `1`：已启动
- `2`：无需启动
- `3`：已完成

## 行为说明

- 同一个插件在同一轮扫描里只生成一条一次性任务明细
- 同一个插件在同一轮扫描里最多只发送一次 `<开始执行>可通过新消息更新任务`
- 开始通知与完成回推都会动态读取 connect 元数据中的 `callback` 绝对路径，不会把插件程序路径写死在 proxy 内
- 如果生成任务失败，会把本轮已锁定的三方消息状态回滚为 `0`
- 如果 Agent 被删除，或模型未注册 / Token 为空，对应任务明细不会执行
- 执行请求 metadata 中会额外包含最后一条原始消息的 `META_ID`

## 验证

本次迭代已补充自动化测试，覆盖：

- 待处理消息聚合为一条一次性任务
- 任务明细写入完整 `meta_ref`
- 执行时 metadata 注入最后一条消息的 `META_ID`
- 生成待执行明细时按插件发送一次 `<开始执行>可通过新消息更新任务`
- 任务执行完成后状态更新为 `started=3`
