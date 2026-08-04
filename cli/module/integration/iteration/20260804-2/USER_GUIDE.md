# 上游请求中的主配置路径

Integration 会验证启动时实际使用的主 `config/config.json`，并在内置 `/cli/get` 心跳、普通会话、备忘录任务（含飞书和邮件 Connect 消息）以及设置页模型测试的上游 `/v1/chat/completions` 请求中写入顶层 `metadata.config`。该值是配置文件的绝对路径。

macOS `.app` 使用 `<App>.app/Contents/Resources/config/config.json`；直接运行的 macOS 二进制、Linux 与 WSL 使用 `<integration 可执行文件所在目录>/config/config.json`，典型 WSL 安装路径为 `~/deepright/config/config.json`。

`metadata.config` 由 Integration 在转发前生成并覆盖调用方提供的同名值，不使用 Agent 工作目录中的 `config.json`。`/cli/pub` 与独立 `cli-get` 不新增此字段。

主配置必须存在、为可读普通文件且 JSON 合法，否则 Integration 启动失败。服务运行后配置被删除、替换为目录、变得不可读或 JSON 非法时，受影响的上游请求会在本机失败，不会发送不可靠的路径。
