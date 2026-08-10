# Site：特殊 SSE 页面打开

每轮会话发送前，Site 从运行时配置读取 `page.new_tab` 与 `page.iframe`。若 SSE 包的 `code` 与任一值一致，Site 会把 `choices[0].delta.content` 解释为：

```json
{
  "url": "https://example.com",
  "message": "已在浏览器中打开页面"
}
```

assistant 气泡始终只显示 `message`。

- 命中 `page.new_tab` 时，Site 异步请求 Integration 用系统默认浏览器打开 `url`。浏览器打开失败不会影响这条气泡、后续 SSE 分片或会话完成状态。
- 命中 `page.iframe` 时，Site 在页面内打开 `url` 的 iframe 覆盖窗口。它与右侧 URL iframe 的展开态使用相同的尺寸和样式：覆盖可视区域、四周保留 `15px` 边距，并带圆角、阴影和背景模糊；关闭按钮、点击背景或按 `Esc` 可关闭窗口，且不会改变右侧栏已有预览。
- 设置页的模型配置测试使用同一套业务码和内容格式。命中后，测试结果只显示 JSON 中的 `message`，不会添加“配置错误：”前缀；`new_tab` 会打开系统浏览器，`iframe` 会覆盖设置弹窗。关闭该 iframe 后会回到设置页，测试不会因此取消。

对应需求见 [REQUIREMENT.md](REQUIREMENT.md)。
