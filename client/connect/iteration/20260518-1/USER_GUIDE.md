# Connect 迭代手册（20260518-1）

## 本次变更

本次迭代为 `connect add-request` 补充了新可选参数 `--schema`。

该参数的值为 Json String，用于保存 Response JSON Schema，并在后续被 `proxy` / `integration` 桥接为一次性 cron 任务时，继续透传到任务明细的 `task_detail.response_schema`。

## 当前行为

1. `add-request` 继续支持原有参数：`--key`、`--externalId`、`--content`、`--artifacts`、`--original`、`--status`、`--created`
2. 新增可选参数 `--schema`
3. `--schema` 会先持久化到 `connect_request.response_schema`
4. 当 connect 待处理消息被桥接成一次性 cron 任务时，会继承最后一条 request 的 `responseSchema`
5. 桥接后的任务明细会把这份值写入 `task_detail.response_schema`

## CLI 示例

```bash
./connect add-request --key feishu --externalId msg-1 --content "HELLO WORLD"
./connect add-request --key feishu --content "提取结构化消息" --schema '{"type":"object","properties":{"reply":{"type":"string"}},"required":["reply"]}'
```

## 兼容性说明

- `--schema` 为可选参数；不传时行为与旧版本一致
- `--schema` 保持原始 Json String 形式保存，不会在 connect 内部重新解析为对象
- 本次变更只扩展请求结构，不改变已有 `add-request`、`request-list`、`add-response`、`response-list` 的主流程
- connect 继续复用启动时初始化的共享数据库连接，不会因为新增字段而改回“每次请求单独开关数据库”

## 说明

- 本次迭代手册对应当前目录下的 `REQUIREMENT.md`
- 当前目录交付文件为：`REQUIREMENT.md`、`USER_GUIDE.md`
