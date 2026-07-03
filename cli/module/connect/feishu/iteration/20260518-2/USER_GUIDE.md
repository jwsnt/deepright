# Feishu 迭代手册（20260518-3）

## 本次变更

本次迭代补齐了飞书插件调用 `add-request` 时的 `schema` 透传。

飞书插件在接收到新消息并调用 `integration connect add-request` 时，会复用自身 `feishu schema` 命令对应的同一份 Response JSON Schema，并通过 `--schema` 参数传递给 integration 顶层入口。

## 当前行为

1. 飞书插件收到消息后，仍然会先整理消息正文、附件路径和原始报文
2. 调用 `integration connect add-request` 时，会额外附加 `--schema`
3. `--schema` 的值与 `feishu schema` 命令返回值语义完全一致
4. integration 持久化时会写入 `connect_request.response_schema`
5. 后续待处理消息桥接为一次性 cron 任务时，会继续透传到 `task_meta.response_schema` 和 `task_detail.response_schema`

## 设计说明

- 飞书插件不额外发明第二份 schema 协议
- 对外声明使用 `feishu schema`
- 对内提交请求时也使用同一份 schema 数据
- schema 中 `content.description` 固定为 `MARKDOWN FORMAT` 说明
- 这样可以保证插件声明、CLI 输出和后续执行链路的 `response_schema` 语义完全一致

## 接收链路示例

飞书插件收到消息后，最终会走：

```bash
./integration connect add-request \
  --key feishu \
  --externalId <md5(create_time+content)> \
  --content <归一化后的文本> \
  --artifacts <附件绝对路径，逗号分隔> \
  --original <原始飞书事件 JSON> \
  --created <消息时间戳> \
  --schema <feishu schema输出>
```

其中 `--schema` 的值与下列命令返回值完全一致：

```bash
../plugins/feishu schema
```

## 验证方式

先创建 meta 并启动飞书插件：

```bash
./integration connect meta-create --name feishu --meta '{"appId":"x","appSecret":"y"}' --callback ./feishu --agent a --model deepseek
./integration plugins start --name feishu
```

也可以先确认插件公开的 schema：

```bash
../plugins/feishu schema
```

随后检查飞书插件触发的 `integration connect add-request` 链路，预期行为为：

- 会向 `add-request` 传递 `--schema`
- 传递值与 `feishu schema` 的输出保持一致
- request 持久化结果中可看到 `responseSchema`

## 兼容性说明

- 本次新增参数仅在飞书插件调用 `add-request` 时附加
- 未传 `schema` 的旧请求仍可继续处理
- 该值保持原始 Json String 语义，不在飞书插件侧重新解释

## 说明

- 本次迭代手册对应当前目录下的 `REQUIREMENT.md`
- 当前目录交付文件为：`REQUIREMENT.md`、`USER_GUIDE.md`
