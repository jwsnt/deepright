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
+ 在居中对话框SSE响应区接收到响应报文中choices[0].metadata.__PROCESS__不为空的报文后，该段报文内容不渲染到居中对话框正文里，而是展示到当前会话最后一个assistant气泡底部footer预留槽位
```
{
    "choices": [
        {
            "metadata": {
                ...
                "__PROCESS__": 任意值,
                "agentId": 当前AgentId
            },
            "delta": {
                "content": ...
            }
        }
    ],
    "code": 200,
    "biz": "main"
}
```
+ 展示内容还是choices[0].delta.content
``` 如以下为\n[正在制定规划]\n
data: {"choices":[{"index":0,"metadata":{"__PROCESS__":"plan.create","agentId":"X"},"delta":{"content":"\n[正在制定规划]\n","role":"assistant"}}],"workflow":"create","biz":"plan",...}
```
+ __PROCESS__展示内容取choices[0].delta.content，并按前端现有收口规则做文本归一化：
    + 去掉\r
    + trim首尾空白
    + 连续换行折叠为空格
    + 连续空白折叠为单个空格
+ __PROCESS__的展示逻辑仅作用于当前ChatId（当前会话）：
    + AgentId不参与提示作用域匹配
    + 作用域key仅为ChatId
    + 不同ChatId（当前会话）之间互不串扰
+ __PROCESS__临时提示展示5秒：
    + 如果下一轮报文同样__PROCESS__不为空，则需要立即替换并刷新5秒计时
    + 5秒后仅在当前提示的ChatId仍匹配时清空提示
    + 如果choices[0].delta.content归一化后为空，则清空当前会话提示
+ “努力工作中”位置逻辑保持原样，仅跟当前会话busy状态显示/隐藏，不被__PROCESS__替换
+ 纯__PROCESS__报文在实时SSE和历史恢复时都不进入居中对话框正文
+ 不带__PROCESS__且不处于__RETRY__响应段内的正常SSE报文，继续展示在居中对话框正文
+ __RETRY__保持当前收口逻辑：
    + 仅识别choices[0].metadata.__RETRY__
    + 红框重试提示只作用于当前这段SSE响应，不影响下一段响应
    + biz=cli且workflow=sub或workflow=__sub时，__RETRY__不进入红框重试提示
+ finish_reason=error保持现有错误展示逻辑，不被__PROCESS__和__RETRY__覆盖

### 编写代码
+ 以现有site页面技术栈编写以上代码，要求：
    + 在../../index.html中按现有HTML/CSS/JavaScript组织方式实现
    + 不引入新的构建流程和额外运行时依赖
    + 代码简洁，尽量复用现有SSE解析、历史恢复、消息渲染和动画逻辑

### 撰写手册
+ 如有必要同步更新USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
