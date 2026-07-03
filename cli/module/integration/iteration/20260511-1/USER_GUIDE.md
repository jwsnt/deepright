# 20260511-1 User Guide

## 目标

本次迭代将 `proxy` 的知识库相关 HTTP 能力收口到 `integration` 唯一入口，包括：

- `/knowledge`
- `/knowledge_lastUpdate`

## 使用方式

启动 `integration` 后，可直接通过同一服务访问：

```bash
curl http://127.0.0.1:8080/knowledge
curl http://127.0.0.1:8080/knowledge_lastUpdate
```

其中 `/knowledge_lastUpdate` 返回格式为：

```text
2026-05-11 09:45
```

## 说明

- `/knowledge` 继续映射应用启动目录下的 `knowledge` 目录
- `/knowledge_lastUpdate` 继续读取共享 knowledge runtime 中的 `last_update`
- `site` 从 `integration` 提供的同源接口读取知识库内容和更新时间，不再依赖独立 `proxy` 进程
