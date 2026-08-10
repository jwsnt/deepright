# 20260511-2 User Guide

## 目标

本次迭代为 `proxy` 新增了知识库更新时间查询接口 `/knowledge_lastUpdate`。

- 接口读取共享 knowledge runtime 中保存的 `last_update`
- 返回格式固定为 `yyyy-MM-dd HH:mm`
- 可由 `site` 或其他同源模块直接调用

## 使用方式

启动 `proxy` 后，请求：

```bash
curl http://127.0.0.1:9876/knowledge_lastUpdate
```

返回示例：

```text
2026-05-11 09:45
```

## 说明

- 仅支持 `GET`
- 时间格式使用当前进程本地时区
- 当 knowledge 共享 sqlite 尚未写入更新时间时，会返回初始时间戳 `0` 对应的格式化结果
- 更完整说明请继续参考上级手册 `../../USER_GUIDE.md`
