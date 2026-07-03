# 20260527-3 User Guide

本次迭代补充了 `SOUL.md / USER.md` 文档整理请求的 metadata。

- 当用户通过 `整理 SOUL.md` 或 `整理 USER.md` 的确认浮层发起请求时，前端会在请求体 `body.metadata` 中额外写入 `profile_commit: true`
- 这条规则只针对 `SOUL.md / USER.md` 文档整理浮层触发的请求
- `知识库WIKI` 整理仍然保持原有行为，继续写入 `knowledge_commit: true`
- 为避免后续重试或链路转换时丢失，统一发送函数也会对 `requestSource = "soul_tidy"` 的请求再次兜底补齐 `profile_commit: true`
