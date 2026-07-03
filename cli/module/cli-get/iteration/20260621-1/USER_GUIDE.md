# 迭代说明

本次迭代为 `cli-get` 增加了插入消息随 `cli/pub` 自动上报的能力。

## 行为变更

- 每次准备提交 `cli/pub` 前，都会先读取共享 SQLite `data` 中 `message_insert` 表里当前 `chat` 的待上传消息
- 单次最多读取 `5` 条，按 `created_at` 顺序写入：

```json
{
  "insert": [
    { "mid": "1718966400000", "message": "..." }
  ]
}
```

- `/cli/pub` 成功后，这批 `mid` 会自动更新为 `status=1`
- 读取或回写失败不会中断原有 `cli/pub`，只会输出错误日志

## 状态存储

- 插入消息与原有日志、沙盒状态共用同一个 SQLite `data`
- `message_insert.status` 固定为：
  - `0`：待上传
  - `1`：已上传
  - `2`：取消

## 已验证

已执行：

```bash
cd /path/to/deepright/cli/module/cli-get
go test ./...
```
