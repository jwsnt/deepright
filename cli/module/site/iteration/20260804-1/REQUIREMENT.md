### 第一性原则
+ 仅可以新增、更新或删除 site（../..）同目录及其子目录下的文件和文件夹，以及本需求直接涉及的 `../../../integration`、`../../../config/config.json` 与应用发布资源。

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
+ 使用 Integration 暴露的运行时配置和本机浏览器打开接口；不能破坏普通 SSE、错误处理、URL 预览或现有页面导航限制。

### 需求介绍
+ 发起每轮 SSE 请求及模型配置测试时，从 `GET /api/runtime_config` 返回的 `config.page.new_tab` 与 `config.page.iframe` 读取两类目标业务码。
+ 当一个 SSE 数据包的 `code` 与任一目标业务码相等时，将 `choices[0].delta.content` 按 JSON 字符串解析为 `{"url":"...","message":"..."}`。服务端保证 `url` 和 `message` 均为必填字符串。
+ 该包在原本的 assistant SSE 气泡中只展示 `message`，不展示 JSON 包装文本；原始 SSE 记录仍保留，以维持历史和诊断能力。
+ `page.new_tab` 包不应按非 2xx SSE 错误处理；同时异步调用 `POST /api/browser/open` 打开 `url`，不得让打开失败影响当前 SSE 的展示、后续分片或完成状态。
+ `page.iframe` 包在页面内打开 `url` 的 iframe 覆盖窗口。窗口尺寸和视觉样式必须与右侧 URL iframe 点击“展开”后相同，且不改变右侧栏已有预览状态；关闭按钮、点击背景和 `Esc` 均可关闭该窗口。
+ 模型配置测试收到同类特殊 SSE 包时，测试结果区也只展示 JSON 中的 `message`，不添加“配置错误：”前缀；`new_tab` 与 `iframe` 的打开行为和普通会话一致。`iframe` 必须覆盖设置弹窗，关闭后恢复原设置弹窗且不取消本次测试。

### 编写代码
+ 新增逻辑必须复用现有运行时配置缓存、SSE 包解析、assistant 气泡增量渲染、模型测试 SSE 结果处理、URL 规范化和既有右侧 iframe 展开态视觉样式。
+ 不得直接调用被页面统一禁用的 `window.open`，也不得改变当前页面的导航拦截策略。
+ 配置不可用或目标包内容不符合约定时，保持现有 SSE 行为，不将错误状态扩散到其他包。

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`，说明两类特殊 SSE 包的内容格式、普通会话与模型测试的展示效果、系统浏览器打开行为和页内 iframe 窗口行为。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求、边界和验收行为，不记录实现过程。
