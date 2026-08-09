# Site 迭代手册（20260517-3）

## 本次更新

- 居中对话框新增安全 HTML 片段渲染能力。
- HTML 片段继续兼容现有 Markdown 渲染与 LaTeX 公式渲染，不会影响原有消息展示链路。
- HTML 片段会复用现有会话气泡样式，常见的标题、列表、表格、引用、媒体和折叠块都能直接展示。

## 支持范围

- 文本结构：`div`、`section`、`article`、`p`、`span`、`br`、`hr`
- 标题：`h1` 到 `h6`
- 强调：`strong`、`em`、`u`、`del`、`sub`、`sup`
- 列表：`ul`、`ol`、`li`
- 引用与折叠：`blockquote`、`details`、`summary`
- 代码：`pre`、`code`
- 表格：`table`、`thead`、`tbody`、`tr`、`th`、`td`
- 媒体：`img`、`iframe`、`video`、`audio`、`source`
- 语义容器：`figure`、`figcaption`
- 链接：`a`

## 允许属性

- 全局属性：`class`、`title`、`aria-label`、`aria-hidden`、`role`
- 链接：`href`、`target`、`rel`
- `iframe`：`src`、`loading`、`allow`、`allowfullscreen`、`referrerpolicy`
- 图片：`src`、`alt`、`loading`、`width`、`height`
- 视频：`src`、`controls`、`preload`、`poster`、`muted`、`playsinline`、`loop`、`autoplay`
- 音频：`src`、`controls`、`preload`
- `source`：`src`、`type`
- 表格单元格：`colspan`、`rowspan`，其中 `th` 额外支持 `scope`

## 安全限制

- 以下内容会被自动过滤：`script`、`style`、`srcdoc`、所有行内事件属性（如 `onclick`）
- 以下危险协议会被移除：`javascript:`、`vbscript:`、非图片型 `data:`
- `iframe` 的 `src` 仅接受 `http(s)` 或 `/` 开头的同源路径

## 使用示例

```html
<div>
  <h3>嵌入卡片</h3>
  <p>这是一段 <strong>HTML</strong> 内容，同时仍可和 Markdown、LaTeX 混排。</p>
  <details>
    <summary>展开更多</summary>
    <table>
      <thead>
        <tr><th>类型</th><th>状态</th></tr>
      </thead>
      <tbody>
        <tr><td>iframe</td><td>允许</td></tr>
      </tbody>
    </table>
  </details>
</div>
```

```text
支持和 Markdown、LaTeX 混排，例如：

## 标题

<div>HTML 块</div>

$$E = mc^2$$
```

## 说明

- 本次迭代手册对应当前目录下的 `REQUIREMENT.md`
- 当前目录交付文件为：`REQUIREMENT.md`、`USER_GUIDE.md`
