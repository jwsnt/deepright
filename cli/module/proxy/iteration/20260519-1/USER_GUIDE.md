# Proxy Iteration 20260519-1 User Guide

## 变更内容

- 插件日志统一通过 `/api/plugins/log?name=插件key` 读取
- 插件日志文件路径固定为 `release/plugins/插件名.log`
- 不再根据当前工作目录、上级目录或其他候选目录推断日志路径
- 日志文件不存在时，返回明确错误：`log file not found: release/plugins/插件名.log`

## 示例

```text
GET /api/plugins/log?name=feishu&last=10
```

固定读取：

```text
release/plugins/feishu.log
```

```text
GET /api/plugins/log?name=email&last=10
```

固定读取：

```text
release/plugins/email.log
```

## 验收结论

- `feishu` 日志读取固定为 `release/plugins/feishu.log`
- `email` 日志读取固定为 `release/plugins/email.log`
- 其他插件同理，统一按 `release/plugins/插件名.log` 处理
