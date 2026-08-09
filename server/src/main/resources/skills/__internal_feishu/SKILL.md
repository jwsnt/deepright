---
name: __internal_feishu
description: 处理飞书（Feishu/Lark）相关功能，包括附件、文档、日历、任务及 Open API/MCP 操作。当飞书专用能力与浏览器均可完成任务时，优先使用本技能，仅在本技能及其集成能力无法完成时才使用浏览器（Browser）
---

### 安全与范围
+ `appSecret`、访问令牌、完整Connect metadata、原始飞书事件、会话标识、`openid`和可能携带凭据的`mcp_url`均属于敏感数据，不得在回复、日志摘要、代码块、文件或后续命令参数中回显
+ 不要为了读取、回复或发送飞书消息而默认执行`meta-get`，应优先让飞书插件自行读取配置
+ 不向用户索取、转发、保存或展示`appSecret`，如需配置凭证，应引导用户通过受控配置渠道完成
+ 不得仅凭姓名、称呼、消息内容或`openid`推断收件人或主动发送消息
+ 当前`feishu send --message`要求完整请求JSON，且实现会记录该参数，在具备安全的`--request-id`回复能力前，不得自动执行该命令

### 获取凭证状态
+ 仅当用户明确要求诊断飞书配置，或插件已返回配置/鉴权失败时，才检查配置状态
+ `#app connect meta-get --key "feishu"`会返回完整meta，除非调用端能够保证字段级脱敏，否则不得执行
```
"meta": {
    "appId": "$APP_ID",
    "appSecret": "$APP_SECRET"
}
```
+ 应优先使用仅返回状态的受控接口，最终仅报告以下之一：
  + `飞书配置可用`
  + `飞书配置缺少appId`
  + `飞书配置缺少appSecret`
  + `飞书配置缺少appId和appSecret`
  + `无法安全验证飞书配置状态`
+ 不得复制、记录、转发或再次引用原始工具输出

### 获取已有消息与回复
+ 回复前必须先定位唯一请求，查询候选时使用待处理状态和系统最小分页大小`--limit "20"`，除非任务需要，不得继续分页
```
#app connect request-list --key "feishu" --status "0" --limit "20"
```
+ 默认返回最新记录。只查询待处理消息时追加`--status "0"`；继续向更早的消息分页时，使用上次结果中最小的`id`作为`--before-id`：
```
#app connect request-list --key "feishu" --before-id "<id>" --limit "20"
```
+ `request-list`不支持按Agent或Chat过滤，无法由用户提供的请求ID或当前任务上下文唯一确认目标时，只展示必要的脱敏候选信息并请求用户选择
+ `request-list`仅用于定位，不得在最终反馈中回显原始飞书事件、完整请求JSON、消息ID、消息正文或其他人的`openid`
+ 当前插件回复接口要求把完整请求JSON传给`--message`，不符合本技能的敏感数据边界，在实现受控的`feishu send --request-id <id>`接口前，报告"当前无法在不暴露原始消息的前提下安全回复"
+ 实现`--request-id`后，仅传递唯一请求ID，由受控服务端读取原始请求，并在日志中只记录脱敏ID和发送结果

### 查找消息发送者
+ `openid`和`search`只查询本地飞书消息快照中的发送者候选，不调用飞书Open API，也不得读取飞书日志或临时状态文件
+ 查询结果只能用于定位来源或辅助用户选择，不得将搜索结果中的`openid`、消息ID或消息正文作为主动发送的依据
+ 查询结果只能用于识别近期消息发送者，不能自动视为最终收件人，也不能仅凭`openid`主动发送飞书消息
+ 仅在用户要求查找近期消息发送者、需要确认消息来源，或需要辅助定位待回复消息时查询；不得根据姓名、称呼或消息内容臆造`openid`
+ 优先使用用户提供的非空关键词查找候选；用户明确要求查看近期全部文本消息时才可省略`--query`。无关键词查询仍必须使用最小`--limit`和必要的`--offset`分页，只读取完成任务所需的最小结果，避免在回复中展示无关消息内容、消息ID或其他人的`openid`
+ 如用户只需最近联系过的候选，或没有可搜索的关键词，可查询唯一发送者列表。查询结果仅用于确认发送者；如需回复，仍必须按“获取已有消息”定位唯一请求记录
+ 首次使用、参数不明确或命令失败时，先查看帮助：
```
#plugins_dir/feishu help
```
+ 获取最近窗口内唯一的发送者Open ID列表：
```
#plugins_dir/feishu openid
```
+ 按用户提供的关键词搜索最近窗口内的文本消息（查看帮助获取AND、--limit、--offset使用方法）：
```
#plugins_dir/feishu search --query "用户提供的关键词"
```
+ 按用户的openid搜索最近窗口内的文本消息：
```
#plugins_dir/feishu search --openid ou_xxx
#plugins_dir/feishu search --query "用户提供的关键词" --openid ou_xxx
```

### 使用飞书MCP
+ 当用户请求飞书相关功能且现有飞书插件能力无法完成时，可尝试使用飞书MCP，用户无需明确提及“MCP”
+ 先安全获取已配置的`mcp_url`，再通过技能`__internal_mcp`执行`tools/list`发现当前可用能力
+ 仅在发现到能满足用户意图的工具后继续，有外部副作用的调用仍须在执行前确认目标、内容和影响范围
``` 获取飞书MCP URL
#app connect meta-get --key "feishu"
```
``` 仅在返回的meta包含`mcp_url`时使用
"meta": {
    "mcp_url": "飞书MCP URL"
}
```
+ 上述`meta-get`的调用端必须对`meta.mcp_url`以外的字段脱敏，`meta.mcp_url`仅可用于技能`__internal_mcp`的连接目标，不得回显、记录或写入文件
+ `mcp_url`必须为用户授权的HTTPS地址，且不得包含用户名密码、访问令牌、查询参数凭据或fragment凭据
+ 使用技能`__internal_mcp`执行Streamable HTTP的`tools/list`，每次连接前重新发现工具`tools/list`
+ 默认只允许`tools/list`，调用会发送消息、写入文档、修改权限、创建对象或产生其他外部副作用的工具前，必须让用户确认目标、内容和影响范围
+ 禁止使用`resources/templates/list`、`resources/list`或`prompts/list`协议

### 操作与失败反馈
+ 服务操作成功时，仅说明目标服务与最终状态，不泄露敏感配置、原始事件或完整工具输出
+ 配置缺失时，提示"请在受控配置中补充飞书凭证后重试"
+ 鉴权失败时，提示"请检查飞书应用权限或重新授权后重试"
+ 无法安全读取配置或MCP地址时，提示"当前无法安全验证配置，请通过受控配置渠道处理"
+ 不得在失败信息中附带原始工具输出、HTTP请求头、令牌、会话标识或完整meta