# SSE 响应打字机效果

## 功能说明

居中会话框展示 SSE 响应时，助手消息末尾显示闪烁光标（▌），模拟打字机效果。SSE 流结束后光标消失。

## 技术实现

- CSS：`.message.assistant.streaming .md-content::after` 添加闪烁光标伪元素
- SSE 流式渲染时给 msgEl 添加 `streaming` class
- `finishStreamingFor` 完成时移除 `streaming` class
