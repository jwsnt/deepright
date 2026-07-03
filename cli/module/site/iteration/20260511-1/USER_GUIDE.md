# 20260511-1 User Guide

## 目标

本次迭代将右侧边栏底部的 `知识库WIKI` 面板从站点内置的 `wiki_md` 静态测试目录切换为通过相对路径 `/knowledge` 读取真实知识库内容，并通过 `/knowledge_lastUpdate` 读取知识库更新时间。

- 保持原有右侧标题栏、刷新按钮和最近刷新时间样式
- 使用同源 `/knowledge` 接口，避免硬编码 IP 或域名
- 访问目录时展示可点击的知识树
- 访问文件时按类型展示内容：Markdown、普通文本、图片、HTML 页面或原始文件入口
- 标题下方新增可分层点击的导航栏
- 刷新按钮会回到知识库首页并重新加载 `/knowledge`
- `最近刷新` 时间来自同源 `/knowledge_lastUpdate`

## 交互说明

1. 打开页面后，查看右侧边栏最下方的 `知识库WIKI` 面板。
2. 面板首次加载会请求 `/knowledge`，默认展示知识库根目录树。
3. 点击目录或文件名称后，会在当前面板内继续打开对应路径：
   - Markdown 文件按 Markdown 渲染
   - 文本类文件按代码块展示
   - 图片直接预览
   - HTML 文件以内嵌页面展示
   - 其他不支持内联的文件提供“打开原始文件”入口
4. 标题下方会显示当前知识库路径导航，例如 `/knowledge/l2/entity-artifacts.md`：
   - 点击 `/knowledge/l2` 会加载 `/knowledge/l2`
   - 点击 `/knowledge` 会回到知识库根目录
5. 点击右上角刷新按钮后，会回到首页并重新请求 `/knowledge`，同时重新读取 `/knowledge_lastUpdate`。

## 说明

- 当前迭代仅更新 `site` 前端展示层，不改动原有浮层分域和其他右侧 Sidebar 面板行为
- `/knowledge` 的目录树格式由 `proxy` / `integration` 统一提供，前端按树形文本解析为可点击导航
- `最近刷新` 使用 `/knowledge_lastUpdate` 返回的 `yyyy-MM-dd HH:mm` 文本，不再使用前端本地 Mock 时间
- 树形结构和文件内容都会尽量占满整栏宽度；当内容超出当前展示区域时，统一使用面板自身滚动条
- 更完整说明请继续参考上级手册 `../../USER_GUIDE.md`
