### 第一性原则
+ 仅可以新增/更新/删除site（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ site介绍：../../REQUIREMENT.md
+ site手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 居中对话框在一次SSE响应生命周期内只维护一个居中assistant响应气泡；中心区域、CLI子任务栏、页脚提示、任务气泡、爆炸快照灯泡共用同一轮SSE流，但各自按不同规则分流
+ 当前居中渲染与爆炸逻辑不再按“过程报文/结果报文”二分，而是按“是否允许进入中心区域”“中心逻辑段`id`是否切换”以及“是否属于控制型报文”处理：
```
{
    "id":
    "biz":
    "workflow":
    "created":
    "choices":
    ...
    其他报文
}
```
    + 对于非`cli/sub`、非`close`报文，中心逻辑段`id`优先取`center:<biz>@<workflow>:<created>`；仅当`created`或`biz/workflow`缺失时，才回退到原始报文`id`
    + 对于`cli/sub`与`close`报文，仍使用原始报文`id`参与各自链路处理，但它们不参与中心爆炸切换
    + `biz=main`且`workflow=base@close`，以及`biz=base`且`workflow=close`，视为收口报文；这类报文需要继续追加到当前居中assistant气泡，不触发爆炸清空
    + 结束标记`[DONE]`仅作为SSE结束标记，不单独触发新的渲染分组
    + `biz=cli`且`workflow=sub`只进入CLI子任务栏展示，不应在居中对话框额外留下空气泡
    + `choices[0].metadata.__PROCESS__`存在时，视为过程提示报文；这类报文只进入footer/inline hint逻辑，不进入中心正文，也不触发中心爆炸切换
    + `choices[0].metadata.__RETRY__`存在、当前报文不是终态错误包，且满足“显式带`delay`或`delta.content`归一化后可识别为重试提示”条件时，视为retry inline hint；这类报文不进入中心正文，也不触发中心爆炸切换
    + `choices[0].metadata.__TASK_START__` / `__TASK_CLOSE__`存在且当前报文不是终态错误包时，视为任务气泡报文；这类报文不进入中心正文，也不触发中心爆炸切换
    + `choices[0].metadata.__TARGET__`只决定inline hint显示在气泡内侧还是外侧，不单独触发分流；如果同时没有`__PROCESS__` / `__RETRY__` / `__TASK_START__` / `__TASK_CLOSE__`，正文仍按普通中心报文处理
+ 居中assistant内容的爆炸/清空只由“非close、非cli/sub、具备可渲染`delta.content`、且当前报文未被inline hint / task badge接管时，中心逻辑段`id`发生切换”或“当前中心段收到`__RESET__`”触发：
    + 同一个中心逻辑段`id`的报文连续到达时，保持现有叠加效果
    + 当新的中心逻辑段`id`开始占据当前居中气泡时，旧中心逻辑段对应的已渲染内容先暂存为待爆炸快照，再清空当前中心正文；新中心逻辑段从当前报文重新开始累加
    + 已经被爆炸淘汰的中心逻辑段`id`后续即使乱序再次到达，也不要重新渲染到中心气泡，而是继续并入该逻辑段已有的爆炸快照
    + 如果当前中心段收到`choices[0].metadata.__RESET__`，则需要把当前已渲染正文立即收入爆炸快照并清空中心正文，再从当前报文重新开始累加；这次reset只重置当前正文，不把当前逻辑段记为已淘汰逻辑段
    + 示例：A1 A2 B1 A3 B2 => A1/A2累加；B1到达时A被爆炸淘汰并清空，B1变成当前内容；A3不再回到中心正文，只追加到A对应爆炸快照；B2继续累加
+ `base@close` / `base.close`收口报文不要“炸”掉前面的居中内容，常见如回传说明或`[耗时 xxx's]`，应继续追加到当前中心响应里
+ 爆炸快照灯泡/思考快照与中心正文切换是两条并行逻辑：
    + 爆炸快照分组`key`优先使用`logical:<packetLogicalId>`；仅当当前报文没有可用逻辑段`id`时，才回退到`snapshot:<biz>@<workflow>:<created>`
    + `workflow`以`__`开头、且当前报文不是`cli/sub`也不是`close`时，需要实时追踪到爆炸快照中，不必等待后续中心逻辑段切换
    + 同一组快照再次到达时，需要按“完整包含优先、边界重叠拼接、否则顺序追加”的规则合并，避免简单重复堆叠
    + 当中心逻辑段切换或`__RESET__`发生且旧正文存在时，旧正文也需要先写入爆炸快照；即使旧正文没有快照分组key，也要作为独立爆炸条目保留下来
    + 只要当前assistant消息不是错误/取消/已完成终态，且存在至少一条爆炸快照，居中assistant气泡左上角就需要保留爆炸灯泡入口；点击或悬浮后展示快照浮层列表
+ 页脚提示与任务气泡沿用最新SSE控制报文规则：
    + `__PROCESS__` 与 `__RETRY__` 共用同一个临时提示位；作用域只按`ChatId`匹配，`agentId`与`__TARGET__`都不参与匹配或清理
    + `__TARGET__`只改变提示展示位置，不改变提示作用域与计时规则；有值时展示在当前等待响应assistant气泡内侧左下，无值时展示在外侧footer
    + `__TASK_START__` / `__TASK_CLOSE__` 任务气泡同样只按`ChatId`隔离；这类临时状态不进入中心正文，也不应在restore中被回放成新的中心assistant消息
+ 爆炸动画范围需要包括最后一个居中SSE响应渲染的整个区域，而不是只处理文本节点
+ 爆炸切换动画时序需要与现代码保持一致：
    + 常规模式：先播放爆炸覆盖层与gap动画，再保留空窗`holdMs=2000ms`，随后播放新内容进入动画`incomingMs=900ms`，爆炸清理动画`cleanupMs=2000ms`
    + 节能/低功耗模式：空窗缩短为`holdMs=320ms`，新内容进入动画`incomingMs=220ms`，爆炸清理动画`cleanupMs=420ms`
+ 重新加载会话或恢复历史未完成响应时，沿用同一套规则恢复：
    + 居中区域仍只恢复一个中心响应气泡
    + 已淘汰的中心逻辑段`id`不要重新回放
    + `close`报文继续回绑到最近的中心assistant气泡
    + `cli/sub`仍进入CLI子任务栏，不在居中生成空气泡
    + 纯`__PROCESS__` / `__RETRY__` / `__TASK_START__` / `__TASK_CLOSE__`控制记录，如果没有中心正文或CLI子任务正文，不要在恢复时单独生成中心assistant消息；但如果当前请求已存在待绑定assistant气泡，这些控制记录仍需要刷新对应的footer hint / 任务气泡UI
    + restore回放时如果某条assistant记录只剩CLI子任务正文，则继续复用该assistant记录承载`cli/sub`链路数据，但居中区域不要显示空正文；如果后续回放又补到了中心正文或爆炸快照，再按同一条assistant记录合并
    + restore回放时如果同一轮请求因为旧逻辑产生了多条assistant消息，需要在恢复结束后折叠为单条主assistant消息，并把正文、爆炸快照、已淘汰逻辑段`id`、CLI子任务内容一并合并到主消息上

### 编写代码
+ 能用开源包的就用开源包
+ 代码简洁，包体积越小越好

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
