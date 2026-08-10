# Integration 迭代手册（20260517-3）

## 本次变更

本次迭代收口了 `response_schema` 在 integration 定时任务执行链路中的 metadata 透传。

当任务明细存在 `response_schema` 时，integration 在转发 `/v1/chat/completions` 请求时，会把这份 Json String 追加到请求体的 `metadata.response_schema`。

同时，integration 在任务完成后通过 `META_ID` / `meta_ref` 找到原始 `add-request` 并自动调用对应插件 `send` 回复结果前，也会对响应报文做 JSON 标准化处理。

## 当前行为

1. integration 定时器开始执行 `task_detail` 时，会读取该明细的 `response_schema`
2. 如果该字段为空，则保持原有行为不变
3. 如果该字段非空，则在请求体 `metadata` 中附加 `response_schema`
4. 同时继续保留现有 `response_format.type=json_schema` 的透传逻辑
5. 因此同一份 schema 会同时出现在：
   - `metadata.response_schema`
   - `response_format.json_schema.schema`
6. 自动回推插件 `send` 时，如果 `task_detail.result_content` 是 ```json ... ``` 或 ``` ... ``` 包裹的 Json Object / Array，则会先去掉 Markdown 外壳并标准化为紧凑 JSON
7. 如果标准化失败，或解析结果不是 Json Object / Array，则继续使用原始响应文本回推

## 请求体效果

转发到上游 `/v1/chat/completions` 的请求体会包含：

```json
{
  "metadata": {
    "response_schema": "{\"type\":\"object\",\"properties\":{\"summary\":{\"type\":\"string\"}}}"
  },
  "response_format": {
    "type": "json_schema",
    "json_schema": {
      "name": "scheduled_task_response",
      "schema": {
        "type": "object"
      }
    }
  }
}
```

## 适用范围

- `integration` 内嵌的 proxy / cron 执行链路
- 周期任务明细执行
- 由 `add-request` 桥接生成的一次性任务明细执行
- 已完成非 `cron` 任务明细的自动回推插件 `send`

## 兼容性说明

- 普通外部调用 `/v1/chat/completions` 的 metadata 注入规则不变
- 只有任务明细本身存在 `response_schema` 时才会新增该字段
- `response_schema` 保持原始 Json String，不在 metadata 中重组为对象
- 现有 structured output 调用方不受影响
- JSON 标准化仅发生在自动回推插件 `send` 前，不会修改数据库里保存的原始 `result_content`

## 说明

- 本次迭代手册对应当前目录下的 `REQUIREMENT.md`
- 当前目录交付文件为：`REQUIREMENT.md`、`USER_GUIDE.md`
