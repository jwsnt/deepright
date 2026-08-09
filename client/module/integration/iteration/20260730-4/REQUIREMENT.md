### 第一性原则
+ 仅可以新增/更新/删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../build.sh`、`../../../build/install.ps1`、`../../../build/USER_GUIDE.md`、`../../../build/USER_GUIDE.txt` 与 `../../../config/app/API.md`、`../../../config/app/CANVAS.md`、`../../../config/app/DESIGN.md`。

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
+ 不新增外部依赖，不破坏既有 API 的字段兼容性。
+ `http.timeout.read` 仅约束定时备忘录请求的 SSE 正文读取空闲时长；不得改为全局 HTTP 总时长，也不得改变 `/restore` 接口、参数、返回结构或前端恢复协议。

### 需求介绍
+ 主应用静态 `config/config.json` 的 `http.timeout.read` 为定时备忘录 SSE 读取空闲超时，单位为秒。每次收到任意 SSE 字节都重新开始计时；连续无数据达到该值时中断读取。该值缺失、不是正整数、为零或超过 `time.Duration` 可表示范围时，必须记录配置错误并使用 `120` 秒默认值。
+ 定时备忘录在请求创建失败、转发请求失败、读取空闲超时、读取错误、非 2xx、响应业务异常或响应结束但未出现 `data: [DONE]` 时，均为失败终态：保留已收到的原始 SSE 片段，额外写入现有 `abnormal` 日志“连接已中断，请重试”，并把 `task_detail.started` 更新为 `4`。`4` 表示失败，不再参与 `started = 0` 的定时扫描和自动重试；现有 `2`（无需启动）和 `3`（已完成）语义不变。
+ 失败日志必须继续由现有聊天日志与前端恢复逻辑消费，不新增或修改 `/restore` HTTP 接口。备忘录列表、详情和查询结果应将 `started = 4` 显示为“失败”，并将其视为终态而不是待处理项。

### 编写代码
+ 最小范围更新，不新增外部依赖。
+ 仅调整 Integration 的定时备忘录执行链路、其状态展示和覆盖上述情形的测试；不得把读取空闲超时套用到普通对话、`/restore`、心跳或其它 HTTP 调用。

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`，说明 `miniapp` 的配置格式、`miniapp.recover` 的单位和生效方式、三份受保护文档的范围与单文件恢复语义、`/api/runtime_config` 的受控透传语义、静态配置路径、服务地址 SQLite 存储、优先级、重置行为、WSL/目录发布兼容性以及 `.app` 内配置不可修改的原因。
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`，说明 `http.timeout.read` 的秒单位、`120` 秒缺省回退与日志、SSE 空闲超时语义、失败终态和不自动重试语义，以及 `/restore` 接口保持不变的边界。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求，不记录实现过程。
