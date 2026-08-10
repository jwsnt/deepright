### 第一性原则
+ 仅可以新增、更新或删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../site/index.html`、`../../../config/config.json` 与应用发布资源。

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
+ 不新增外部 Go 依赖，不改变既有 `/api/runtime_config`、普通消息发送、附件上传或 Agent 工作区访问协议。

### 需求介绍

为迷你应用调试入口提供受控运行时模板配置。Integration 只负责从当前发布包读取并向 Site 透传允许的 `miniapp` 配置；模板替换、普通消息提交和附件组合由 Site 完成。Integration 不执行修复命令、不创建调试任务，也不修改 Agent 的 HTML 文件。

- 主应用实际使用的 `config/config.json.miniapp` 必须支持 `debug` 字段。发布默认模板为 `请修复 $path，问题是 $reason`；它使用 `$path` 表示迷你应用 HTML 的绝对路径，使用 `$reason` 表示用户填写的修复原因。
- `GET /api/runtime_config` 必须继续只读地从当前发布包（macOS 为 `DeepRight.app/Contents/Resources/config/config.json`）读取受控配置，并在已允许的完整 `miniapp` 对象中原样透传 `debug`。不得读取 Agent 工作目录中的同名配置、不得写入配置、不得执行模板或修复命令。
- 配置接口不得扩大其它配置字段的暴露范围；`provider`、模型密钥及未列入既有白名单的数据仍不得返回给浏览器。`miniapp.debug` 缺失、不是字符串、为空或缺少 `$path`、`$reason` 时，由 Site 在用户确认时显示配置错误并阻止发送；Integration 不提供隐式模板回退。
- 调试请求仍使用既有普通会话消息与附件协议。Integration 不新增 `/api/miniapp/debug` 或类似端点，不解析 `$path`、`$reason`，不根据浏览器值访问、读取、写入或执行该路径。
- 发布时必须把包含 `miniapp.debug` 的 `config/config.json` 与对应 Site 静态页面同步到实际应用资源目录；应用资源更改后必须使用发布证书重新签名并通过严格验证。

### 编写代码
+ 复用既有运行时配置位置解析、`miniapp` 白名单透传与 macOS/WSL 资源路径逻辑；不得引入第二份运行时配置、缓存写回或 Agent 级覆盖逻辑。
+ 为 `miniapp.debug` 的运行时透传、发布包配置读取和非公开字段继续不可见提供自动化回归验证。
+ 保持实现最小化，避免为纯文本模板增加数据库、任务队列、子进程、文件系统写入或新的 HTTP 路由。

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求、边界和验收行为，不记录实现过程。
