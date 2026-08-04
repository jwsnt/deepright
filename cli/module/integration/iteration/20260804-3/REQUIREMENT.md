### 第一性原则
+ 仅可以新增、更新或删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../site/index.html`、`../../../config/config.json` 与应用发布资源。

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
+ 不新增外部 Go 依赖；保留既有 `/api/runtime_config` 返回字段和现有接口行为。

### 需求介绍
+ 主运行配置 `config/config.json` 的 `page.new_tab` 与 `page.iframe` 分别为正整数 SSE 业务码。Site 需要通过现有 `GET /api/runtime_config` 获得该配置，因此该接口的 `config` 中必须透传完整 `page` 对象。
+ 新增仅限本机管理请求调用的 `POST /api/browser/open`。请求体为 `{"url":"https://..."}`；仅接受带 host 的 `http` 或 `https` URL，并使用当前操作系统默认浏览器打开该 URL。
+ 非 POST、本机以外请求、非法 JSON、缺少或非法 URL 必须返回明确的 HTTP 错误，且不得启动浏览器。成功返回 JSON 状态；启动失败返回服务端错误。
+ Site 接收到 `page.new_tab` 对应的 SSE 包时会调用该接口；接收到 `page.iframe` 对应的 SSE 包时只在页面内展示 URL。两类包的 `url` 和 `message` 均由上游服务保证为必填字符串。
+ 模型配置测试的上游 SSE 命中任一已配置页面业务码且内容符合约定时，Integration 必须将仅含该 `code` 和 `choices[0].delta.content` 的特殊包转给 Site，并继续完成原有 SSE 校验；不得转发普通模型测试 SSE 分包。
+ `cli/get` 返回的 HTTP 状态码或 JSON 业务 `code` 命中 `page.new_tab` 或 `page.iframe` 时，是成功的无任务心跳，不得记录为 heartbeat 失败或推动页面远程服务报警。

### 编写代码
+ 复用现有运行时配置读取、URL 解析、本机管理请求校验及跨平台系统目标打开能力；不得在前端直接恢复被页面禁用的 `window.open`。
+ 为运行时配置透传和浏览器打开接口补充单元测试，覆盖正常打开、非本机请求和非 HTTP(S) URL；覆盖模型测试特殊包的最小转发，以及 `cli/get` 页面业务码以 HTTP 状态码和 JSON 业务 `code` 返回时的 heartbeat 豁免。

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`，说明 `page.new_tab`、`page.iframe`、模型测试特殊包、`cli/get` heartbeat 豁免、`/api/runtime_config` 的透传范围和 `/api/browser/open` 的调用限制。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求、边界和验收行为，不记录实现过程。
