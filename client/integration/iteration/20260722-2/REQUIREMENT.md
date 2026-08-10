### 第一性原则
+ 仅可以新增/更新/删除integration（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Integration介绍：../../REQUIREMENT.md
+ Integration手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 新增仅供本地设置页调用的模型配置测试接口 `POST /api/model/test`。接口使用请求中当前行、尚未保存的服务商、密钥和模型客户化配置发起一次测试；不得写入 token_store、Agent `config.json`、聊天历史、会话恢复记录、记忆、技能、任务或通知。
+ 请求体只允许包含服务商名及该行有效配置：`model`、`token`、`__url`、`__model`、`__model_fast`、`__model_thinking`、`__model_multi_input`、`__model_multi_output`。服务端必须拒绝未知字段、空服务商、空密钥，以及空 `__model`；测试始终使用 `__model`，不得回退到 fast、thinking 或多模态模型。
+ `__url` 可为空以使用服务商默认地址；非空时必须是无用户名和密码、带主机名的绝对 `http://` 或 `https://` URL，且只接受标准 ASCII URL 字符；反斜杠、空白、中文标点（如 `、`）和无效百分号编码均必须拒绝。非法 URL 必须在请求转发前拒绝，并返回“配置错误：模型 URL 格式不正确，请填写 http:// 或 https:// 开头的完整 URL”。
+ 接口必须从运行时 `config/config.json` 的 `test.content` 和 `test.timeout` 读取测试内容与超时秒数。`test`、`content` 或 `timeout` 缺失，内容为空，timeout 非正整数时，接口返回可展示的配置错误；不得使用前端传入的测试文本或超时兜底值。
+ 服务端为每次测试生成 UUID 会话 ID，并以最小 SSE 请求转发到现有上游 `/v1/chat/completions`。转发请求必须标识 `metadata.test = true`、`metadata.chat = UUID`、`metadata.type = test`，携带当前行 Token 和模型配置；`metadata.test` 必须原样透传至 `--host` 指定的最终处理服务器，供其启用测试隔离逻辑。不得合并 Agent、workspace、skills、memory、knowledge、router、插件、历史、任务或媒体元数据，也不得将该 UUID 加入连接表或日志表。
+ `metadata.test` 是测试专用保留字段：除 `/api/model/test` 构造的测试转发请求外，普通页面聊天、Cron、CLI 与其他 `/v1/chat/completions` 请求均不得携带该字段；普通转发收到客户端伪造的 `metadata.test` 时必须删除，不能进入测试执行分支。
+ 测试请求需保留当前服务商客户化配置，使上游可按与实际使用一致的 URL、`__model` 和 Token 调用服务商。HTTP Authorization 与内部 Token 元数据只在本次内存请求中使用，任何错误、日志与响应均不得回显密钥。
+ 测试接口须在服务端消费上游 SSE，并仅向浏览器返回不含模型正文的测试结果 SSE；用 `context.WithTimeout` 约束连接、首包与整个流的总耗时。客户端断开时立即取消上游请求。测试取消不写入任何状态，也不影响普通会话。
+ 成功仅在同时满足以下条件时成立：HTTP 状态为 200、收到至少一个有效业务 SSE `data` 事件、未出现 SSE error/错误 JSON、且收到 `[DONE]` 后正常结束。HTTP 非 2xx、首包或总超时、流中断、空流、SSE 内错误或缺少 `[DONE]` 必须以 SSE 错误事件返回，并包含服务商错误的脱敏、截断说明。
+ 所有测试错误内容需移除/遮蔽 Token、Bearer、Authorization、API Key 等敏感值，并限制可展示的错误正文长度；非 JSON 错误响应、网络错误、异常状态码和 SSE 错误事件均需转换为稳定、可展示的 `配置错误：…` 信息。服务商错误 JSON 存在 `content` 时（包括 `choices[].delta.content`）必须优先提取该字段，不得回显完整原始响应或其传输元数据。HTTP 状态、错误 JSON 的非 2xx `code/status`，以及 `content` 内显式标记的 `code/status` 必须统一展示标准文案：401 身份认证失败、403 拒绝访问、404“请检查模型 URL 和基础模型”、429 请求受限、503 暂不可用、其余 5xx 服务异常；命中上述标准文案时不得附加服务商 `content` 或分隔符。
+ 测试接口只向页面返回已脱敏、截断后的最终测试 SSE 结果，不返回模型正文；页面必须将成功、失败或取消结果展示在对应设置模型行内，不得依赖会被设置浮层遮挡的虚拟文件系统 Toast。

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
    + 在 `../../main.go` 新增受限测试处理器并注册到 Integration HTTP mux；不得复用会写入聊天日志、连接表、Agent 元数据、记忆、技能或通知的普通 `/v1/chat/completions` 处理路径。
    + 复用既有 HTTP 客户端、SSE 分帧和异常响应识别能力，但为测试请求增加严格的首包/总超时、有效数据、`[DONE]` 与错误事件判定。测试流的完成状态必须由服务端判断，不能仅依据浏览器读到了 HTTP 200。
    + 新增配置读取、请求校验、UUID 生成、敏感信息脱敏和 SSE 测试结果封装的最小辅助函数；不得持久化请求中的 Token 或客户化配置，不得接受客户端覆盖 `config.json.test`。
    + 上游核心收到 `metadata.test = true` 时，必须跳过 Agent、技能、记忆、知识库、路由、任务和工具相关处理，也不得提交记忆或持久化会话；只保留发起目标服务商调用所需的最小模型配置，并确保该字段到 `--host` 最终服务端不丢失。
    + 覆盖自动化测试：配置缺失/非法、请求字段与 `__model` 校验、UUID 与测试标记及至 `--host` 的透传、普通链路剥离伪造 test 标记、未保存配置透传、普通日志与连接状态未写入、HTTP 非 200、首包超时、流中断、空流、流内错误、缺失 `[DONE]`、成功 SSE、客户端取消以及错误脱敏/截断。
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写
+ 最小范围更新

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
