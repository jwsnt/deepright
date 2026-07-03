# 20260524-4 User Guide

本次迭代把 Integration 的插件配置与桥接任务统一切到 `router_disable`。

- 插件配置保存时写入 `router_disable`
- 插件元数据返回时暴露 `router_disable`
- Connect 请求桥接成即时 cron 任务时，会把 `router_disable` 透传到任务元数据和任务明细

推荐示例：

```http
POST /api/plugins/config?key=feishu&agentId=A&chatId=chat-001&model=OpenAI&thinking=true&router_disable=false
```

