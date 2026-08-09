# Proxy 迭代手册（20260518-1）

## 本次变更

本次迭代为由 `add-request` 进入 proxy 链路的消息补齐了 `schema` 透传说明。

`add-request` 新可选参数 `--schema` 的值为 Json String，最终对应桥接任务明细的 `task_detail.response_schema`。

## 当前行为

1. `proxy` 通过内嵌的 connect 能力接收 `add-request`
2. `schema` 会先进入 `connect_request.response_schema`
3. 待处理消息桥接为一次性 cron 任务时，会继承最后一条 request 的 `responseSchema`
4. 桥接后生成的 `task_meta` 与 `task_detail` 都会保存 `response_schema`
5. 查询结果与日志也会返回 `responseSchema`

## 代理示例

如果通过 integration 顶层代理调用：

```bash
./integration connect add-request --key feishu --content "提取结构化消息" --schema '{"type":"object","properties":{"reply":{"type":"string"}},"required":["reply"]}'
```

则 proxy 后续桥接出的任务明细会带有：

```json
{
  "responseSchema": "{\"type\":\"object\",\"properties\":{\"reply\":{\"type\":\"string\"}},\"required\":[\"reply\"]}"
}
```

## 兼容性说明

- `--schema` 为可选参数；不传时桥接逻辑保持不变
- 该值保持原始 Json String 形式，不会在 proxy 桥接阶段重新解析
- 本次变更不会破坏原有 connect 待处理消息聚合、cron 任务创建、查询和执行流程

## 说明

- 本次迭代手册对应当前目录下的 `REQUIREMENT.md`
- 当前目录交付文件为：`REQUIREMENT.md`、`USER_GUIDE.md`
