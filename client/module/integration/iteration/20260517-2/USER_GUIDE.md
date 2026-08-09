# Integration 迭代手册（20260517-2）

## 本次变更

本次迭代为 `integration connect add-request` 收口了新可选参数 `--schema`。

该参数的值为 Json String，对应后续桥接任务明细的 `task_detail.response_schema`。

## 当前行为

1. `./integration connect add-request` 支持新可选参数 `--schema`
2. `--schema` 会先通过 integration 顶层入口透传给底层 connect 能力
3. connect 持久化时会写入 `connect_request.response_schema`
4. 当待处理消息后续桥接成一次性 cron 任务时，会透传到 `task_meta.response_schema` 和 `task_detail.response_schema`
5. cron 明细执行时，这份值还会继续参与后续 structured output 透传

## 参数说明

- `--schema` 为可选参数
- 参数值类型为 Json String
- integration 不重新定义 schema 内容，只负责按顶层 CLI 收口原则向下透传
- 因此调用方应直接传递插件或上游模块定义好的原始 Response JSON Schema

## CLI 示例

```bash
./integration connect add-request --key feishu --externalId msg-1 --content "HELLO WORLD"
./integration connect add-request --key feishu --content "提取结构化消息" --schema '{"type":"object","properties":{"title":{"type":"string"}},"required":["title"]}'
```

## 收口说明

- `integration` 继续作为顶层二进制入口收口 `add-request`
- 不需要切换到 `connect` 二进制，也不需要直接操作数据库
- 该变更遵循 integration 的 CLI 收口原则，底层仍复用共享 connect/service 逻辑

## 兼容性说明

- `--schema` 为可选参数，不传时保持原有行为
- integration 只是顶层适配层，不单独复制一份 connect 逻辑
- 本次新增字段不会改变原有 `add-request`、`request-list`、`add-response`、`response-list` 主流程
- request-list 和后续桥接结果会继续保留该字段，便于后续执行链路复用

## 说明

- 本次迭代手册对应当前目录下的 `REQUIREMENT.md`
- 当前目录交付文件为：`REQUIREMENT.md`、`USER_GUIDE.md`
