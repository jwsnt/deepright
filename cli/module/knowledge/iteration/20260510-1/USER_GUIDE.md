# Knowledge 迭代手册

## 本次迭代内容

本次迭代为 `knowledge` 模块补充 `update-time` CLI 命令，用于更新共享 sqlite 中的 Knowledge 最后更新时间。

- 命令名为 `update-time`
- 支持位置参数时间戳写法：`knowledge update-time 时间戳`
- 同时保留 `set-update` 作为兼容别名
- 继续遵守 `integration` 的二进制与 CLI 收口原则，不在 `proxy`/`integration` 内重复实现 Knowledge 数据逻辑

## 命令用法

```bash
knowledge update-time --app-dir /path/to/app --timestamp 1715337600000
knowledge update-time 1715337600000
knowledge set-update --app-dir /path/to/app --timestamp 1715337600000
```

## 行为说明

- `update-time` 会打开应用启动目录下共享的 `data` sqlite
- 若底层表结构不存在，会自动初始化 `knowledge_runtime`
- 成功写入后，输出当前 Knowledge 状态：
  - `path`
  - `lastUpdate`
- `lastUpdate` 的单位统一为 Unix 毫秒时间戳

## 设计约束

- Knowledge 目录固定为 `<app-dir>/knowledge`
- 共享 sqlite 固定为 `<app-dir>/data`
- 最后更新时间保存在 `knowledge_runtime.last_update`
- 不破坏已有 `Metadata`、`MetadataIfExists`、`MergeMetadata` 等能力

## 示例输出

```json
{
  "path": "/app/knowledge",
  "lastUpdate": 1715337600000
}
```
