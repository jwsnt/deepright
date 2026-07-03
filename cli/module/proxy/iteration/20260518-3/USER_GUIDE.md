# Proxy 迭代手册（20260518-3）

## 本次变更

本次迭代补齐了 `proxy` 在自动调用插件 `send` 回复任务结果前的 JSON 标准化处理。

当 `proxy` 通过 `META_ID` / `meta_ref` 找到原始 `add-request`，并准备执行对应插件的 `send` 命令时，会先检查 `task_detail.result_content` 是否为 Markdown 代码块包裹的 JSON。

## 当前行为

1. `proxy` 自动回推三方消息时，仍然使用原始消息 JSON 作为 `--message`
2. 回推文本仍来源于 `task_detail.result_content`
3. 但在真正执行插件 `send` 前，会先对 `content` 做一次标准化
4. 如果内容满足以下格式之一：
   - 以 ````json` 开头并以 ``` 结尾
   - 以 ``` 开头并以 ``` 结尾
5. 则会先去掉 Markdown 外壳，再尝试解析 JSON
6. 只有当解析结果是 Json Object（`{}`）或 Json Array（`[]`）时，才会压缩标准化为紧凑 JSON
7. 如果不是对象/数组，或解析失败，则保持原始文本不变

## 示例

输入：

```json
{
  "hello": "world"
}
```

发送给插件的 `--content`：

```json
{"hello":"world"}
```

输入：

```
[
  {
    "today": "sunday"
  }
]
```

发送给插件的 `--content`：

```json
[{"today":"sunday"}]
```

如果输入是普通文本、非法 JSON，或 JSON 解析后不是对象/数组，则继续使用原始内容。

## 适用范围

- `proxy` 自动回推插件 `send` 的链路
- `task_detail.result_content` 到插件 `--content` 的转换
- 非 `cron` 类型已完成任务明细的自动回复

## 兼容性说明

- 本次仅影响自动回推给插件时的 `content`
- 不修改数据库中保存的 `task_detail.result_content`
- 不影响普通 `/v1/chat/completions` 请求转发
- 标准化失败时回退到原始文本，不会破坏现有普通文本回复

## 说明

- 本次迭代手册对应当前目录下的 `REQUIREMENT.md`
- 当前目录交付文件为：`REQUIREMENT.md`、`USER_GUIDE.md`
