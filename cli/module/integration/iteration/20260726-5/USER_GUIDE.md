# Integration 迭代手册（20260726-5）

## 画布文件持久化

`.canvas` 是保存于 Agent 工作区内的 UTF-8 JSON 文件。Integration 通过既有受限写入接口保存它；浏览器生成的 PNG/PDF 也经同一接口存入工作区，不增加画布专用服务或任意本机路径读写能力。

- Site 仅在用户点击“保存”时调用 `POST /api/edit?agentId=<agentId>&path=<relativePath>`。无后缀名称会先自动补成 `.canvas`，已带 `.canvas` 的名称保持不变，其它后缀不会发起请求。缺失的相对 `.canvas` 路径会在此时创建；用户在保存前取消或关闭时不会产生文件。
- 再次保存会更新同一文件。覆盖确认在 Site 完成，Integration 继续校验 Agent、相对路径、工作区边界和符号链接边界，并在失败时返回可展示的错误。
- 服务端将画布 JSON 原样保存，不解释节点（包含可选背景色）、图片、连线（包含 `forward`、`reverse`、`both` 方向与备注）或标题。PNG/PDF 仍由浏览器生成，但会以 Base64 通过受限 `/api/edit` 写入当前 Agent 工作目录的 `canvas/`；目录不存在时自动创建，`saveAsNew=true` 会为每次导出附加高精度时间戳以避免覆盖。响应的 `savedAs` 返回绝对路径，供页面复制到系统剪贴板。
