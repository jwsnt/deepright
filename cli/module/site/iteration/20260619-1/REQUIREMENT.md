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
+ 在居中对话框SSE响应区接收到响应报文中choices[0].metadata.__RETRY__不为空的报文后，该段报文内容不渲染到居中对话框正文里，也不再插入独立红框，而是展示到当前会话最后一个assistant气泡底部footer预留槽位（与__PROCESS__共用展示位）
```
{
    "choices": [
        {
            "metadata": {
                ...
                "__RETRY__": 任意值,
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
``` 如以下为\n[服务将在 15 秒后重试，错误码 503]\n
data: {"choices":[{"index":0,"metadata":{"__RETRY__":"503","agentId":"X"},"delta":{"content":"\n[服务将在 15 秒后重试，错误码 503]\n","role":"assistant"}}],"workflow":"main","biz":"main",...}
```
+ __RETRY__展示内容取choices[0].delta.content，并按前端现有收口规则做文本归一化：
    + 去掉\r
    + trim首尾空白
    + 连续换行折叠为空格
    + 连续空白折叠为单个空格
+ __RETRY__的展示逻辑仅作用于当前ChatId（当前会话）：
    + AgentId不参与提示作用域匹配
    + 作用域key仅为ChatId
    + 不同ChatId（当前会话）之间互不串扰
+ __RETRY__临时提示展示时间优先取choices[0].metadata.delay（与__RETRY__同级）：
    + delay按毫秒处理
    + 如果delay不存在、非法或小于等于0，则默认展示30秒
    + 如果下一轮报文同样__RETRY__不为空，则需要立即替换并按最新delay刷新计时
    + 到时后仅在当前提示的ChatId仍匹配时清空提示
    + 如果choices[0].delta.content归一化后为空，则清空当前会话提示
+ __RETRY__与__PROCESS__共用同一footer展示位：
    + 当前展示__PROCESS__时，如果收到__RETRY__，立即覆盖成__RETRY__
    + 当前展示__RETRY__时，如果收到新的__PROCESS__，立即覆盖成__PROCESS__
    + 两者各自按自己的展示时长重新计时：__PROCESS__为5秒，__RETRY__为metadata.delay或默认30秒
+ __RETRY__展示样式复用footer提示结构，但颜色改为荧光风格的红色警告态；__PROCESS__原有样式保持不变
+ “努力工作中”位置逻辑保持原样，仅跟当前会话busy状态显示/隐藏，不被__RETRY__替换
+ 纯__RETRY__报文在实时SSE和历史恢复时都不进入居中对话框正文
+ 不带__PROCESS__且不带__RETRY__的正常SSE报文，继续展示在居中对话框正文
+ 原本__RETRY__的红框展示逻辑整体删除：
    + 不保留兼容或回退代码
    + 不再生成独立retry错误消息
    + 当前页面上的红框错误展示仅保留错误场景
+ __RETRY__不再使用前端兜底文案`[服务将重试，错误码 #code]`：
    + 仅使用服务端实际下发的choices[0].delta.content
    + 如果该content归一化后为空，则只清空当前会话提示，不再前端拼兜底文案
+ 红框错误展示仅保留现有错误处理逻辑：
    + HTTP非200或网络错误时，继续按现有错误气泡展示
    + SSE响应报文里只要code不在200-299范围内，也继续按现有错误气泡展示
    + SSE报文finish_reason=error时，继续按现有错误气泡展示
    + 单包__RETRY__不再触发红框
    + 如果错误报文仍携带__RETRY__标记，但不属于真正的retry提示包，则仍按错误红框展示，不进入footer
+ 首次收到__RETRY__时保持现有SSE 503引导链路，但引导第一步的高亮目标改为footer里的__RETRY__提示，而不是独立红框

### 编写代码
+ 以现有site页面技术栈编写以上代码，要求：
    + 在../../index.html中按现有HTML/CSS/JavaScript组织方式实现
    + 不引入新的构建流程和额外运行时依赖
    + 代码简洁，尽量复用现有SSE解析、历史恢复、消息渲染和动画逻辑

### 撰写手册
+ 如有必要同步更新USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
