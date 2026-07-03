# Proxy 迭代手册（20260518-2）

## 本次变更

本次迭代补齐了 cron 任务明细执行时的 `response_schema` metadata 透传。

当 `task_detail.response_schema` 非空时，`proxy` 在转发 `/v1/chat/completions` 请求时，除了继续透传 `response_format` 之外，还会额外在请求体 `metadata` 中附加：

```json
{
  "metadata": {
    "response_schema": "{\"type\":\"object\",\"properties\":{\"summary\":{\"type\":\"string\"}}}"
  }
}
```

## 当前行为

1. cron 定时器开始执行任务明细时，会先读取该明细的 `response_schema`
2. 如果该字段为空，则保持现有行为不变，不新增 `metadata.response_schema`
3. 如果该字段非空，则会把原始 Json String 直接写入转发请求的 `metadata.response_schema`
4. 同时，现有的 `response_format.type=json_schema` 透传行为继续保留，不受影响

## 请求体效果

转发到上游 `/v1/chat/completions` 时，相关字段会同时表现为：

```json
{
  "metadata": {
    "response_schema": "{\"type\":\"object\",\"properties\":{\"summary\":{\"type\":\"string\"}}}"
  },
  "response_format": {
    "type": "json_schema"
  }
}
```

## 适用范围

- `proxy` 内部 cron 执行链路
- 周期任务明细
- 由 connect 待处理消息桥接生成的一次性任务明细

## 兼容性说明

- 本次新增字段只发生在 cron 明细执行转发 `/v1/chat/completions` 时
- 普通外部调用 `/v1/chat/completions` 的 metadata 注入规则不变
- `response_schema` 仍然保持原始 Json String 形式，不会在 `metadata` 中重新解析为对象
- 现有 `response_format` 透传逻辑不变，因此不会破坏当前依赖 structured output 的调用方

## 说明

- 本次迭代手册对应当前目录下的 `REQUIREMENT.md`
- 当前目录交付文件为：`REQUIREMENT.md`、`USER_GUIDE.md`
