# 20260516-2 User Guide

## 目标

本次迭代在 `integration` 下继续收口知识库相关读取能力。

- 保持 `/knowledge_lastUpdate` 读取知识库最后更新时间
- 新增 `/knowledge_path` 读取知识库真实文件系统绝对路径
- 两者都遵守 `proxy` 已有协议，并由 `integration` 统一对外暴露

## 使用方式

HTTP：

```bash
curl http://127.0.0.1:8080/knowledge_lastUpdate
curl http://127.0.0.1:8080/knowledge_path
```

返回示例：

```text
2026-05-16 14:21
/abs/path/to/knowledge
```

CLI：

```bash
./integration knowledge last-update
```

## 说明

- `/knowledge_lastUpdate` 仍保持原有 `GET` 协议不变
- `/knowledge_path` 为本次新增的 `GET` 接口，返回真实 knowledge 目录绝对路径
- `knowledge last-update` 是 integration 顶层 CLI 收口入口，`knowledge_path` 目前仅提供 HTTP 收口
- `/knowledge_lastUpdate` 与 `knowledge last-update` 统一复用 integration 内部 `knowledge` 子模块读取并格式化共享 runtime 中的 `last_update`
- `/knowledge_path` 与 `proxy` 的路径规则保持一致：固定解析为当前应用启动目录下的 `knowledge`
