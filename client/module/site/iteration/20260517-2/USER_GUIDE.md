# 20260517-2 使用说明

## 迭代目标

本次迭代为居中对话框补充 LaTeX 公式渲染能力，并要求继续兼容现有 Markdown 渲染。

## 变更说明

- 居中会话区的用户消息与 assistant 消息现在支持 LaTeX 公式渲染。
- 支持的公式写法包括：
  - 行内公式：`$E = mc^2$`
  - 行间公式：`$$\int_0^1 x^2 dx$$`
  - 行内公式：`\(...\)`
  - 行间公式：`\[...\]`
- 代码块、行内代码、`pre/code` 区域不会参与公式替换，避免把普通代码中的 `$` 误判为公式。

## 资源要求

- 所有前端依赖必须使用本地资源，不能依赖 CDN。
- 页面引用到的本地 JS/CSS 必须在静态站点目录中真实可访问，并返回正确的 MIME。
- 禁止出现页面已引用某个本地资源，但静态服务未实际发布该文件，最终返回 `404 text/plain` 的情况。

## 已知根因示例

若浏览器控制台出现以下报错，通常表示本地依赖未被正确发布，而不是公式语法本身有误：

- `Refused to execute script ... because its MIME type ('text/plain') is not executable`
- `Refused to apply style ... because its MIME type ('text/plain') is not a supported stylesheet MIME type`
- `site/vendor/*.js 404 (Not Found)`

## 验收建议

可在居中对话框中直接输入以下内容验证：

```text
$E = mc^2$
```

```text
$$\int_0^1 x^2 dx$$
```

若页面能正常显示公式，且控制台中不再出现 `vendor` 相关 `404` 或 MIME 报错，则本次迭代目标达成。
