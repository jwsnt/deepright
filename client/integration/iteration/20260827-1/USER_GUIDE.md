# 按会话注入 OpenAI 历史

Integration 会为具有非空 `metadata.chat`（定时/Connect 任务为 `chatId`）的成功模型轮次保存 OpenAI Chat 格式的 `user` 与 `assistant` 历史消息。每条记录都携带毫秒级 `created`：user 为本地创建上游请求的时间，assistant 为本地收到完整 SSE 结束的时间。

下一次由同一 `chatId` 发起的普通页面会话、备忘录任务或 Connect 即时任务（包括飞书）会自动携带此前完成的历史。上游 `messages` 按旧到新排列：开头的 `system`、`developer` 指令保持最前，历史 user/assistant 居中，本次请求消息位于最后。历史按单条 user/assistant 消息计数，最多发送 `config/config.json` 中 `chat.restore` 指定的最新条数；每条注入消息均含 `created`。

只有收到正常的完整 SSE（含 `data: [DONE]`）且能获得 assistant 文本时才会保存一轮历史。取消、网络失败、异常或不完整 SSE 不会进入后续上下文。历史使用独立存储，不改变页面恢复和导出使用的原始请求/SSE 日志。

`chat.clean` 使用小时单位。Integration 与 Proxy 都从各自 `config/config.json` 读取该值，并在启动后异步清理超过该时间的 `agent_message_log`、`chat_log`、`cmd_log` 和 `chat_history_message`。例如默认 `chat.clean=168` 表示保留最近 168 小时。
