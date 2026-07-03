# Integration Iteration 20260519-1 User Guide

## 变更内容

- Integration 按技术规范收口插件日志读取逻辑
- `/api/plugins/log?name=插件key` 的日志路径解析与 Proxy 保持一致
- 所有插件日志固定读取 `release/plugins/插件名.log`

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

## 错误返回

日志不存在时，返回：

```text
log file not found: release/plugins/插件名.log
```
