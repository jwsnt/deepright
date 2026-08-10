# 右侧 Sidebar 分割带与 CLI 子任务面板

## 分割带

右侧 Sidebar 第一排（日历+备忘录）和第二排（CLI 子任务面板）之间增加 30px 高度的分割带，视觉上区分两个区域。

## CLI 子任务面板

- `biz=cli && workflow=sub` 的 SSE 内容重定向到第二排（合并为一整行）
- 气泡字体 50%，两侧 padding 10px
- 内容与 Agent+Chat 绑定，切换会话或刷新后恢复
- 自动滚动到最新内容
