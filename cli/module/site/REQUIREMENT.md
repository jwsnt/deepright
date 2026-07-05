### 第一性原则
+ 仅可以新增/更新/删除当前需求文档（REQUIREMENT.md）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../DESIGN.md
+ 本模块设计文档：DESIGN.md

### 需求介绍
+ 复制`https://gemini.google.com/app`作为基础页面，完整拷贝，不要遗漏
+ 交互和设计需要完全一致，LOGO使用`DeepRight`
+ 需要本地可以打开和使用

### 布局修正
+ 去掉左上角的`DeepRight`下拉框
+ 左侧`sidebar`和右侧对话框的背景色需要有同色系色差，产生区分度：包括白天模式和黑夜模式

### 使用设置
+ 点击左下角的`设置`，将弹出浮层
+ 在设置中的第一栏固定为`选择Agent`，是一个选择框，接口为/api/agentId
    + 关联Proxy需求：../proxy/iteration/20260419_1/REQUIREMENT.md
    + 下拉列表赋值后需要校验是否匹配成功，如果本地存储的agentId在服务端列表中不存在，则重置为默认选项，避免显示空行
+ 随后是模型和密钥选项，可以有多个，由2个输入组成：
    + 选择模型：是一个下拉框，可选内容为：deepright、deepseek、bigmodel、gemini、openai、kimi、minimax、qwen
    + 填写密钥：是一个密码输入框，位数不固定，但最长250位，默认显示*，提供`小眼睛`显示真实内容
    + 同一个模型只能存在一个配置，模型保存前必须配置密钥，否则禁止保存
+ 浮层下方是保存和取消按钮，保存后所有配置将在浏览器本地存储中，允许在该页面再次打开、编辑、删除
    + 仅点击或取消后可关闭设置
+ 变量信息：
    + 选择的AgentID：变量名`agentId`，必填
    + 模型和密钥：变量名`models`，至少有一组，每个模型名称为`models`的一个key
+ 设置设计的整体风格需要与全局一致
    + 所有下拉框点击和关闭要有淡入淡出

### 使用模型
+ 在对话框下方，`语音输入`按钮左侧添加`小地球`图标，用来选择已经配置的模型
+ 图标风格需要与全局一致

### 页面状态
+ 绑定AgentId + ChatId或仅绑定ChatId的功能：STATE.md

### 左侧边栏（Sidebar）和`新会话`
+ 会话可以有多个，每个会话都有不同且唯一用UUID或类似唯一算法生成的会话ID
+ 新建：点击新会话即表示创建一个新会话（和会话ID）并标记为`活跃会话`
+ 删除：鼠标悬停在会话上左上角出现3x3像素红色小叉，点击后删除会话相关所有存储内容
+ `最近`下方展示会话列表，每个会话只展示最后发送的用户请求，如果是活跃会话需要在响应结束后更新
+ 点击会话列表中不同会话的最后发送请求区域时，切换活跃会话（会话ID）和右侧对话框的对应展示内容，伴随500毫秒有渐变过场

### 对话框交互
+ 初始对话框仅保留问候语（如你好，我是AohRight，有什么可以帮你的吗？），不要有多余的chip（如`撰写`、`计划`等）
+ 对话框内容输入回车后，向`http://127.0.0.1:8080/v1/chat/completions`发送SSE协议
+ 请求报文和响应报文均使用标准的Open AI协议，能用开源包就用开源包实现
+ 请求报文协议：
  + 报文中`model`属性为当前`小地球`勾选的模型，报文Header的`Authorization`为模型密钥，标记Stream=True，消息内容为输入框内容
  + 在`message`同一层，增加`metadata`属性，是一个Map结构，必须包含
      + agentId：如果未选择则在发送消息前提示，并阻止消息发送
      + chat：为`活跃会话`的ID
+ 响应报文协议案例：
```
data: {"id":"cmpl-1305b94c570f447fbde3180560736287","object":"chat.completion.chunk","created":1698999575,"model":"kimi-k2.5","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}
data: {"id":"cmpl-1305b94c570f447fbde3180560736287","object":"chat.completion.chunk","created":1698999575,"model":"kimi-k2.5","choices":[{"index":0,"delta":{"content":"你好"},"finish_reason":null}]}
data: {"id":"cmpl-1305b94c570f447fbde3180560736287","object":"chat.completion.chunk","created":1698999575,"model":"kimi-k2.5","choices":[{"index":0,"delta":{"content":"。"},"finish_reason":null}]}
data: {"id":"cmpl-1305b94c570f447fbde3180560736287","object":"chat.completion.chunk","created":1698999575,"model":"kimi-k2.5","choices":[{"index":0,"delta":{},"finish_reason":"stop","usage":{"prompt_tokens":19,"completion_tokens":13,"total_tokens":32}}]}
data: [DONE]
```
    + 当HTTP响应码200时且没有Done时，表示流还没有结束，需要不断展示报文中的多个content
    + 当HTTP响应码非200或当"finish_reason":"error"时，表示当前碰到错误，用小红框区别
    + 请求和响应以自动滚动的形式展示，每次SSE有新的content展示，都需要自动拉到底部（最新）
    + 请求和响应的展示都需要支持标准Markdown，需要同时支持嵌入打开HTML网站和图片展示
        + 单个SSE展示多个content时，仅做Markdown渲染处理，不要增加额外字符或数据
+ 请求消息到响应结束期间，在消息输入框上5px位置展现内容为"努力工作中"的浮层，宽度为150个像素，高度为50个像素，与消息框垂直对齐，在响应结束或异常后渐变消失
    + 努力工作中浮层在展示期间有渐变闪动效果，同时对话框发送消息按钮变为淡红色的暂停键图标，点击暂停键则终止等待响应，并以动画效果收起"努力工作中"的浮层
    + 点击暂停键后在原努力工作中同位置展示一个半径为5px的气泡破裂效果，持续500毫秒
    + 消息接收完毕或点击暂停键后，对话框恢复原发送消息按钮，并伴随500毫秒闪动效果

### 会话存储
+ 每次请求和响应报文要与会话ID关联保存，以滑动窗口的形式最多存储最后2000条，但仅展示最近500条，向上滚动时才惰性加载历史消息
+ 页面会话Dom数量与展示数量相同，惰性加载和渲染，滑动窗口形式加载，保证页面的小内存占用和滑动流畅

> 新增自 iteration/20260519-1/REQUIREMENT.md
+ 打开浏览器默认收起左右侧边栏
+ 收起时如未配置模型无法操作则自动展开左侧边栏
+ 每次打开默认收起，仅当无法操作时才自动展开
> 新增自 iteration/20260520-1/REQUIREMENT.md
+ 模型设置中增加客户化配置小图标（删除按钮后）
+ 点击展开补充配置：__url、__model、__model_fast、__model_thinking、__model_multi_input、__model_multi_output
+ 不同模型有不同默认值（deepseek/bigmodel/gemini/openai/kimi/minimax/qwen/anthropic）
+ 配置项右侧有重置（恢复默认）和清空按钮
+ 清空后保持为空不自动回填
+ 存在客户化配置时小图标闪动
+ 删除模型通过/api/config持久化
> 新增自 iteration/20260519-1/REQUIREMENT.md
+ 打开浏览器时默认收起左右侧边栏
+ 收起时如因未配置模型无法选模型或发送消息，自动展开左侧边栏
+ 每次打开浏览器必须默认收起，仅当无法操作时才自动展开左侧边栏
> 新增自 iteration/20260520-1/REQUIREMENT.md
+ 模型设置中增加客户化配置小图标，位于删除按钮后
+ 点击展开补充配置：模型URL、基础模型、快速响应、深度思考、多模态输入、多模态输出
+ 不同模型有不同默认值（deepseek、bigmodel、gemini、openai、kimi、minimax、qwen、anthropic）
+ 配置项右侧有重置（恢复默认）和清空按钮
+ 清空后保持为空不自动回填
+ 存在客户化配置时小图标闪动
+ 删除模型需通过/api/config持久化
> 新增自 iteration/20260519-1/REQUIREMENT.md
+ 打开浏览器时默认收起左右侧边栏
+ 收起时如因未配置模型无法选模型或发送消息，自动展开左侧边栏
+ 每次打开浏览器必须默认收起，仅当无法操作时才自动展开左侧边栏
> 新增自 iteration/20260520-1/REQUIREMENT.md
+ 模型设置中增加客户化配置小图标，位于删除按钮后
+ 点击展开补充配置：模型URL、基础模型、快速响应、深度思考、多模态输入、多模态输出
+ 不同模型有不同默认值（deepseek、bigmodel、gemini、openai、kimi、minimax、qwen、anthropic）
+ 配置项右侧有重置（恢复默认）和清空按钮
+ 清空后保持为空不自动回填
+ 存在客户化配置时小图标闪动
+ 删除模型需通过/api/config持久化
> 新增自 iteration/20260519-1/REQUIREMENT.md
+ 打开浏览器时默认收起左右侧边栏
+ 收起时如因未配置模型无法选模型或发送消息，自动展开左侧边栏
+ 每次打开浏览器必须默认收起，仅当无法操作时才自动展开左侧边栏
> 新增自 iteration/20260520-1/REQUIREMENT.md
+ 模型设置中增加客户化配置小图标，位于删除按钮后
+ 点击展开补充配置：模型URL、基础模型、快速响应、深度思考、多模态输入、多模态输出
+ 不同模型有不同默认值（deepseek、bigmodel、gemini、openai、kimi、minimax、qwen、anthropic）
+ 配置项右侧有重置（恢复默认）和清空按钮
+ 清空后保持为空不自动回填
+ 存在客户化配置时小图标闪动
+ 删除模型需通过/api/config持久化

### @文件与技能菜单
> 新增自 iteration/20260419_10/REQUIREMENT.md
+ 选择文件：弹出长度为300px的浮层输入框，水平位置与`努力工作中`相同，监听输入，请求'/api/files=xxx'，展示指定路径可选择的文件或目录名称
> 新增自 iteration/20260419_10/REQUIREMENT.md
+ 选择技能：请求'/api/skills?agentId=xxx'，展示当前Agent可选择的技能名称
> 新增自 iteration/20260419_10/REQUIREMENT.md
+ 气泡插入位置为@触发时的光标位置，需要在@触发时保存光标状态，选择完成后恢复光标再插入，避免因异步操作或焦点转移导致光标丢失
> 新增自 iteration/20260419_10/REQUIREMENT.md
+ 按ESC键或空格，即中断监听并取消浮层（包括@菜单、文件选择、技能选择所有层级），但保留@符号，焦点回到输入框
> 新增自 iteration/20260419_10/REQUIREMENT.md
+ 模糊匹配时，路径拼接基于父目录而非输入值本身，例如：输入~/D匹配到DEV后应拼接为~/DEV而非~/D/DEV
> 新增自 iteration/20260419_17/REQUIREMENT.md
+ 配合hashchange监听兜底，防止任何导航触发页面重载或断开SSE连接
> 新增自 iteration/20260422_1/REQUIREMENT.md
+ 隐藏的file input需要放在body级别，不能放在overflow:hidden的容器内；programmatic click必须在用户手势的同步调用栈内触发，不能在异步操作（如关闭浮层）之后
> 新增自 iteration/20260425_3/REQUIREMENT.md
+ 需要支持同时（一次性）拖动单个或多个文件或目录，转换为多个系统绝对路径
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 需要支持@文件和技能的能力，效果等同：./iteration/20260419_10/REQUIREMENT.md
> 新增自 iteration/20260502-8/REQUIREMENT.md
+ 实际提交内容：[FILE:图片绝对路径]

### SSE流式渲染与展示
> 新增自 iteration/20260419_10/REQUIREMENT.md
+ 如果没有任何技能，则不展示技能菜单，每次实时判断
> 新增自 iteration/20260419_12/REQUIREMENT.md
+ Markdown风格：./iteration/20260419_11/REQUIREMENT.md
> 新增自 iteration/20260419_16/REQUIREMENT.md
+ 文本格式：MD、TXT、XML、JSON等依旧使用Markdown：./iteration/20260419_12/REQUIREMENT.md
> 新增自 iteration/20260419_16/REQUIREMENT.md
+ 不支持的格式仅提示，不展示编辑

**编码约束：**
> 新增自 iteration/20260419_7/REQUIREMENT.md
+ 在对话框中发送用户请求后，不要立即为响应占位，直到报文返回才开始占位渲染
> 新增自 iteration/20260421_1/REQUIREMENT.md
+ 已经新建的会话，但没有任何消息，即使切换也不展示动画（仅一次）

**编码约束：**
> ~~覆盖自 iteration/20260421_3/REQUIREMENT.md，已被 20260421_4 覆盖~~
+ ~~前端调用API时需要处理非JSON响应（如404、405等纯文本错误），解析前先检查Content-Type或用try-catch包裹resp.json()，避免JSON解析异常~~
> 修正/扩展自 iteration/20260421_4/REQUIREMENT.md
+ 前端调用API时需要处理非JSON响应（如404、405等纯文本错误），解析前先检查Content-Type或用try-catch包裹resp.json()，避免JSON解析异常
> 新增自 iteration/20260422_1/REQUIREMENT.md
+ 点击上传展示浮层，选择文件还是目录，需要支持多个文件/目录同时拖动选入
> 新增自 iteration/20260422_1/REQUIREMENT.md
+ 如果有多个则用空格分隔，超出展示则省略
> 新增自 iteration/20260425_2/REQUIREMENT.md
+ 空列表或Agent不存在或已删除则不展示小蜜蜂图标
> 新增自 iteration/20260425_4/REQUIREMENT.md
+ 空列表或Agent不存在或已删除则不展示Thinking图标
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 下半部分展示今天按时间排序的滚动备忘录明细列表，样式为时间轮，先用模拟数据填充
> 新增自 iteration/20260427_2/REQUIREMENT.md
+ 备忘录的tips浮层向下展示
> 新增自 iteration/20260427_4/REQUIREMENT.md
+ 浮层展示排序：模型（如果未选择则默认使用当前居中会话框使用的模型）、周期、思考方式（Thinking、Auto）、开始时间
> 新增自 iteration/20260428-11/REQUIREMENT.md
+ 点击确认，为该AgentID和Chat（会话ID）重新加载并渲染最后一条SSE请求时间之后的对话数据，加载完后Tips已加载完成
> 新增自 iteration/20260428-11/REQUIREMENT.md
+ 恢复会话必须幂等：不得重复渲染当前页面已展示过的本轮 `Q/A`，恢复边界以最后一次实际发送时间为准，不能使用本地预渲染时间
> 新增自 iteration/20260428-14/REQUIREMENT.md
+ 浏览器刷新/重进导致的本地连接中断不属于请求异常，不展示 `网络错误`，需保留未完成状态并在重新进入后自动轮询恢复
> 新增自 iteration/20260428-3/REQUIREMENT.md
+ 在左上备忘录的信封小图标后增加一个切换小图标，点击后备忘录模块的展示切换为已经创建的、非仅一次的备忘录元数据
> 新增自 iteration/20260428-3/REQUIREMENT.md
+ 上半部分占7/8，以列表的形式展示备忘录元数据，需要支持数据过多时的滚动
> 新增自 iteration/20260428-3/REQUIREMENT.md
+ 切换和悬浮展示/收起内容需要有淡入淡出的效果

**编码约束：**
> 新增自 iteration/20260428-3/REQUIREMENT.md
+ 列表各列固定宽度，整体宽度不要超过容器宽度，超出内容以省略号截断；时间列格式为 MM-DD HH:mm，不展示年份
> 新增自 iteration/20260428-5/REQUIREMENT.md
+ 右上角备忘录第一格日历下半部分的备忘录明细列表，用真实数据填充，样式保持时间轮，高度要先保证日历完全展示（不需要滚动）再间隔5px，然后才是列表
> 新增自 iteration/20260428-5/REQUIREMENT.md
+ 列表展示：时间、备忘录内容（超过列宽则用...替代），每30秒刷新一次（重新读取），临近30分钟内待执行的需要有淡橙色的闪烁框
> 新增自 iteration/20260428-5/REQUIREMENT.md
+ 如果当前任务明细为空，使用--无待执行任务--的灰色字体展示空列表，不需要其他内容

**编码约束：**
> 新增自 iteration/20260428-5/REQUIREMENT.md
+ 列表各列固定宽度（时间展示不能换行），整体宽度不要超过容器宽度；时间列格式为 MM-DD HH:mm，不展示年份，超出备忘录内容以省略号截断，保证单行不超出不换行
> 新增自 iteration/20260428-8/REQUIREMENT.md
+ 备忘录明细列表和备忘录元数据列表都是固定高，如果数据过多需要滚动加载，而不是超出容器高度
> 新增自 iteration/20260428-8/REQUIREMENT.md
+ 备忘录明细在展示时如果有多条且需要加载滚动条时需要将相对于当前时间，下一个待执行任务明细（淡橙色）滚动到列表当前展示的第一条
> 新增自 iteration/20260428-8/REQUIREMENT.md
+ 列表自动滚动定位到指定元素时，使用getBoundingClientRect计算相对于滚动容器的实际偏移量，不能依赖offsetTop（受offsetParent层级影响导致定位不准），且需在DOM渲染完成后延迟执行
> 新增自 iteration/20260430-1/REQUIREMENT.md
+ 右上角备忘录选择周期的展示调整：
> 新增自 iteration/20260430-1/REQUIREMENT.md
+ 选择周期后展示的标签文字（如"每30分钟"）字号为12px，不换行（white-space:nowrap）
> 新增自 iteration/20260430-2/REQUIREMENT.md
+ 右上角备忘录明细处于完成状态时，在最前方展示小放大镜图标
> 新增自 iteration/20260430-3/REQUIREMENT.md
+ 备忘录明细列表的操作图标（垃圾桶/放大镜）为互斥关系，合并为同一个图标位，非对应状态的行仅保留单个占位对齐，避免双占位产生多余空白
> 新增自 iteration/20260430-4/REQUIREMENT.md
+ 刷新备忘录元数据列表和备忘录明细列表的同时，滚动到列表当前展示的第一条
> 新增自 iteration/20260430-4/REQUIREMENT.md
+ 滚动需求：./iteration/20260428-8/REQUIREMENT.md:14
> 新增自 iteration/20260430-5/REQUIREMENT.md
+ 备忘录元数据列表展示在向下展开的浮层上，先展示Agent名称，然后换行符后再展示任务内容
> 新增自 iteration/20260430-5/REQUIREMENT.md
+ 备忘录明细列表展示在现有浮层上，增加一行
> 新增自 iteration/20260430-8/REQUIREMENT.md
+ 右侧面板独立渲染的cli+sub内容独立存储在消息的cliSubContent字段，不混入居中对话框的content，避免刷新后回显到居中
> 新增自 iteration/20260430-8/REQUIREMENT.md
+ 右侧面板独立渲染的cli+sub单个气泡高度为70px，字体为10px，和居中会话框相同的自动滚动效果，不要超出外部容器高度
> 新增自 iteration/20260430-9/REQUIREMENT.md
+ 代码块保留水平滚动，滚动条仅悬停时显示（默认隐藏）
> 新增自 iteration/20260501-2/REQUIREMENT.md
+ 展开浮层需要有危险提示（10px红色小字），展示执行规则：仅允许本机请求执行，且包含rm的命令会被拦截
> 新增自 iteration/20260501-3/REQUIREMENT.md
+ 右侧CLI子面板的每一条SSE响应都必须独立渲染为单独气泡，在鼠标悬停在气泡左上角展示一个5x5像素的闪光黏贴小图标，禁止多个响应共用一个外层复制图标
> 新增自 iteration/20260501-4/REQUIREMENT.md
+ 每个会话（Agent+Chat）的居中会话框最多展示2000条最新消息（包括请求加响应）
> 新增自 iteration/20260501-5/REQUIREMENT.md
+ 点击后展示浮动确认是否需要终止命令，点击确认后执行终止
> 新增自 iteration/20260502-4/REQUIREMENT.md
+ 居中对话框中已成功展示过的远程图片，首次渲染后必须写入浏览器本地持久缓存，后续即使源图片链接失效或返回404，页面刷新后仍需优先从本地缓存恢复显示
> 新增自 iteration/20260502-7/REQUIREMENT.md
+ 如果备忘录勾选复用当前会话，则任务完成后点击放大镜查看会话时，无论该会话是否已在左侧存在，都必须基于本次任务明细的执行时间立即恢复并展示本次备忘录新增的问答内容，禁止仅切换到旧会话而不加载本次任务内容

**编码约束：**
> 新增自 iteration/20260504-1/REQUIREMENT.md
+ 扇形按钮的数量和展示文字从api/plugins/meta获取
> 新增自 iteration/20260504-1/REQUIREMENT.md
+ 返回数组中每个元素的name就是扇形菜单展示的名字
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 第二部分：展示该plugin的可选meta和已填meta（默认值）
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 如果定义参数过多，保持窗口大小，添加滚动条
> 新增自 iteration/20260505-1/REQUIREMENT.md
+ 右上角备忘录明细列表在鼠标悬停时弹出的详情浮层中新增一行“类型”字段展示
> 新增自 iteration/20260505-4/REQUIREMENT.md
+ Bug修复：右上角备忘录明细中，已完成任务的“查看”按钮重复点击时，应恢复同一会话但不得重复追加已渲染的历史消息
> 新增自 iteration/20260505-6/REQUIREMENT.md
+ 扇形菜单点击展开时，已经启动的插件应用要展示为绿色闪动

### SSE请求与响应
> 新增自 iteration/20260419_1/REQUIREMENT.md
+ 略微不同于OPEN AI，/v1/chat/completions中的message仅需要包含用户最后（最新）的请求，而不需要历史记录
> 新增自 iteration/20260419_15/REQUIREMENT.md
+ 努力工作中和暂停键需求：./REQUIREMENT.md
> 新增自 iteration/20260419_15/REQUIREMENT.md
+ 如果还在等待，则恢复边缘锐化闪光、"努力工作中"及暂停键效果
> 新增自 iteration/20260428-10/REQUIREMENT.md
+ SSE响应可能分为多条，每次渲染时都需要更新时间和状态
> 新增自 iteration/20260428-10/REQUIREMENT.md
+ SSE还未结束：状态=等待
> 新增自 iteration/20260428-10/REQUIREMENT.md
+ SSE已经结束：状态=完成
> ~~覆盖自 iteration/20260428-13/REQUIREMENT.md，已被 20260428-14 覆盖~~
+ ~~SSE渲染状态需求：./iteration/20260428-10/REQUIREMENT.md~~
> 修正/扩展自 iteration/20260428-14/REQUIREMENT.md
+ SSE渲染状态需求：./iteration/20260428-10/REQUIREMENT.md
> 新增自 iteration/20260428-14/REQUIREMENT.md
+ 异常响应为HTTP错误等无法完成请求的情况
> 新增自 iteration/20260428-14/REQUIREMENT.md
+ 轮询到数据立即渲染，并更新下次使用的最后一条SSE请求时间：../proxy/iteration/20260427-8/REQUIREMENT.md
> 新增自 iteration/20260428-14/REQUIREMENT.md
+ 如果在轮询点了终止会话图标，则按终止会话的流程处理，注意轮询时并没有会话的SSE连接，不要关闭
> 新增自 iteration/20260428-14/REQUIREMENT.md
+ 轮询恢复过程需用户无感，体验应与连续等待SSE返回一致；非终态轮询不得出现明显闪屏、反复提示或整页重绘
> 新增自 iteration/20260430-2/REQUIREMENT.md
+ 加载到SSE响应结束（请求完成）需要按正常的恢复逻辑切换到待输入样式
> 新增自 iteration/20260430-7/REQUIREMENT.md
+ 居中会话框展示SSE响应时需要有打字机效果
> 新增自 iteration/20260501-3/REQUIREMENT.md
+ SSE响应由markdown ```标记包裹，需要去除包裹
> 新增自 iteration/20260501-3/REQUIREMENT.md
+ SSE响应原内容：
> 新增自 iteration/20260502-6/REQUIREMENT.md
+ 需要当前无SSE连接时才可以点击重试图标

### SSE表格横向滚动
> 新增自 iteration/20260520-2/REQUIREMENT.md
+ 确保所有通过SSE或HTML渲染出的表格，在窄容器、长文本、流式更新场景下，仍保持结构清晰、列宽合理、可横向滚动，不出现关键信息列被挤压成竖排的问题。
+ 表格渲染必须支持外层横向滚动，不能依赖表格自身被压缩到容器宽度内显示。
+ 表格宽度应优先按内容自然撑开，在内容超出消息容器时允许横向滚动。
+ "位置""路径""行号""范围""Location""Path""Line""Range"等定位类列，默认应使用不换行策略展示。
+ 消息容器中的普通文本可以换行，但表格单元格不能继承导致按字符强制断开的样式。
+ SSE 流式过程中和流式结束后的最终渲染结果应保持一致，不能在结束后出现列宽塌陷或排版突变。
+ Markdown 表格和原始 HTML 表格都必须使用同一套可读性策略处理。
+ 当表格列数较多或单元格内容较长时，优先出现横向滚动，而不是压缩关键列。
+ 表格表头与数据列应保持对齐，不能因动态渲染导致列错位。

### 会话存储与回放
> 新增自 iteration/20260428-13/REQUIREMENT.md
+ 已经收到并渲染过任意响应，但SSE请求还没有结束，则立即中断SSE连接，并终止会话存储
> 新增自 iteration/20260428-13/REQUIREMENT.md
+ 已经发送请求，但没有收到并渲染过任何响应，则等待3s后中断SSE连接，并终止会话存储
> 新增自 iteration/20260430-8/REQUIREMENT.md
+ 轮询恢复需求：./iteration/20260428-14/REQUIREMENT.md:20
> 新增自 iteration/20260430-9/REQUIREMENT.md
+ 轮询恢复会话数据时，解析SSE原始流中的biz=cli且workflow=sub内容，分离到cliSubContent字段独立存储，恢复完成后同步刷新右侧CLI子任务面板
> 新增自 iteration/20260502-7/REQUIREMENT.md
+ 查看复用会话的备忘录结果时，已有会话不能只做switchChat，必须强制触发一次按该任务execTime起点的restore/reload。验收标准：点击放大镜后，页面能看到这次备忘录新增的Q/A，而不是只看到历史旧内容
> 新增自 iteration/20260504-6/REQUIREMENT.md
+ 日志查看需求：./iteration/20260504-5/REQUIREMENT.md
> 新增自 iteration/20260505-4/REQUIREMENT.md
+ 点击查看时，保留备忘录会话的当前游标，再次使用/api/restore不要把已经恢复过的消息重新追加一遍
> 新增自 iteration/20260505-5/REQUIREMENT.md
+ 左上角点击备忘录明细并查看进行对话恢复时，如果需要恢复通过/api/restore没有任何数据返回，通常是因为当前保存的的最后恢复时间晚于该备忘录消息

### 左侧边栏（Sidebar）和会话列表
> 新增自 iteration/20260419_11/REQUIREMENT.md
+ 对话框中用户气泡需要使用Markdown展示，但不影响输入框和左侧边栏（Sidebar）的会话历史

**编码约束：**
> 新增自 iteration/20260419_12/REQUIREMENT.md
+ 在左侧边栏（Sidebar）下方，设置上方加入高度为250px的虚拟文件系统，用来展示当前Agent工作目录的文件或目录
> 新增自 iteration/20260419_13/REQUIREMENT.md
+ 小垃圾桶风格参考：./iteration/20260419_4/REQUIREMENT.md
> 新增自 iteration/20260419_15/REQUIREMENT.md
+ 切换会话后，需要以会话纬度重新判断是否正在等待消息
> 新增自 iteration/20260419_15/REQUIREMENT.md
+ 不同会话的SSE响应流相互隔离，切换会话时仅更新当前活跃会话的DOM，后台会话的SSE数据写入内存但不操作DOM，切换回时从数据重新渲染
> 新增自 iteration/20260419_15/REQUIREMENT.md
+ 切换会话时，需要先清除当前UI状态（努力工作中、暂停键、边缘锐化闪光），再切换到目标会话并根据目标会话状态重新设置UI
> 新增自 iteration/20260419_2/REQUIREMENT.md
+ 左侧边栏（Sidebar）的会话数量最多不超过10个，超过10个则提示删除后才可以新建会话
> 新增自 iteration/20260419_2/REQUIREMENT.md
+ 浮层文字：会话数量已达上限（10个）
> 新增自 iteration/20260419_20/REQUIREMENT.md
+ 左侧边栏（Sidebar）最上方Logo（DeepRight）的左侧的SVG配合心跳联动
> 新增自 iteration/20260419_20/REQUIREMENT.md
+ 每3秒扫描一次最近心跳：
> 新增自 iteration/20260419_3/REQUIREMENT.md
+ 左侧边栏（Sidebar）最上方Logo（DeepRight）的右侧增加'收起菜单'按钮，用来隐藏左侧边栏
> 新增自 iteration/20260419_3/REQUIREMENT.md
+ 收起菜单相对于左侧边栏右对齐
> 新增自 iteration/20260419_3/REQUIREMENT.md
+ 隐藏后的左侧边栏位置依然要展示'展开菜单'按钮，用来展开左侧边栏
> 新增自 iteration/20260419_3/REQUIREMENT.md
+ 收起菜单相对于左侧边栏左对齐
> 新增自 iteration/20260419_4/REQUIREMENT.md
+ 左侧边栏（Sidebar）删除会话时图标修改为10x10像素，透明度50%的小垃圾桶
> 新增自 iteration/20260419_4/REQUIREMENT.md
+ 小垃圾桶色系风格需要与侧边栏风格保持一致
> 新增自 iteration/20260419_4/REQUIREMENT.md
+ 二次提醒时如果切换会话，则自动收回
> 新增自 iteration/20260421_1/REQUIREMENT.md
+ 切换会话时需要立即终止当前动画（cancelAnimationFrame并清除状态），避免旧动画引用已移除的DOM导致卡住
> 新增自 iteration/20260421_4/REQUIREMENT.md
+ 在设置中新增Agent图标的右侧增加一个删除图标，点击后弹出二次确认框，确认后删除Agent并立即刷新下拉列表
> 新增自 iteration/20260425_1/REQUIREMENT.md
+ 切换会话后检查选择会话
> 新增自 iteration/20260425_1/REQUIREMENT.md
+ 提示框未选择时切换会话，则收起提示框再次后检查选择会话
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 在右侧开辟右对齐Right Sidebar，宽为左侧Sidebar的2倍
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 左侧Sidebar水平分割成对称的上下两部分
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 下半部分左侧左对齐是展示选择时间的控件，精确到分钟，样式为时间轮（小时为选择，分钟为输入），默认时间为当前时间，紧贴右侧为保存和取消按钮，保存样式为信封，取消样式为垃圾桶，与时间轮水平对齐但水平右对齐
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 连同输入框的会话框在左右Sidebar之间居中

**编码约束：**

> 新增自 iteration/20260427_3/REQUIREMENT.md
+ 选择后在时间轮水平左侧展示备选项
> 新增自 iteration/20260427_4/REQUIREMENT.md
+ 浮层宽度为左侧Sidebar宽度250px，右侧对齐，浮层未保存期间任何备忘录配置的改动都立即收起浮层
> ~~覆盖自 iteration/20260427_4/REQUIREMENT.md，已被 20260428-1 覆盖~~
+ ~~所有Cron相关前端高频触发的数据刷新（如切换会话、切换Agent）需要防抖处理，避免短时间内重复请求~~
> ~~覆盖自 iteration/20260428-1/REQUIREMENT.md，已被 20260428-3 覆盖~~
+ ~~所有Cron相关前端高频触发的数据刷新（如切换会话、切换Agent）需要防抖处理，避免短时间内重复请求~~
> 新增自 iteration/20260428-12/REQUIREMENT.md
+ 转圈等待效果是与当前会话绑定的，切换会话时需要实时判断，如果有等待发送则立即展示等待样式，直到计时结束
> 新增自 iteration/20260428-12/REQUIREMENT.md
+ 转圈等待的时间（3s）一旦开始就要固定开始计时的结束时间，即使切换会话也不应该重新计时
> 新增自 iteration/20260428-13/REQUIREMENT.md
+ 终止关闭指定Agent和指定chat（会话ID）的转发请求需求：../proxy/iteration/20260427-9/REQUIREMENT.md
> 新增自 iteration/20260428-14/REQUIREMENT.md
+ 当用户首次进入或切换会话后，检查当前会话是否有SSE连接并正在等待响应
> 新增自 iteration/20260428-14/REQUIREMENT.md
+ 如果无连接则检查当前Agent和Chat（会话ID）在页面渲染的最后非异常响应内容是否标记为完成或取消
> ~~覆盖自 iteration/20260428-3/REQUIREMENT.md，已被 20260428-5 覆盖~~
+ ~~所有Cron相关前端高频触发的数据刷新（如切换会话、切换Agent）需要防抖处理，避免短时间内重复请求~~
> 新增自 iteration/20260428-3/REQUIREMENT.md
+ 下半部分右侧展示切换小图标，点击则切换回原来的备忘录布局
> ~~覆盖自 iteration/20260428-5/REQUIREMENT.md，已被 20260428-6 覆盖~~
+ ~~所有Cron相关前端高频触发的数据刷新（如切换会话、切换Agent）需要防抖处理，避免短时间内重复请求~~
> 新增自 iteration/20260428-5/REQUIREMENT.md
+ 如果存在多条备忘录明细超过滚动条时，自动滚到离开当前时间最近的待执行明细
> ~~覆盖自 iteration/20260428-6/REQUIREMENT.md，已被 20260428-7 覆盖~~
+ ~~所有Cron相关前端高频触发的数据刷新（如切换会话、切换Agent）需要防抖处理，避免短时间内重复请求~~
> ~~覆盖自 iteration/20260428-7/REQUIREMENT.md，已被 20260428-8 覆盖~~
+ ~~所有Cron相关前端高频触发的数据刷新（如切换会话、切换Agent）需要防抖处理，避免短时间内重复请求~~
> ~~覆盖自 iteration/20260428-8/REQUIREMENT.md，已被 20260428-9 覆盖~~
+ ~~所有Cron相关前端高频触发的数据刷新（如切换会话、切换Agent）需要防抖处理，避免短时间内重复请求~~
> 修正/扩展自 iteration/20260428-9/REQUIREMENT.md
+ 所有Cron相关前端高频触发的数据刷新（如切换会话、切换Agent）需要防抖处理，避免短时间内重复请求
> 新增自 iteration/20260428-9/REQUIREMENT.md
+ 小垃圾桶风格参考会话列表：./iteration/20260419_4/REQUIREMENT.md
> 新增自 iteration/20260428-9/REQUIREMENT.md
+ 仅状态为待执行时展示小垃圾桶图标，即使不需要展示小垃圾桶时依旧要保留每列对齐
> 新增自 iteration/20260430-2/REQUIREMENT.md
+ 放大镜图标样式和展示时机同小垃圾桶：./iteration/20260428-9/REQUIREMENT.md:6
> 新增自 iteration/20260430-2/REQUIREMENT.md
+ 仅状态为已完成时展示小放大镜图标，即使不需要展示小放大镜时依旧要保留每列对齐（包括小垃圾桶和小放大镜同时存在布局）
> 新增自 iteration/20260430-2/REQUIREMENT.md
+ 展开会话前需要确认会话数量：./iteration/20260419_2/REQUIREMENT.md:20
> 新增自 iteration/20260430-2/REQUIREMENT.md
+ 展开会话前需要确认对应Agent、Chat（会话ID）的会话不存在（一个Agent、Chat）只能有一个展示会话，同时跳转到已经创建过的会话
> 新增自 iteration/20260430-2/REQUIREMENT.md
+ 如果会话不存在，则自动创建对应Agent、Chat（会话ID）、模型和思考模式的会话，并切换到该会话
> 新增自 iteration/20260430-2/REQUIREMENT.md
+ 左侧sidebar中会话展示内容任务明细的内容
> 新增自 iteration/20260430-3/REQUIREMENT.md
+ 右上角备忘录明细的小垃圾桶和小放大镜图标方法1.5倍
> 新增自 iteration/20260430-4/REQUIREMENT.md
+ 删除Agent时需要同时删除该Agent的备忘录元数据和备忘录明细，同时立即刷左上的备忘录元数据列表和备忘录明细列表、同时删除左侧Sidebar中该Agent的其他会话（Chat)
> 新增自 iteration/20260430-8/REQUIREMENT.md
+ 将右侧边栏第二排左格子合并为一个（宽度加长）
> 新增自 iteration/20260430-8/REQUIREMENT.md
+ 居中会话框展示SSE响应时如果报文的属性biz=cli且workflow=sub则不要展示在居中对话框，重定向到右侧边栏第二排左格子独立渲染
> 新增自 iteration/20260430-8/REQUIREMENT.md
+ 右侧面板独立渲染的cli+sub内容与当前Agent+Chat绑定，切换会话或刷新页面后需要恢复展示
> 新增自 iteration/20260430-9/REQUIREMENT.md
+ 在右侧Sidebar第一栏和第二栏之间增加高度为30px的分割带，让2个部分看起来有一定分区
> 新增自 iteration/20260430-9/REQUIREMENT.md
+ 右侧第二栏CLI子任务面板样式调整：
> 新增自 iteration/20260430-9/REQUIREMENT.md
+ 右侧CLI子任务面板实时渲染SSE内容时，必须校验当前SSE流的会话ID是否为活跃会话，非活跃会话的cli+sub内容只存储不渲染，避免不同会话的内容串显
> 新增自 iteration/20260501-1/REQUIREMENT.md
+ 右侧Sidebar第一排的高度增加到绝对值220px、第二排的高度增加到绝对值250px
> 新增自 iteration/20260501-1/REQUIREMENT.md
+ 右侧Sidebar第二排仅CLI子任务列表容器底部固定5px占位留白，确保最后一个气泡不再紧贴面板底边
> 新增自 iteration/20260501-1/REQUIREMENT.md
+ 右侧Sidebar第二排仅CLI子任务气泡使用霓虹荧光绿色而气泡外层保持整体原有色调，如果CLI子任务面板有新SSE响应刷新则高频闪动3秒
> 新增自 iteration/20260501-4/REQUIREMENT.md
+ 每个会话（Agent+Chat）的右侧CLI子面板最多展示100条最新消息
> 新增自 iteration/20260501-5/REQUIREMENT.md
+ 右侧CLI子面板的每一条SSE响应的黏贴图标后展示一个终止小图标，与黏贴小图标使用不用色差
> 新增自 iteration/20260502-7/REQUIREMENT.md
+ 使用当前会话ID作为CHAT_ID，并在后续任务作为转发/v1/chat/completions的chat使用
> 新增自 iteration/20260504-1/REQUIREMENT.md
+ 左上新会话按钮右侧水平位置新增一个即时通讯的图标（紧贴右侧），原新会话按钮自适应缩进
> 新增自 iteration/20260504-1/REQUIREMENT.md
+ 扇形展开后，被遮挡背景（新会话按钮、会话列表、最近）做模糊处理，收起后恢复
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 是否复用会话勾选后需要在右侧虚拟文件夹Tips提示，提示内容和样式同右上角备忘录的复用会话
> 新增自 iteration/20260504-5/REQUIREMENT.md
+ 日志读取的展示风格需要与右侧CMD子面板中的样式相同，但不需要复制和终止按钮
> 新增自 iteration/20260505-3/REQUIREMENT.md
+ 当会话数量已达上限时，提示内容（浮层）需要相对于居中对话框水平垂直居中，参考扇形菜单
> 新增自 iteration/20260509-1/REQUIREMENT.md
+ 左上角扇形菜单圆按钮:中文插件名最多显示4个字,2/3/4个字自动放大圆按钮保持圆形,中文不强制大写和拉开字距
> 新增自 iteration/20260509-2/REQUIREMENT.md
+ 左上角扇形菜单主入口在任一插件已启动时同步显示绿色闪动,不只是展开后插件圆按钮变绿

### 其他补充
> ~~覆盖自 iteration/20260419_16/REQUIREMENT.md，已被 20260419_18 覆盖~~
+ ~~Proxy需求：../proxy/iteration/20260419_10/REQUIREMENT.md~~
> 新增自 iteration/20260419_17/REQUIREMENT.md
+ 禁止后退、转跳等所有会离开页面本身的操作或脚本执行

**编码约束：**
> 新增自 iteration/20260419_17/REQUIREMENT.md
+ 禁止后退、跳转等所有会离开页面本身的操作或脚本执行
> 新增自 iteration/20260419_17/REQUIREMENT.md
+ 使用replaceState替换history条目而非pushState压栈，确保后退按钮无可退记录
> ~~覆盖自 iteration/20260419_18/REQUIREMENT.md，已被 20260419_20 覆盖~~
+ ~~Proxy需求：../proxy/iteration/20260419_5/REQUIREMENT.md~~
> 新增自 iteration/20260419_18/REQUIREMENT.md
+ REQUIREMENT.md，则在当前Agent工作目录下模糊查找
> 新增自 iteration/20260419_18/REQUIREMENT.md
+ 无法无法匹配任何文件，则不做任何提示
> ~~覆盖自 iteration/20260419_18/REQUIREMENT.md，已被 20260419_19 覆盖~~
+ ~~查找明确的单一路径，如/a/b/c.md或./iteration/20260510-1/wiki_md/index.md、./iteration/20260510-1/wiki_md/index.md或单个像文件名的内容，如REQUIREMENT.md~~
> ~~覆盖自 iteration/20260419_18/REQUIREMENT.md，已被 20260419_19 覆盖~~
+ ~~不查找：~~
> ~~覆盖自 iteration/20260419_18/REQUIREMENT.md，已被 20260419_19 覆盖~~
+ ~~一整句中文描述~~
> ~~覆盖自 iteration/20260419_18/REQUIREMENT.md，已被 20260419_19 覆盖~~
+ ~~带标点的说明文字~~
> ~~覆盖自 iteration/20260419_18/REQUIREMENT.md，已被 20260419_19 覆盖~~
+ ~~混合说明和路径的长句~~
> ~~覆盖自 iteration/20260419_18/REQUIREMENT.md，已被 20260419_19 覆盖~~
+ ~~多行一起选中的路径块~~
> 修正/扩展自 iteration/20260419_19/REQUIREMENT.md
+ 查找明确的单一路径，如/a/b/c.md或./iteration/20260510-1/wiki_md/index.md、./iteration/20260510-1/wiki_md/index.md或单个像文件名的内容，如REQUIREMENT.md
> 修正/扩展自 iteration/20260419_19/REQUIREMENT.md
+ 不查找：
> 修正/扩展自 iteration/20260419_19/REQUIREMENT.md
+ 一整句中文描述
> 修正/扩展自 iteration/20260419_19/REQUIREMENT.md
+ 带标点的说明文字
> 修正/扩展自 iteration/20260419_19/REQUIREMENT.md
+ 混合说明和路径的长句
> 修正/扩展自 iteration/20260419_19/REQUIREMENT.md
+ 多行一起选中的路径块
> 新增自 iteration/20260419_19/REQUIREMENT.md
+ 前后标点、空格、空行分隔的文字
> 新增自 iteration/20260419_19/REQUIREMENT.md
+ 链接、超链接
> ~~覆盖自 iteration/20260419_20/REQUIREMENT.md，已被 20260421_3 覆盖~~
+ ~~Proxy需求：../proxy/iteration/20260419_11/REQUIREMENT.md~~
> 新增自 iteration/20260419_20/REQUIREMENT.md
+ 连续5次心跳失败才算失败

**编码约束：**
> 新增自 iteration/20260419_20/REQUIREMENT.md
+ 切换状态时必须先remove所有状态class（hb-ok、hb-fail、hb-task）再add目标class，包括错误分支和catch分支，避免旧状态class残留
> 新增自 iteration/20260419_20/REQUIREMENT.md
+ hue-rotate角度需要根据原色（accent约217°）精确计算到目标色相，蓝色到绿色（120°）应为-97deg而非+90deg
> 新增自 iteration/20260421_1/REQUIREMENT.md
+ 使用八角星（The Octagram）
> 新增自 iteration/20260421_2/REQUIREMENT.md
+ 如果当前用户自己指定的模式，则在不关闭的情况下不进行自动切换
> 新增自 iteration/20260421_2/REQUIREMENT.md
+ 如果用户没有自己指定，即使用户在使用中，也需要切换
> ~~覆盖自 iteration/20260421_3/REQUIREMENT.md，已被 20260421_4 覆盖~~
+ ~~Proxy需求：../proxy/iteration/20260420_1/REQUIREMENT.md~~
> 新增自 iteration/20260421_3/REQUIREMENT.md
+ 名称要符合操作系统命名规范，只允许英文，不允许特殊字符和空格

**编码约束：**
> ~~覆盖自 iteration/20260421_4/REQUIREMENT.md，已被 20260421_6 覆盖~~
+ ~~Proxy需求：../proxy/iteration/20260420_2/REQUIREMENT.md~~
> ~~覆盖自 iteration/20260421_6/REQUIREMENT.md，已被 20260421_6 覆盖~~
+ ~~Proxy需求：../proxy/iteration/20260420_3/REQUIREMENT.md~~
> ~~覆盖自 iteration/20260421_6/REQUIREMENT.md，已被 20260422_1 覆盖~~
+ ~~Proxy需求：../proxy/iteration/20260419_3/REQUIREMENT.md~~
> 新增自 iteration/20260421_6/REQUIREMENT.md
+ 打开：打开当前相对于Agent的workspace的目录
> 修正/扩展自 iteration/20260422_1/REQUIREMENT.md
+ Proxy需求：../proxy/iteration/20260422_1/REQUIREMENT.md
> 新增自 iteration/20260422_1/REQUIREMENT.md
+ 按钮文案：文件 目录
> 新增自 iteration/20260422_1/REQUIREMENT.md
+ 上传目录需要保持相对目录结构，例如上传xxx/yyy目录，那么yyy目录及其子孙目录结构都要保留
> 新增自 iteration/20260422_1/REQUIREMENT.md
+ 上传成功则内容为上传后相对Agent工作目录的上传`目录`，高30px，5秒后自动收起
> 新增自 iteration/20260422_1/REQUIREMENT.md
+ 上传失败则内容为失败原因
> 新增自 iteration/20260422_1/REQUIREMENT.md
+ 拖动上传时必须等待所有拖入项（文件和目录）全部遍历完成后再统一发起上传请求，不能逐个发起
> 新增自 iteration/20260422_2/REQUIREMENT.md
+ 当访问的Host不是localhost或127.0.0.1（本地访问时）隐藏：
> 新增自 iteration/20260422_3/REQUIREMENT.md
+ 会话提交执行时也需要使用当前会话绑定的Agent和模型

**编码约束：**
> 新增自 iteration/20260422_3/REQUIREMENT.md
+ 会话绑定的状态（Agent、模型等）需要在新建会话时立即绑定当前值，并持久化到localStorage随saveChats一起存取，不能仅存在内存中
> 新增自 iteration/20260422_4/REQUIREMENT.md
+ 悬浮框需求：./iteration/20260422_3/REQUIREMENT.md
> 新增自 iteration/20260425_1/REQUIREMENT.md
+ 删除后立即检查当前会话
> 新增自 iteration/20260425_1/REQUIREMENT.md
+ 如果选择会话的Agent存在则不再提示，如果不存在则重新提示
> 新增自 iteration/20260425_2/REQUIREMENT.md
+ 蜂群（swarm）配置项：开启/关闭，蜂群描述
> 新增自 iteration/20260425_2/REQUIREMENT.md
+ Config需求：../proxy/iteration/20260425-1/REQUIREMENT.md
> ~~覆盖自 iteration/20260425_2/REQUIREMENT.md，已被 20260425_4 覆盖~~
+ ~~读取服务端实时数据的fetch必须带时间戳参数防缓存，读取失败时UI重置为默认值~~
> ~~覆盖自 iteration/20260425_2/REQUIREMENT.md，已被 20260425_4 覆盖~~
+ ~~多个异步调用共享同一全局状态时，必须用序列号丢弃过期回调结果~~
> 修正/扩展自 iteration/20260425_4/REQUIREMENT.md
+ 读取服务端实时数据的fetch必须带时间戳参数防缓存，读取失败时UI重置为默认值
> 修正/扩展自 iteration/20260425_4/REQUIREMENT.md
+ 多个异步调用共享同一全局状态时，必须用序列号丢弃过期回调结果
> 新增自 iteration/20260425_4/REQUIREMENT.md
+ Config的需求：../agent/REQUIREMENT.md，Thinking时属性thinking=true，Auto则为false
> 新增自 iteration/20260425_4/REQUIREMENT.md
+ 每个Agent的Thinking配置是独立的，与蜂巢需求一致
> 新增自 iteration/20260425_4/REQUIREMENT.md
+ 蜂巢需求：./iteration/20260425_2/REQUIREMENT.md
> 新增自 iteration/20260426_1/REQUIREMENT.md
+ Thinking按钮需求：./iteration/20260425_4/REQUIREMENT.md
> 新增自 iteration/20260426_1/REQUIREMENT.md
+ 每个Agent的Thinking配置是独立的

**编码约束：**
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 上半部分切分为田字四方格
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 左上第一个格子水平切分为上下两部分
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 右上第一个格子切分为上下两部分
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 需要支持选择Thinking滑动按钮（从右上对齐改为右下对齐）：./iteration/20260426_1/REQUIREMENT.md
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 如果用户焦点不在时间轮，则每10秒同步当前系统时间
> 新增自 iteration/20260427_2/REQUIREMENT.md
+ 拖入备忘录需求：./iteration/20260427_1/REQUIREMENT.md
> 新增自 iteration/20260427_2/REQUIREMENT.md
+ 光圈需求：./iteration/20260425_3/REQUIREMENT.md
> ~~覆盖自 iteration/20260427_4/REQUIREMENT.md，已被 20260427_4 覆盖~~
+ ~~Proxy需求：../proxy/iteration/20260427-2/REQUIREMENT.md~~
> ~~覆盖自 iteration/20260427_4/REQUIREMENT.md，已被 20260428-11 覆盖~~
+ ~~Proxy需求：../proxy/iteration/20260427-2/REQUIREMENT.md~~
> 新增自 iteration/20260427_4/REQUIREMENT.md
+ 对周期任务（工作日/自然日）在创建后会立即生成后5天内的所有任务明细，不用等定时器，对一次性任务如果执行时间早于当前时间（比如当前时间是10点01分，10点禁止执行，10点01分可以执行，10点02分可以执行），禁止创建并提示Tips（等于当前时间可以创建）
> ~~覆盖自 iteration/20260428-11/REQUIREMENT.md，已被 20260428-14 覆盖~~
+ ~~Proxy需求：../proxy/iteration/20260427-10/REQUIREMENT.md~~
> ~~覆盖自 iteration/20260428-14/REQUIREMENT.md，已被 20260428-2 覆盖~~
+ ~~Proxy需求：../proxy/iteration/20260427-10/REQUIREMENT.md~~
> 新增自 iteration/20260428-14/REQUIREMENT.md
+ 如果有连接则保持原逻辑
> 新增自 iteration/20260428-14/REQUIREMENT.md
+ 对话状态需求：./REQUIREMENT.md
> 新增自 iteration/20260428-14/REQUIREMENT.md
+ 终止会话需求：./iteration/20260428-13/REQUIREMENT.md

**编码约束：**
> ~~覆盖自 iteration/20260428-2/REQUIREMENT.md，已被 20260428-3 覆盖~~
+ ~~Proxy需求：../proxy/iteration/20260427-3/REQUIREMENT.md~~
> ~~覆盖自 iteration/20260428-3/REQUIREMENT.md，已被 20260428-5 覆盖~~
+ ~~Proxy需求：../proxy/iteration/20260427-2/REQUIREMENT.md~~
> 新增自 iteration/20260428-3/REQUIREMENT.md
+ 布局（右上）保持原来的上下两部分：./iteration/20260427_1/REQUIREMENT.md
> 新增自 iteration/20260428-3/REQUIREMENT.md
+ 周期 精确到小时的时间点 模型类型 思考模式
> 新增自 iteration/20260428-3/REQUIREMENT.md
+ 在列表最后增加一个大叉的按钮，用于表示删除
> 新增自 iteration/20260428-3/REQUIREMENT.md
+ 删除后立即刷新备忘录列表和左上日历的备忘录明细列表
> ~~覆盖自 iteration/20260428-5/REQUIREMENT.md，已被 20260428-9 覆盖~~
+ ~~Proxy需求：../proxy/iteration/20260427-5/REQUIREMENT.md~~
> 新增自 iteration/20260428-5/REQUIREMENT.md
+ 备忘录需求：./iteration/20260427_1/REQUIREMENT.md
> ~~覆盖自 iteration/20260428-6/REQUIREMENT.md，已被 20260428-8 覆盖~~
+ ~~备忘录元数据：./iteration/20260428-3/REQUIREMENT.md~~
> ~~覆盖自 iteration/20260428-7/REQUIREMENT.md，已被 20260428-8 覆盖~~
+ ~~备忘录明细需求：./iteration/20260428-5/REQUIREMENT.md~~
> 修正/扩展自 iteration/20260428-8/REQUIREMENT.md
+ 备忘录元数据：./iteration/20260428-3/REQUIREMENT.md
> ~~覆盖自 iteration/20260428-8/REQUIREMENT.md，已被 20260428-8 覆盖~~
+ ~~备忘录明细需求：./iteration/20260428-5/REQUIREMENT.md~~
> ~~覆盖自 iteration/20260428-8/REQUIREMENT.md，已被 20260428-9 覆盖~~
+ ~~备忘录明细需求：./iteration/20260428-5/REQUIREMENT.md~~
> ~~覆盖自 iteration/20260428-9/REQUIREMENT.md，已被 20260501-2 覆盖~~
+ ~~Proxy需求：../proxy/iteration/20260427-6/REQUIREMENT.md~~
> 修正/扩展自 iteration/20260428-9/REQUIREMENT.md
+ 备忘录明细需求：./iteration/20260428-5/REQUIREMENT.md
> 新增自 iteration/20260428-9/REQUIREMENT.md
+ 任务明细列表刷新：
> 新增自 iteration/20260428-9/REQUIREMENT.md
+ 指定状态成功后立即刷新任务明细列表
> 新增自 iteration/20260428-9/REQUIREMENT.md
+ 如果鼠标不在任务明细列表中，每30秒刷新一次任务明细列表

**编码约束：**
> 新增自 iteration/20260430-5/REQUIREMENT.md
+ 右上角的备忘录任务明细列表和备忘录元数据列表增加Agent名称
> 新增自 iteration/20260430-6/REQUIREMENT.md
+ 选择后onchange不触发，浏览器会认为值未变化
> 新增自 iteration/20260430-9/REQUIREMENT.md
+ 分割带左对齐标题：正在执行的系统指令
> 新增自 iteration/20260501-1/REQUIREMENT.md
+ CLI子板板：./iteration/20260430-9/REQUIREMENT.md
> 新增自 iteration/20260501-1/REQUIREMENT.md
+ 分割带需求：./iteration/20260430-9/REQUIREMENT.md
> ~~覆盖自 iteration/20260501-2/REQUIREMENT.md，已被 20260502-7 覆盖~~
+ ~~Proxy需求：../proxy/iteration/20260501-1/REQUIREMENT.md~~
> 新增自 iteration/20260501-2/REQUIREMENT.md
+ 执行成功后需要有成功提示
> 新增自 iteration/20260501-2/REQUIREMENT.md
+ 每次重新打开都要重置为空，不保留之前的执行记录
> 新增自 iteration/20260501-3/REQUIREMENT.md
+ 黏贴的内容：HELLO WORLD
> 新增自 iteration/20260501-3/REQUIREMENT.md
+ 不允许复制整组聚合内容
> 新增自 iteration/20260501-4/REQUIREMENT.md
+ CLI子面板需求：./iteration/20260501-3/REQUIREMENT.md
> 新增自 iteration/20260501-5/REQUIREMENT.md
+ Proxy终止需求：../proxy/iteration/20260501-2/REQUIREMENT.md
> 新增自 iteration/20260502-1/REQUIREMENT.md
+ 修改后的图片不要覆盖原图片，在原图目录下新建原图片名加时间戳的新图片
> 新增自 iteration/20260502-1/REQUIREMENT.md
+ 新建图片需求：../proxy/iteration/20260502-1/REQUIREMENT.md
> 新增自 iteration/20260502-2/REQUIREMENT.md
+ 需要支持的拖拉图像：圆、椭圆、矩形、三角形
> 新增自 iteration/20260502-2/REQUIREMENT.md
+ 需要支持以上图像的拖拉放大缩小
> 新增自 iteration/20260502-3/REQUIREMENT.md
+ 拖拉图像需求：./iteration/20260502-2/REQUIREMENT.md
> 新增自 iteration/20260502-3/REQUIREMENT.md
+ 透明图片需求：./iteration/20260502-1/REQUIREMENT.md
> ~~覆盖自 iteration/20260502-7/REQUIREMENT.md，已被 20260502-8 覆盖~~
+ ~~Proxy需求：../proxy/iteration/20260502-3/REQUIREMENT.md~~
> ~~覆盖自 iteration/20260502-8/REQUIREMENT.md，已被 20260503-1 覆盖~~
+ ~~Proxy需求：../proxy/iteration/20260502-1/REQUIREMENT.md~~
> 新增自 iteration/20260502-8/REQUIREMENT.md
+ 超长文件名会省略显示
> 新增自 iteration/20260502-8/REQUIREMENT.md
+ 删除按钮保持可见
> ~~覆盖自 iteration/20260503-1/REQUIREMENT.md，已被 20260504-1 覆盖~~
+ ~~Proxy需求：../proxy/iteration/20260503-2/REQUIREMENT.md~~
> ~~覆盖自 iteration/20260504-1/REQUIREMENT.md，已被 20260504-5 覆盖~~
+ ~~Proxy需求：../proxy/iteration/20260503-9/REQUIREMENT.md~~
> 新增自 iteration/20260504-1/REQUIREMENT.md
+ 扇形菜单样式参考：style.jpeg
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 浮动分三部分
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 获取meta需求：../proxy/iteration/20260503-9/REQUIREMENT.md
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 最后一个参数与按钮的分隔线距离15px
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 点击重启，立即通过/api/plugins/config更新配置，成功后通过/api/plugins/start启动插件
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ Config需求：../proxy/iteration/20260503-11/REQUIREMENT.md
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 开启关闭需求：../proxy/iteration/20260503-14/REQUIREMENT.md
> ~~覆盖自 iteration/20260504-2/REQUIREMENT.md，已被 20260505-6 覆盖~~
+ ~~需要使用/api/plugins/status判断是否可关闭~~
> ~~覆盖自 iteration/20260504-2/REQUIREMENT.md，已被 20260505-6 覆盖~~
+ ~~是否已启动需求：../proxy/iteration/20260503-16/REQUIREMENT.md~~
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 已关闭则关闭状态变灰，重启按钮始终可用
> ~~覆盖自 iteration/20260504-3/REQUIREMENT.md，已被 20260504-4 覆盖~~
+ ~~扇形菜单：./iteration/20260504-2/REQUIREMENT.md~~
> ~~覆盖自 iteration/20260504-3/REQUIREMENT.md，已被 20260504-4 覆盖~~
+ ~~需要有同样的背景模糊处理~~
> ~~覆盖自 iteration/20260504-4/REQUIREMENT.md，已被 20260505-3 覆盖~~
+ ~~扇形菜单：./iteration/20260504-2/REQUIREMENT.md~~
> ~~覆盖自 iteration/20260504-4/REQUIREMENT.md，已被 20260505-3 覆盖~~
+ ~~需要有同样的背景模糊处理~~
> ~~覆盖自 iteration/20260504-5/REQUIREMENT.md，已被 20260505-1 覆盖~~
+ ~~Proxy需求：../proxy/iteration/20260503-10/REQUIREMENT.md~~
> 新增自 iteration/20260504-6/REQUIREMENT.md
+ 扇形菜单需求：./iteration/20260504-2/REQUIREMENT.md
> 新增自 iteration/20260504-6/REQUIREMENT.md
+ 更新配置需求需求：../proxy/iteration/20260503-11/REQUIREMENT.md
> 新增自 iteration/20260504-6/REQUIREMENT.md
+ 配置更新成功后启动插件
> 新增自 iteration/20260504-6/REQUIREMENT.md
+ 启动插件需求：../proxy/iteration/20260503-13/REQUIREMENT.md
> 修正/扩展自 iteration/20260505-1/REQUIREMENT.md
+ Proxy需求：../proxy/iteration/20260503-13/REQUIREMENT.md
> 新增自 iteration/20260505-1/REQUIREMENT.md
+ Cron需求：../cron/iteration/20260502-6/REQUIREMENT.md
> 新增自 iteration/20260505-2/REQUIREMENT.md
+ Proxy上传需求：../proxy/iteration/20260422_1/REQUIREMENT.md
> 新增自 iteration/20260505-2/REQUIREMENT.md
+ 上传时需要锁定界面，防止操作，效果同：./iteration/20260504-6/REQUIREMENT.md
> 新增自 iteration/20260505-2/REQUIREMENT.md
+ 备忘录上传需求：
> ~~覆盖自 iteration/20260505-3/REQUIREMENT.md，已被 20260505-6 覆盖~~
+ ~~扇形菜单：./iteration/20260504-2/REQUIREMENT.md~~
> ~~覆盖自 iteration/20260505-3/REQUIREMENT.md，已被 20260505-7 覆盖~~
+ ~~需要有同样的背景模糊处理~~
> 修正/扩展自 iteration/20260505-6/REQUIREMENT.md
+ 需要使用/api/plugins/status判断是否可关闭
> 修正/扩展自 iteration/20260505-6/REQUIREMENT.md
+ 是否已启动需求：../proxy/iteration/20260503-16/REQUIREMENT.md
> 修正/扩展自 iteration/20260505-6/REQUIREMENT.md
+ 扇形菜单：./iteration/20260504-2/REQUIREMENT.md
> 修正/扩展自 iteration/20260505-7/REQUIREMENT.md
+ 需要有同样的背景模糊处理

### 迭代合并需求（按主题分类）
#### API请求与消息
* 居中对话框思考模式左侧增加HTML输出开关，开启后转发/v1/chat/completions请求增加metadata属性html [新增于 it#99 20260517-4]
* 为虚拟文件系统中SOUL.md文件添加小灯泡图标，点击后展示浮层提示用户是否需要整理 [新增于 it#92 20260516-1]
* 为虚拟文件系统中USER.md文件添加小灯泡图标，点击后展示浮层提示用户是否需要整理 [新增于 it#92 20260516-2]
* 为右下角知识库WIKI添加小灯泡图标，点击后展示浮层提示用户是否需要整理知识库 [新增于 it#92 20260516-3]
* 设置中思考模式下移到蜂群开关右侧，不再与当前会话的思考模式开关联动 [新增于 it#94 20260516-5]
* 为蜂群增加选择模型下拉框（风格同备忘录的小地球图标） [新增于 it#94 20260516-5]
* 创建新会话时，思考模式默认为Auto（不要继承上一次使用会话） [新增于 it#93 20260516-4]
* 右侧Sidebar增加收起菜单，点击后收起并将空间释放给居中对话框 [新增于 it#96 20260517-1]
* 右侧Sidebar默认收起，收起时有微弱的闪光提示可以打开 [新增于 it#96 20260517-1]
* 左上角插件入口每30秒自动刷新一次插件列表，并更新按钮状态 [新增于 it#95 20260517-8]
* 略微不同于OPEN AI，/v1/chat/completions中的message仅需要包含用户最后（最新）的请求，而不需要历史记录 [新增于 it#1 20260419_1]
* 选择文件：弹出长度为300px的浮层输入框，水平位置与`努力工作中`相同，监听输入，请求'/api/files=xxx'，展示指定路径可选择的文件或目录名称 [新增于 it#2 20260419_10]
* 选择技能：请求'/api/skills?agentId=xxx'，展示当前Agent可选择的技能名称 [新增于 it#2 20260419_10]
* 选择具体文件或目录，将在对话框中`待提交内容的光标处`展示特殊反差色系气泡，气泡展示内容为`[FILE:文件路径的最后一级文件或目录名称]`，气泡在提交请求是内容替换为`[FILE:文件或目录绝对路径]` [新增于 it#2 20260419_10]
* 选择具体技能，将在对话框中`待提交内容的光标处`展示特殊反差色系气泡，气泡展示内容和提交请求的内容均为`[SKILL:技能名称]` [新增于 it#2 20260419_10]
* 对话框用户提问气泡的边缘锐化闪光效果、"努力工作中"、暂停键效果与会话绑定，仅在当前会话正在等待消息时触发 [新增于 it#7 20260419_15]
* 切换会话后，需要以会话纬度重新判断是否正在等待消息 [新增于 it#7 20260419_15]
* 如果已经完成，则不用恢复，保持再次等待发送消息的效果 [新增于 it#7 20260419_15]
* 在对话框中发送用户请求后，不要立即为响应占位，直到报文返回才开始占位渲染 [新增于 it#18 20260419_7]
* 对话框中展示用户已发送请求和已接收响应的气泡<div>在左右各留空50px，避免展示过于紧凑 [新增于 it#20 20260419_9]
* 已经新建的会话，但没有任何消息，即使切换也不展示动画（仅一次） [新增于 it#21 20260421_1]
* 拖动上传时必须等待所有拖入项（文件和目录）全部遍历完成后再统一发起上传请求，不能逐个发起 [新增于 it#27 20260422_1]
* 未选择Agent或Agent不存在均不允许发送请求 [新增于 it#32 20260425_1]
* 点击确认，为该AgentID和Chat（会话ID）重新加载并渲染最后一条SSE请求时间之后的对话数据，加载完后Tips已加载完成 [新增于 it#43 20260428-11]
* 在居中输入框发送SSE请求时，在界面等待3s后再向后段发送请求，等待期间`整个页面`的垂直水平居中位置展示100x100px的转圈等待图标，发送成功后消失 [新增于 it#44 20260428-12]
* 如果当前界面正在等待发送，则立即终止待发送消息 [新增于 it#45 20260428-13]
* 如果已经发送请求 [新增于 it#45 20260428-13]
* 已经收到并渲染过任意响应，但SSE请求还没有结束，则立即中断SSE连接，并终止会话存储 [新增于 it#45 20260428-13]
* 已经发送请求，但没有收到并渲染过任何响应，则等待3s后中断SSE连接，并终止会话存储 [新增于 it#45 20260428-13]
* 终止关闭指定Agent和指定chat（会话ID）的转发请求需求：../proxy/iteration/20260427-9/REQUIREMENT.md [新增于 it#45 20260428-13]
* 如果没有渲染任何SSE响应（包括取消还没发送的请求），则自动补充最后一条响应消息为已取消，标记为已取消，更新时间（样式与正常的SSE响应一致） [新增于 it#45 20260428-13]
* 因为发送请求和发送取消是异步会有时间差，所有在发送终止请求时要等待3s，界面展示为转圈等待（同等待发送样式） [新增于 it#45 20260428-13]
* 异常响应为HTTP错误等无法完成请求的情况 [新增于 it#46 20260428-14]
* 如果无连接且最后响应不是完成或取消，则立即将页面状态切换为等待响应的样式（悬浮努力工作中，发送变为终止），并为该AgentID和Chat（会话ID）每5秒轮询重新加载并渲染最后一条SSE请求时间之后的对话数据（每次都要实时取最后一条），加载完后Tips已加载完成 [新增于 it#46 20260428-14]
* 轮询到数据立即渲染，并更新下次使用的最后一条SSE请求时间：../proxy/iteration/20260427-8/REQUIREMENT.md [新增于 it#46 20260428-14]
* 恢复会话时，空内容的SSE片段记录（如换行符、data: [DONE]终止标记）不应创建新的助手消息气泡，仅追加到当前助手消息的原始数据中用于终态检测 [新增于 it#46 20260428-14]
* 浏览器刷新/重进导致的本地连接中断不属于请求异常，不展示 `网络错误`，需保留未完成状态并在重新进入后自动轮询恢复 [新增于 it#46 20260428-14]
* 发送后立即刷新页面时，恢复逻辑不得重复渲染当前这次已发送请求；对已存在的本地请求应直接复用并继续恢复后续响应 [新增于 it#46 20260428-14]
* 所有Cron相关前端高频触发的数据刷新（如切换会话、切换Agent）需要防抖处理，避免短时间内重复请求 [新增于 it#54 20260428-9]
* 加载到SSE响应结束（请求完成）需要按正常的恢复逻辑切换到待输入样式 [新增于 it#56 20260430-2]
* 右侧面板独立渲染的cli+sub内容独立存储在消息的cliSubContent字段，不混入居中对话框的content，避免刷新后回显到居中 [新增于 it#62 20260430-8]
* 展开浮层需要有危险提示（10px红色小字），展示执行规则：仅允许本机请求执行，且包含rm的命令会被拦截 [新增于 it#65 20260501-2]
* 每个会话（Agent+Chat）的居中会话框最多展示2000条最新消息（包括请求加响应） [新增于 it#67 20260501-4]
* 每个会话（Agent+Chat）的右侧CLI子面板最多展示100条最新消息 [新增于 it#67 20260501-4]
* 在居中对话框所有的请求和SSE响应气泡右下角（气泡外，与气泡垂直右对齐）增加15x15px的复制图标 [新增于 it#73 20260502-5]
* 在居中对话框所有的请求（不要响起SSE）的气泡右下角（气泡外，与复制图标水平右对齐）增加15x15px的重试图标 [新增于 it#74 20260502-6]
* 点击重发图标后复制请求内容，自动发送HTTP请求，同时新增并刷新居中对话框的请求气泡 [新增于 it#74 20260502-6]
* 点击重发图标并发送请求成功后，在虚拟文件系统的Tips位置提示重试成功 [新增于 it#74 20260502-6]
* 如果消息正在发送状态，则禁止点击重试，并Tips消息正在发送中 [新增于 it#74 20260502-6]
* 使用当前会话ID作为CHAT_ID，并在后续任务作为转发/v1/chat/completions的chat使用 [新增于 it#75 20260502-7]
* 插件配置必须统一以key作为唯一标识，打开插件浮层时强制重新请求/api/plugins/meta回填最新meta，并保证保存后关闭重开及刷新页面都能稳定显示已填参数 [新增于 it#80 20260504-2]
* Bug修复：右上角备忘录明细中，已完成任务的“查看”按钮重复点击时，应恢复同一会话但不得重复追加已渲染的历史消息 [新增于 it#88 20260505-4]
* 点击查看时，保留备忘录会话的当前游标，再次使用/api/restore不要把已经恢复过的消息重新追加一遍 [新增于 it#88 20260505-4]
* 左上角点击备忘录明细并查看进行对话恢复时，如果需要恢复通过/api/restore没有任何数据返回，通常是因为当前保存的的最后恢复时间晚于该备忘录消息 [新增于 it#89 20260505-5]
* 在虚拟文件系统Tips位置提示：消息已恢复，删除会话可重建 [新增于 it#89 20260505-5]

### 对话框与输入
* 对话框中输入`@`触发可以使用键盘控制的菜单浮层。浮层的左下角紧贴@字符向右上展开，有2个选项：文件和技能，可用鼠标或键盘选择 [新增于 it#2 20260419_10]
* 选择文件的图标需要与对话框右侧小文件夹图标风格一致，相关需求：./iteration/20260419_6/REQUIREMENT.md [新增于 it#2 20260419_10]
* 气泡插入位置为@触发时的光标位置，需要在@触发时保存光标状态，选择完成后恢复光标再插入，避免因异步操作或焦点转移导致光标丢失 [新增于 it#2 20260419_10]
* 按ESC键或空格，即中断监听并取消浮层（包括@菜单、文件选择、技能选择所有层级），但保留@符号，焦点回到输入框 [新增于 it#2 20260419_10]
* 当前光标待输入位置：已输入ABC，选择文件/DEV/X，再输入DEF，则展示内容为ABC[FILE:X]DEF，提交内容为ABC[FILE:/DEV/X]DEF，删除气泡则为ABCDEF [新增于 it#2 20260419_10]
* 当前光标待输入位置：已输入ABC，选择技能X，再输入DEF，则展示和提交内容为ABC[SKILL:X]DEF，删除气泡则为ABCDEF [新增于 it#2 20260419_10]
* 模糊匹配时，路径拼接基于父目录而非输入值本身，例如：输入~/D匹配到DEV后应拼接为~/DEV而非~/D/DEV [新增于 it#2 20260419_10]
* 对话框中用户气泡需要使用Markdown展示，但不影响输入框和左侧边栏（Sidebar）的会话历史 [新增于 it#3 20260419_11]
* 对话框使用contenteditable实现，提取用户输入内容时需要递归遍历DOM节点，正确处理换行（BR元素和DIV/P块级元素转为\n），确保多行内容（如Markdown表格）的换行符不丢失 [新增于 it#3 20260419_11]
* 使用contenteditable输入框粘贴内容时必须剥离HTML格式只保留纯文本, 防止带格式内容撑破布局 [新增于 it#3 20260419_11]
* 虚拟文件系统的文件可以打开，在右侧对话框水平垂直都居中，宽度为700px，高度为700px，透明度0%的浮层中，以Markdown格式预览或编辑内容 [新增于 it#4 20260419_12]
* 预览与浮层和对话框使用同色系小色差区分对比，编辑无论在浅色或是还是深色模式都使用白底黑字 [新增于 it#4 20260419_12]
* 浮层需要与对话框有同色系小色差区分对比 [新增于 it#4 20260419_12]
* 对话框用户提问气泡保持当前色系，对比LOGO的颜色，饱和度降低20% [新增于 it#6 20260419_14]
* 最后（最新）的提问气泡在渲染响应报文前，边框以5px边缘锐化闪光，响应报文开始渲染后停止锐化闪光 [新增于 it#6 20260419_14]
* 用户提问气泡的边缘锐化闪光需求：./iteration/20260419_14/REQUIREMENT.md [新增于 it#7 20260419_15]
* 切换会话后，尚未发送的用户输入需要以会话纬度暂存，切换回后重新填充（包括@文件或技能的气泡） [新增于 it#7 20260419_15]
* @文件或技能需求：./iteration/20260419_10/REQUIREMENT.md [新增于 it#7 20260419_15]
* 不同会话的输入框、暂停键（或其他中断）以会话纬度相互独立，互不影响 [新增于 it#7 20260419_15]
* 在对话框中双击、拖拉、框选范围文字时，模糊查找在当前Agent工作目录（workspace）下是否存在该文件且可预览，如果可预览则展示气泡提示 [新增于 it#10 20260419_18]
* 在对话框中双击、拖拉、框选范围链接或HTML超链接（<a>）时，直接展示气泡提示 [新增于 it#10 20260419_18]
* 对话框中点击文件路径气泡时，应基于相对路径或是绝对路径解析并打开实际存在的文件，而不是错误拼接到无关目录后提示文件不存在 [新增于 it#10 20260419_18]
* 在对话框中双击文字时，自动框选双击范围内： [新增于 it#11 20260419_19]
* 浮层采用透明度50%，色系风格需要与对话框出现错误时的风格保持一致 [新增于 it#12 20260419_2]
* 在对话框发送按钮右侧（对话框外），添加15x15像素，透明度50%的小文件夹图标 [新增于 it#17 20260419_6]
* 对话框中输入文字（包括placeholder）需要水平居中对齐，尤其在单行时不要有错位感 [新增于 it#19 20260419_8]
* 用户开始输入或其他任何操作时，立即触发结束动画，然后开始执行操作 [新增于 it#21 20260421_1]
* 在设置中的`打开Agent目录`右侧增加一个新增图标，点击后弹出名称输入框，确认后创建Agent并立即刷新下拉列表 [新增于 it#23 20260421_3]
* 二次输入框内容：确定删除`名称`？ [新增于 it#24 20260421_4]
* 新建：显示浮层，输入名称，选择文件或目录 [新增于 it#26 20260421_6]
* 对话框输入栏右侧`打开Agent目录` [新增于 it#28 20260422_2]
* 会话与会话之间保持独立的Agent和模型，设置里的选择Agent和对话输入框左侧的`选择模型`绑定会话保存 [新增于 it#29 20260422_3]
* 对话输入框上展示高度为25px的悬浮框，居中对齐，展示当前正在使用的模型和Agent，字号需要自动适配 [新增于 it#29 20260422_3]
* 在对话输入框上提示当前Agent和模型的悬浮框右侧，添加一个1.5倍px大小的小太阳图标，用来打开设置页面，为当前会话切换Agent [新增于 it#30 20260422_4]
* 对话框中输入`@`触发文件，如果是相对路径则以当前Agent目录为根目录开始查找 [新增于 it#31 20260422_5]
* @需求：./iteration/20260419_10/REQUIREMENT.md [新增于 it#31 20260422_5]
* 开启蜂群后，对话框输入框后小太阳图标后追加小蜜蜂图标，同节奏闪烁，关闭蜂群则消失 [新增于 it#33 20260425_2]
* 从虚拟文件系统或实际操作系统的文件或目录拖入对话框输入区域后自动转换为系统绝对路径 [新增于 it#34 20260425_3]
* 展示样式和插入区域参考@文件后的小气泡需求：./iteration/20260419_10/REQUIREMENT.md [新增于 it#34 20260425_3]
* 在输入对话框发送按钮正上方同步展示Thinking滑动按钮，状态需要与设置中同步 [新增于 it#36 20260426_1]
* 在对话框之外，与小蜜蜂图标水平对齐，与发送按钮垂直且右侧对齐 [新增于 it#36 20260426_1]
* 在设置更新Thinking配置并保存后，才同时同步Thinking在对话框上的按钮 [新增于 it#36 20260426_1]
* 设置面板内的操作（切换Agent、修改开关）只更新面板内DOM，禁止修改全局状态变量或输入框UI，全局状态仅在保存成功或关闭面板时从当前会话Agent重新加载 [新增于 it#36 20260426_1]
* 上半部分占7/8，展示一个没有发送按钮的对话输入框，样式同居中对话框的输入框，占位符内容为`xxx的备忘录...`，字体大小为20px，高度为150px，其中xxx为当前Agent名称 [新增于 it#37 20260427_1]
* 需要支持虚拟文件系统或实际操作系统的文件或目录拖入对话框 ：./iteration/20260425_3/REQUIREMENT.md [新增于 it#37 20260427_1]
* 当文件或目录拖动至对话框输入区正确位置时，对话框需要圈绕光圈提示，直到正确拖入、移出或取消 [新增于 it#37 20260427_1]
* 使用同样的小气泡展示，包括光标和插入策略：./iteration/20260419_10/REQUIREMENT.md [新增于 it#37 20260427_1]
* 需要支持@文件和技能的能力，效果等同：./iteration/20260419_10/REQUIREMENT.md [新增于 it#37 20260427_1]
* 备忘录菜单浮层也需要响应键盘确认（回车）或取消（ESC），输入完毕或离开备忘录输入框时要自动收起 [新增于 it#37 20260427_1]
* 备忘录菜单浮层的左下角紧贴@备忘录字符向右上展开，与居中输入框的浮层相互独立 [新增于 it#37 20260427_1]
* 备忘录菜单浮层不同于居中的输入框浮层，需要向下展开 [新增于 it#37 20260427_1]
* 需要支持选择模型，使用30px同样的小地球图标，选择后的模型字体大小和选择框内的待选择模型字体大小均为8px，需要一次全部展示，样式同输入框小地球，需要可以看到勾选的选择后模型，选择时需要有淡入淡出效果 [新增于 it#37 20260427_1]
* 下半部分左侧左对齐是展示选择时间的控件，精确到分钟，样式为时间轮（小时为选择，分钟为输入），默认时间为当前时间，紧贴右侧为保存和取消按钮，保存样式为信封，取消样式为垃圾桶，与时间轮水平对齐但水平右对齐 [新增于 it#37 20260427_1]
* 点击取消时，重置输入框为`xxx的备忘录...`，重置时间为当前时间，重置模型为待选择 [新增于 it#37 20260427_1]
* 连同输入框的会话框在左右Sidebar之间居中 [新增于 it#37 20260427_1]
* 多个输入框共存时, 事件操作(气泡插入、拖入等)必须作用于触发事件的目标输入框, 占位符与输入内容的样式需独立控制 [新增于 it#37 20260427_1]
* contenteditable输入框: 粘贴时剥离HTML只保留纯文本, 设置明确的max-height配合overflow-y:auto防止撑破布局 [新增于 it#37 20260427_1]
* 当访问的Host不是localhost或127.0.0.1（本地访问时）禁止向居中对话框和右上的备忘录拖入文件或目录，但允许从虚拟文件系统拖入 [新增于 it#38 20260427_2]
* 拖入对话框需求：./iteration/20260425_3/REQUIREMENT.md [新增于 it#38 20260427_2]
* 禁止拖入时输入框光圈变为红色，并tips原因（居中输入框和备忘录输入框都需要） [新增于 it#38 20260427_2]
* 居中对话框的tips浮层向上展示 [新增于 it#38 20260427_2]
* 提交前先渐变展开浮层，让用户确认除输入框内容之外的配置，再次点击浮层的保存后执行 [新增于 it#40 20260427_4]
* 如果当前未选择模型或未在输入框输入非空内容，使用Tips提示，同时不要展示该浮层，点击非浮层区域自动收起，再次点击保存后再次以最新数据展开 [新增于 it#40 20260427_4]
* 在设置中切换Agent后，如果右上的备忘录有任何输入（包括备忘录输入框、备忘录选择模型、备忘录日历、备忘录时间、备忘录思考模式）在点击设置保存后，重置所有备忘录未保存的输入 [新增于 it#41 20260428-1]
* 如果备忘录有任何输入，在设置选择Agent上发展示Tips提示，红字，字号10px，切换Agent并保存会重置未保存备忘录，不保存不重置 [新增于 it#41 20260428-1]
* 居中输入框在收到SSE响应并渲染时需要增加收到响应的时间日期和状态，以yyyy-MM-dd hh:mm:ss的格式展示在气泡的右下方，右对齐 [新增于 it#42 20260428-10]
* 居中输入框右侧文件夹图标上增加一个重新加载的图标，点击图标展开浮层，提示用户是否要重新加载会话 [新增于 it#43 20260428-11]
* 点击居中对话框的输入框在等待SSE响应期间的终止会话图标 [新增于 it#45 20260428-13]
* 对话框状态在一次到多次到5秒轮询并确认收到完成状态的SSE响应后按原逻辑渲染并恢复为待发送状态，否则持续保持等待响应的样式 [新增于 it#46 20260428-14]
* 自动轮询恢复必须完整拉取本轮SSE的全部增量和终态；收到终态后立即恢复待输入，不允许出现只加载部分内容或结束后仍停留在等待状态 [新增于 it#46 20260428-14]
* 通过虚拟文件系统拖入或@选择文件/目录时，气泡需展示完整绝对路径，而非仅文件名 [新增于 it#49 20260428-4]
* 在二次确认保存备忘录成功，在重置备忘录输入框、备忘录模型、备忘录思考模式后立即切换到备忘录元数据列表 [新增于 it#51 20260428-6]
* 如果任务元数据的输入框中无内容、空内容（trim后无内容）或仅为默认值时，需要立即以当前AgentId的占位符来刷新备忘录输入框的默认值并恢复展示输入框为默认值 [新增于 it#52 20260428-7]
* 居中对话框立即使用轮询来恢复对话内容（因为首次恢复时没有时间，使用备忘录明细的启动时间）、样式与普通会话一致：./iteration/20260428-14/REQUIREMENT.md [新增于 it#56 20260430-2]
* 如果用户的光标/焦点没有停留在备忘录元数据、备忘录明细、备忘录日历，每30秒刷新一次最新数据 [新增于 it#58 20260430-4]
* 居中会话框展示SSE响应时如果报文的属性biz=cli且workflow=sub则不要展示在居中对话框，重定向到右侧边栏第二排左格子独立渲染 [新增于 it#62 20260430-8]
* 点击右侧CMD小图标，居中展开用户输入CMD命令的浮层，宽度500px，高度110px，输入框内字体字号10px，输入指令后需要二次确认后执行 [新增于 it#65 20260501-2]
* 输入框高度调整为不低于35px，并适当增加上下内边距，要求在不遮挡Tip文案和底部按钮区域的前提下，使输入框视觉占比明显提升 [新增于 it#65 20260501-2]
* 切换到二次确认的浮层和二次确认的浮层返回到输入框保持用户无切换感觉，但需要高亮和闪烁危险提示 [新增于 it#65 20260501-2]
* 二次确认的提示文案与输入框危险提示文案使用同样式和位置 [新增于 it#65 20260501-2]
* 文案与输入框保持5px边距，水平右对齐 [新增于 it#65 20260501-2]
* 点击黏贴小图标，自动复制气泡内的SSE响应至粘贴板，展开点击CMD小图标的浮层，并复制内容到输入框 [新增于 it#66 20260501-3]
* CMD面板的遮罩层和浮层仅覆盖居中对话框区域（排除左侧Sidebar和右侧Sidebar），面板在该区域内居中展示 [新增于 it#68 20260501-5]
* 已经展示居中对话框中的图片不能因为后端链接下线而无法展示，需要进行浏览器缓存 [新增于 it#72 20260502-4]
* 居中对话框中已成功展示过的远程图片，首次渲染后必须写入浏览器本地持久缓存，后续即使源图片链接失效或返回404，页面刷新后仍需优先从本地缓存恢复显示 [新增于 it#72 20260502-4]
* 在居中对话框输入框或右上角备忘录输入框进行系统粘贴板黏贴操作时，如果黏贴为图片数据，则使用/api/edit为图片在当前Agent的tmp目录下创建随机命名的图片，并在输入框中复制气泡样式的生成图片后的系统绝对路径 [新增于 it#76 20260502-8]
* 气泡样式需要与@文件时一致： [新增于 it#76 20260502-8]
* 输入框里显示：[FILE:图片文件名] [新增于 it#76 20260502-8]
* 右上角备忘录的气泡最大宽度不超过输入框，不要把输入框横向撑破 [新增于 it#76 20260502-8]
* 点击扇形菜单的按钮，打开居中对话框浮层 [新增于 it#80 20260504-2]
* 输入框一行一个，标题左对齐，参数名和参数输入框放一行 [新增于 it#80 20260504-2]
* 点击设置的浮层展示位置需要与扇形菜单的位置对齐（居中对话框水平垂直居中） [新增于 it#81 20260504-3]
* 点击CLI子面板执行命令的浮层展示位置需要与扇形菜单的位置对齐（居中对话框水平垂直居中） [新增于 it#82 20260504-4]
* 点击扇形菜单展开后的插件浮层中的日志按钮，浮层中间的参数输入框变为展示SSE流响应，日志按钮变为返回，点击后关闭读取日志的SSE流并转换为输入框 [新增于 it#83 20260504-5]
* 从本地文件系统拖入到居中输入框或左上角备忘录输入框的文件或文件夹（含图片），先上传至当前Agent的tmp目录下 [新增于 it#86 20260505-2]
* 当会话数量已达上限时，提示内容（浮层）需要相对于居中对话框水平垂直居中，参考扇形菜单 [新增于 it#87 20260505-3]
* 如果居中输入框没有选择模型，使用覆层提醒：点击输入框左侧小地球选择模型 [新增于 it#91 20260505-7]
* 参考样式：会话数量已达上限时的提示样式（浮层，相对于居中对话框水平垂直居中） [新增于 it#91 20260505-7]

### 侧边栏与会话
* 点击插件展开的具体插件浮层，根据/api/plugins/meta中scope命令结果决定复用当前会话、选择Agent、模型列表和思考方式的可见性 [新增于 it#105 20260517-10]
* scope属性不存在时所有配置展示并可用，scope为空列表([])表示不支持以上所有配置 [新增于 it#105 20260517-10]
* scope为有限值（如["reuse"]）时只支持该配置 [新增于 it#105 20260517-10]
* 右侧Sidebar收起时，原本展示在虚拟文件系统上的Tips内容展示到居中对话框左下角 [新增于 it#100 20260517-5]
* 左侧虚拟文件系统的Tips位置根据左侧Sidebar展开/收起状态动态切换：展开时显示在虚拟文件系统底部不遮挡设置，收起时显示在居中对话框下方空白区 [新增于 it#100 20260517-5]
* 在左侧边栏（Sidebar）下方，设置上方加入高度为250px的虚拟文件系统，用来展示当前Agent工作目录的文件或目录 [新增于 it#4 20260419_12]
* 不同会话的SSE响应流相互隔离，切换会话时仅更新当前活跃会话的DOM，后台会话的SSE数据写入内存但不操作DOM，切换回时从数据重新渲染 [新增于 it#7 20260419_15]
* 切换会话时，需要先清除当前UI状态（努力工作中、暂停键、边缘锐化闪光），再切换到目标会话并根据目标会话状态重新设置UI [新增于 it#7 20260419_15]
* 左侧边栏（Sidebar）的会话数量最多不超过10个，超过10个则提示删除后才可以新建会话 [新增于 it#12 20260419_2]
* 浮层文字：会话数量已达上限（10个） [新增于 it#12 20260419_2]
* 左侧边栏（Sidebar）最上方Logo（DeepRight）的左侧的SVG配合心跳联动 [新增于 it#13 20260419_20]
* 左侧边栏（Sidebar）最上方Logo（DeepRight）的右侧增加'收起菜单'按钮，用来隐藏左侧边栏 [新增于 it#14 20260419_3]
* 收起菜单相对于左侧边栏右对齐 [新增于 it#14 20260419_3]
* 隐藏后的左侧边栏位置依然要展示'展开菜单'按钮，用来展开左侧边栏 [新增于 it#14 20260419_3]
* 收起菜单相对于左侧边栏左对齐 [新增于 it#14 20260419_3]
* 按钮色系风格需要与侧边栏风格保持一致 [新增于 it#14 20260419_3]
* 左侧边栏（Sidebar）删除会话时图标修改为10x10像素，透明度50%的小垃圾桶 [新增于 it#15 20260419_4]
* 小垃圾桶色系风格需要与侧边栏风格保持一致 [新增于 it#15 20260419_4]
* 二次提醒时如果切换会话，则自动收回 [新增于 it#15 20260419_4]
* 保持会话框风格：简洁、干练 [新增于 it#18 20260419_7]
* 首次初始化新会话时展示参考动画（观星者）10秒，然后渐变回现有的操作界面 [新增于 it#21 20260421_1]
* 切换会话时需要立即终止当前动画（cancelAnimationFrame并清除状态），避免旧动画引用已移除的DOM导致卡住 [新增于 it#21 20260421_1]
* 重新打开或切换会话后自动使用最后选择Agent和模型，同时刷新悬浮层和虚拟文件系统 [新增于 it#29 20260422_3]
* 会话提交执行时也需要使用当前会话绑定的Agent和模型 [新增于 it#29 20260422_3]
* 会话绑定的状态（Agent、模型等）需要在新建会话时立即绑定当前值，并持久化到localStorage随saveChats一起存取，不能仅存在内存中 [新增于 it#29 20260422_3]
* 在设置中删除Agent后，如果已经创建的会话使用了该Agent则提示选择Agent [新增于 it#32 20260425_1]
* 删除后立即检查当前会话 [新增于 it#32 20260425_1]
* 切换会话后检查选择会话 [新增于 it#32 20260425_1]
* 提示框未选择时切换会话，则收起提示框再次后检查选择会话 [新增于 it#32 20260425_1]
* 如果选择会话的Agent存在则不再提示，如果不存在则重新提示 [新增于 it#32 20260425_1]
* 提示框收起和提示时都是为当前会话选择Agent，需要注意实时性 [新增于 it#32 20260425_1]
* 蜂群开关变更后立即检查当前会话的Agent以更新小蜜蜂图标 [新增于 it#33 20260425_2]
* Agent绑定关系变更时（新建会话、切换会话、浮层选择Agent）需要异步检查该Agent的config.json状态并更新蜜蜂图标，不能仅在设置面板操作时更新 [新增于 it#33 20260425_2]
* Agent绑定关系变更时（新建会话、切换会话、浮层选择Agent）需要异步检查该Agent的config.json状态并更新Thinking图标，不能仅在设置面板操作时更新 [新增于 it#35 20260425_4]
* 在会话框按钮更新Thinking配置后，需要保存至当前Agent配置 [新增于 it#36 20260426_1]
* 在右侧开辟右对齐Right Sidebar，宽为左侧Sidebar的2倍 [新增于 it#37 20260427_1]
* 左侧Sidebar水平分割成对称的上下两部分 [新增于 it#37 20260427_1]
* 设置面板内的操作只更新面板内DOM, 全局状态仅在保存或关闭面板时从当前会话Agent重新加载 [新增于 it#37 20260427_1]
* 浮层宽度为左侧Sidebar宽度250px，右侧对齐，浮层未保存期间任何备忘录配置的改动都立即收起浮层 [新增于 it#40 20260427_4]
* 浮层展示排序：模型（如果未选择则默认使用当前居中会话框使用的模型）、周期、思考方式（Thinking、Auto）、开始时间 [新增于 it#40 20260427_4]
* 浮层标题（标题居中，不需要内容）：确定重载当前会话？ [新增于 it#43 20260428-11]
* 恢复会话必须幂等：不得重复渲染当前页面已展示过的本轮 `Q/A`，恢复边界以最后一次实际发送时间为准，不能使用本地预渲染时间 [新增于 it#43 20260428-11]
* 转圈等待效果是与当前会话绑定的，切换会话时需要实时判断，如果有等待发送则立即展示等待样式，直到计时结束 [新增于 it#44 20260428-12]
* 转圈等待的时间（3s）一旦开始就要固定开始计时的结束时间，即使切换会话也不应该重新计时 [新增于 it#44 20260428-12]
* 当用户首次进入或切换会话后，检查当前会话是否有SSE连接并正在等待响应 [新增于 it#46 20260428-14]
* 如果无连接则检查当前Agent和Chat（会话ID）在页面渲染的最后非异常响应内容是否标记为完成或取消 [新增于 it#46 20260428-14]
* 如果在轮询点了终止会话图标，则按终止会话的流程处理，注意轮询时并没有会话的SSE连接，不要关闭 [新增于 it#46 20260428-14]
* 终止会话需求：./iteration/20260428-13/REQUIREMENT.md [新增于 it#46 20260428-14]
* 备忘录列表使用当前会话的AgentID及日历选择日期来获取明细，如果更改了Agent（如调整了设置）或日期（如点击日历其他日期）需要立即刷新 [新增于 it#50 20260428-5]
* 首次打开、切换会话、设置中切换Agent、保存成功备忘录后，需要立即以日历指定日期和当前会话AgentId刷新备忘录明细列表，如果同时正在展示任务元数据列表那么也需要立即刷新 [新增于 it#52 20260428-7]
* 小垃圾桶风格参考会话列表：./iteration/20260419_4/REQUIREMENT.md [新增于 it#54 20260428-9]
* 点击小放大镜图标，需要展开浮层二次确认是否展开会话，展示除任备忘录内容外的任务明细， [新增于 it#56 20260430-2]
* 展开会话前需要确认会话数量：./iteration/20260419_2/REQUIREMENT.md:20 [新增于 it#56 20260430-2]
* 展开会话前需要确认对应Agent、Chat（会话ID）的会话不存在（一个Agent、Chat）只能有一个展示会话，同时跳转到已经创建过的会话 [新增于 it#56 20260430-2]
* 如果会话不存在，则自动创建对应Agent、Chat（会话ID）、模型和思考模式的会话，并切换到该会话 [新增于 it#56 20260430-2]
* 左侧sidebar中会话展示内容任务明细的内容 [新增于 it#56 20260430-2]
* 删除Agent时需要同时删除该Agent的备忘录元数据和备忘录明细，同时立即刷左上的备忘录元数据列表和备忘录明细列表、同时删除左侧Sidebar中该Agent的其他会话（Chat) [新增于 it#58 20260430-4]
* 当前会话没有绑定Agent需要选择Agent时的浮层下拉框仅有一个可选项时，需要同时监听click事件作为补充，确保单选项场景下选择后也能正常绑定Agent并关闭浮层 [新增于 it#60 20260430-6]
* 居中会话框展示SSE响应时需要有打字机效果 [新增于 it#61 20260430-7]
* 将右侧边栏第二排左格子合并为一个（宽度加长） [新增于 it#62 20260430-8]
* 右侧面板独立渲染的cli+sub内容与当前Agent+Chat绑定，切换会话或刷新页面后需要恢复展示 [新增于 it#62 20260430-8]
* 右侧面板独立渲染的cli+sub单个气泡高度为70px，字体为10px，和居中会话框相同的自动滚动效果，不要超出外部容器高度 [新增于 it#62 20260430-8]
* 在右侧Sidebar第一栏和第二栏之间增加高度为30px的分割带，让2个部分看起来有一定分区 [新增于 it#63 20260430-9]
* 轮询恢复会话数据时，解析SSE原始流中的biz=cli且workflow=sub内容，分离到cliSubContent字段独立存储，恢复完成后同步刷新右侧CLI子任务面板 [新增于 it#63 20260430-9]
* 右侧CLI子任务面板实时渲染SSE内容时，必须校验当前SSE流的会话ID是否为活跃会话，非活跃会话的cli+sub内容只存储不渲染，避免不同会话的内容串显 [新增于 it#63 20260430-9]
* 右侧Sidebar第一排的高度增加到绝对值220px、第二排的高度增加到绝对值250px [新增于 it#64 20260501-1]
* 右侧Sidebar第二排仅CLI子任务列表容器底部固定5px占位留白，确保最后一个气泡不再紧贴面板底边 [新增于 it#64 20260501-1]
* 右侧Sidebar第二排仅CLI子任务气泡使用霓虹荧光绿色而气泡外层保持整体原有色调，如果CLI子任务面板有新SSE响应刷新则高频闪动3秒 [新增于 it#64 20260501-1]
* 前端弹层层级规范：左侧Sidebar、居中操作区、右侧Sidebar的overlay必须分域管理，禁止跨区域全屏遮挡；所有透明层在非激活态必须pointer-events:none，避免误拦截底层操作 [新增于 it#68 20260501-5]
* CMD面板的全屏遮罩层不能覆盖右侧Sidebar区域，避免遮罩层拦截CLI子任务面板的复制/终止按钮点击事件 [新增于 it#68 20260501-5]
* 在左上角创建备忘录的选择模型按钮左侧（紧贴，水平左对齐）增加一个无标题的复选框，选中后在虚拟文件系统的Tips位置提示该任务会复用当前会话（红字） [新增于 it#75 20260502-7]
* 如果开启了复选框，在保存浮层时增加当前提示（红字）：继续使用当前会话 [新增于 it#75 20260502-7]
* 如果备忘录勾选复用当前会话，则任务完成后点击放大镜查看会话时，无论该会话是否已在左侧存在，都必须基于本次任务明细的执行时间立即恢复并展示本次备忘录新增的问答内容，禁止仅切换到旧会话而不加载本次任务内容 [新增于 it#75 20260502-7]
* 查看复用会话的备忘录结果时，已有会话不能只做switchChat，必须强制触发一次按该任务execTime起点的restore/reload。验收标准：点击放大镜后，页面能看到这次备忘录新增的Q/A，而不是只看到历史旧内容 [新增于 it#75 20260502-7]
* 左上新会话按钮右侧水平位置新增一个即时通讯的图标（紧贴右侧），原新会话按钮自适应缩进 [新增于 it#79 20260504-1]
* 扇形展开后，被遮挡背景（新会话按钮、会话列表、最近）做模糊处理，收起后恢复 [新增于 it#79 20260504-1]
* 第一部分（展示在插件声明下3px）：是否复用会话（ChatID）、选择Agent、选择模型、思考模式 [新增于 it#80 20260504-2]
* 是否复用会话勾选后需要在右侧虚拟文件夹Tips提示，提示内容和样式同右上角备忘录的复用会话 [新增于 it#80 20260504-2]
* 所有固定定位遮罩层、确认框、弹窗必须统一管理层级、显示状态和pointer-events，按左侧Sidebar、居中会话区、右侧Sidebar分域隔离，禁止未激活浮层跨区域遮挡或拦截其他可操作控件点击 [新增于 it#91 20260505-7]

### 文件系统
* 文件/目录或技能的气泡在左上角有一个10x10像素，透明度50%的小垃圾桶，点击后删除气泡 [新增于 it#2 20260419_10]
* 保存：将修改后的内容保存至指定文件 [新增于 it#4 20260419_12]
* 虚拟文件系统的目录可以点击，进入子孙目录 [新增于 it#4 20260419_12]
* 选择Agent后虚拟文件系统需要常驻显示，每10秒刷新一次`当前`展示的路径，仅在有变化时刷新内容 [新增于 it#4 20260419_12]
* 未选择Agent时（如刚初始化）时则不显示虚拟文件系统，但一旦选择并确定则立即显示 [新增于 it#4 20260419_12]
* 虚拟文件系统的文件或目录在水平对齐最后侧展示一个15x15像素的小垃圾桶，点击垃圾桶则删除对应文件或目录 [新增于 it#5 20260419_13]
* 小垃圾桶风格参考：./iteration/20260419_4/REQUIREMENT.md [新增于 it#5 20260419_13]
* 虚拟文件系统的文件预览方式按文件类型（后缀名）区分： [新增于 it#8 20260419_16]
* 如果范围文字是一个文件相对路径链接，则先在Agent工作目录（workspace）查找文件名，不存在使用绝对路径查找 [新增于 it#10 20260419_18]
* REQUIREMENT.md，则在当前Agent工作目录下模糊查找 [新增于 it#10 20260419_18]
* 无法无法匹配任何文件，则不做任何提示 [新增于 it#10 20260419_18]
* 链接或超链接：用文件编辑器浮层的iframe来预览链接，而不是打开新页面 [新增于 it#10 20260419_18]
* 查找明确的单一路径，如/a/b/c.md或./iteration/20260510-1/wiki_md/index.md、./iteration/20260510-1/wiki_md/index.md或单个像文件名的内容，如REQUIREMENT.md [新增于 it#11 20260419_19]
* 设置中`选择 Agent`下拉框的右侧，添加30x30像素，透明度30%的小文件夹图标 [新增于 it#16 20260419_5]
* 点击图标，调用/api/folder?agentId=xxx打开对应目录 [新增于 it#16 20260419_5]
* 小文件夹色系风格需要与设置风格保持一致 [新增于 it#16 20260419_5]
* 小文件夹图标参考：./iteration/20260419_5/REQUIREMENT.md [新增于 it#17 20260419_6]
* 点击图标，调用/api/folder?agentId=xxx打开当前Agent对应目录 [新增于 it#17 20260419_6]
* 虚拟文件系统的返回上级图标的水平位置，紧贴右侧靠拢展示2个按钮：新建和打开 [新增于 it#26 20260421_6]
* 打开：打开当前相对于Agent的workspace的目录 [新增于 it#26 20260421_6]
* 虚拟文件系统的新建和打开的右侧添加一个新图标：上传 [新增于 it#27 20260422_1]
* 点击上传展示浮层，选择文件还是目录，需要支持多个文件/目录同时拖动选入 [新增于 it#27 20260422_1]
* 按钮文案：文件 目录 [新增于 it#27 20260422_1]
* 上传目录需要保持相对目录结构，例如上传xxx/yyy目录，那么yyy目录及其子孙目录结构都要保留 [新增于 it#27 20260422_1]
* 上传文件图标风格与新建、打开一致，外围2px需要锐化发光和中等闪烁 [新增于 it#27 20260422_1]
* 上传时在虚拟文件系统中`返回上级`的右侧展示上传进度图标直到完成后消失，点击上传进度图标则立即取消上传 [新增于 it#27 20260422_1]
* 上传后在虚拟文件系统最下方展开浮层 [新增于 it#27 20260422_1]
* 上传成功则内容为上传后相对Agent工作目录的上传`目录`，高30px，5秒后自动收起 [新增于 it#27 20260422_1]
* 上传目录时，文件的相对路径不能依赖浏览器的文件名参数传递，需要通过独立的表单字段传递完整相对路径 [新增于 it#27 20260422_1]
* 虚拟文件系统的`打开目录` [新增于 it#28 20260422_2]
* 设置中选择Agent的`打开Agent目录` [新增于 it#28 20260422_2]
* 需要支持同时（一次性）拖动单个或多个文件或目录，转换为多个系统绝对路径 [新增于 it#34 20260425_3]
* 虚拟文件系统需要支持一次选择拖拉多个文件或目录，在点击或选择任一文件或文件夹时提示如何选择多个 [新增于 it#34 20260425_3]
* 在备忘录时间空间与取消（小垃圾桶图标）之间新增一个循环小图标，点击向下展开下拉框浮层，有三个选项：仅一次、工作日、自然日、每小时、每15分钟、每30分钟。默认选择（即使没打开）选择仅一次 [新增于 it#39 20260427_3]
* 与文件夹图标垂直对齐 [新增于 it#43 20260428-11]
* 虚拟文件系统中小垃圾桶图标后增加一个下载图标，提供文件和目录的下载 [新增于 it#47 20260428-2]
* 前端通过JS触发文件下载时，必须使用隐藏iframe方式（设置iframe.src为下载URL），不能使用动态创建<a>标签.click()或window.open，因为浏览器安全策略会拦截非直接用户手势触发的下载和弹窗 [新增于 it#47 20260428-2]
* 虚拟文件系统拖入需求：./iteration/20260425_3/REQUIREMENT.md [新增于 it#49 20260428-4]
* 通过操作系统拖入文件时，因浏览器安全策略无法获取绝对路径，需在插入后提示"浏览器安全策略仅展示文件名，建议使用虚拟文件系统" [新增于 it#49 20260428-4]
* 鼠标悬停在备忘录明细列表明细时，在头部展示小垃圾桶图标，8x8像素，透明度50%的小垃圾桶 [新增于 it#54 20260428-9]
* 点击小垃圾桶图标，需要展开浮层二次确认是否忽略任务，展示除任备忘录内容外的任务明细， [新增于 it#54 20260428-9]
* 仅状态为待执行时展示小垃圾桶图标，即使不需要展示小垃圾桶时依旧要保留每列对齐 [新增于 it#54 20260428-9]
* 放大镜图标样式和展示时机同小垃圾桶：./iteration/20260428-9/REQUIREMENT.md:6 [新增于 it#56 20260430-2]
* 仅状态为已完成时展示小放大镜图标，即使不需要展示小放大镜时依旧要保留每列对齐（包括小垃圾桶和小放大镜同时存在布局） [新增于 it#56 20260430-2]
* 右上角备忘录明细的小垃圾桶和小放大镜图标方法1.5倍 [新增于 it#57 20260430-3]
* 备忘录明细列表的操作图标（垃圾桶/放大镜）为互斥关系，合并为同一个图标位，非对应状态的行仅保留单个占位对齐，避免双占位产生多余空白 [新增于 it#57 20260430-3]
* 提示位置在左侧虚拟文件系统的Tips位置，红字提示 [新增于 it#65 20260501-2]
* 复制时在左侧虚拟文件系统的Tips位置，红字提示已复制到粘贴板 [新增于 it#66 20260501-3]
* 执行成功后在左侧虚拟文件系统的Tips位置，红字提示已终止 [新增于 it#68 20260501-5]
* 修改后的图片不要覆盖原图片，在原图目录下新建原图片名加时间戳的新图片 [新增于 it#69 20260502-1]
* 编辑并保存成功后将图片在文件系统的绝对路径复制到系统粘贴板，然后在虚拟文件系统的Tips位置提示路径已保存至黏贴板 [新增于 it#69 20260502-1]
* 点击复制图标后复制气泡内容到系统粘贴板，同时在虚拟文件系统的Tips位置提示黏贴成功 [新增于 it#73 20260502-5]
* 超长文件名会省略显示 [新增于 it#76 20260502-8]
* 启动失败则在虚拟文件系统提示Tips启动失败，并延迟2秒后切换到日志查看（需要有动画切换渐变） [新增于 it#84 20260504-6]
* 然后使用上传后的文件系统绝对路径作为气泡内容（需要包括原逻辑的[FILE]前缀） [新增于 it#86 20260505-2]
* 如果是目录，需要上传后保持目录结构，并将目录的文件系统路径作为气泡内容 [新增于 it#86 20260505-2]
* 虚拟文件系统上传目录：./iteration/20260422_1/REQUIREMENT.md [新增于 it#86 20260505-2]
* 虚拟文件系统上传需求： [新增于 it#86 20260505-2]

### 模型选择
* SVG颜色变更使用CSS filter（如hue-rotate）实现，不要用class选择器覆盖内联stroke属性，因为内联属性优先级更高 [新增于 it#13 20260419_20]
* contenteditable换行产生的子元素（div/p）需要继承居中对齐，使用通配符选择器强制所有子元素text-align:center [新增于 it#19 20260419_8]
* 在设置中选择Agent下拉框中Agent名字后增加小蜜蜂图标，点击后展开填写蜂群（swarm）配置 [新增于 it#33 20260425_2]
* 在设置中选择Agent后，在紧贴下拉框左侧展开一个Checkbox框，在Thinking和Auto之间切换 [新增于 it#35 20260425_4]
* 上半部分占2/3，居中展示可翻阅的日历，靠上对齐，以当前深色或浅色系标记今天，以淡红色标记点击选择的日期 [新增于 it#37 20260427_1]
* 需要支持选择Thinking滑动按钮（从右上对齐改为右下对齐）：./iteration/20260426_1/REQUIREMENT.md [新增于 it#37 20260427_1]
* 选择小时时需要有淡入淡出效果 [新增于 it#37 20260427_1]
* 复用其他组件的CSS class时, 必须检查其所有样式属性(包括布局约束和视觉样式), 确认不会与当前组件冲突, 必要时用更高优先级选择器隔离 [新增于 it#37 20260427_1]
* 选择后在时间轮水平左侧展示备选项 [新增于 it#39 20260427_3]
* 周期 精确到小时的时间点 模型类型 思考模式 [新增于 it#48 20260428-3]
* 右上角备忘录选择周期的展示调整： [新增于 it#55 20260430-1]
* 下方操作栏（周期标签、时间选择、图标按钮）间距紧凑化，禁止换行（flex-wrap:nowrap），gap和padding缩小 [新增于 it#55 20260430-1]
* 选择周期后展示的标签文字（如"每30分钟"）字号为12px，不换行（white-space:nowrap） [新增于 it#55 20260430-1]
* 选择后onchange不触发，浏览器会认为值未变化 [新增于 it#60 20260430-6]
* 设置中的模型与密钥数据不在存储在页面，改为使用/api/token保存或获取 [新增于 it#77 20260503-1]
* 选择模型后下拉框需要勾选后收起 [新增于 it#80 20260504-2]

### UI样式与浮层
* 所有浮层都需要先关闭当前打开的浮层后再打开自己，不允许重叠 [新增于 it#104 20260517-9]
* 左上角插件按钮打开的浮层透明度调整为与设置浮层一致 [新增于 it#102 20260517-7]
* 设置中打开Agent目录和对话输入框打开Agent目录的浮层调整为居中对话框水平垂直居中 [新增于 it#101 20260517-6]
* 设置面板中"打开Agent目录"点击后应先关闭设置浮层，再展示居中的目录确认浮层 [新增于 it#101 20260517-6]
* 打开Agent目录浮层时背景模糊，点击取消或任意其他位置则关闭 [新增于 it#101 20260517-6]
* 如果没有任何技能，则不展示技能菜单，每次实时判断 [新增于 it#2 20260419_10]
* 浮层右下展示2个按钮：保存、取消 [新增于 it#4 20260419_12]
* 取消：关闭浮层，点击ESC也可以关闭覆层 [新增于 it#4 20260419_12]
* 所有删除前都需要进行二次确认浮层，参考：./iteration/20260419_3/REQUIREMENT.md [新增于 it#5 20260419_13]
* 保持浮层的方式，窗口大小自适应 [新增于 it#8 20260419_16]
* 气泡风格：./iteration/20260419_10/REQUIREMENT.md [新增于 it#10 20260419_18]
* 提示采用对话栏水平和垂直居中宽250px,高100px的浮层，文字内容在覆层内垂直水平居中 [新增于 it#12 20260419_2]
* 删除需要二次提醒，风格为浮层，参考：./iteration/20260419_3/REQUIREMENT.md [新增于 it#15 20260419_4]
* 打开前需要二次提醒，风格为浮层，参考：./iteration/20260419_3/REQUIREMENT.md [新增于 it#17 20260419_6]
* 浅色背景使用原色系背景，蓝色星辰 [新增于 it#21 20260421_1]
* 深色背景使用原色系背景，白色星辰 [新增于 it#21 20260421_1]
* 隐藏的file input需要放在body级别，不能放在overflow:hidden的容器内；programmatic click必须在用户手势的同步调用栈内触发，不能在异步操作（如关闭浮层）之后 [新增于 it#27 20260422_1]
* 小太阳图标需要与当前背景色（浅色或深色）有撞色效果，快速闪动，提示用户可点击 [新增于 it#30 20260422_4]
* 提示方式同下方浮层：./iteration/20260422_1/REQUIREMENT.md [新增于 it#34 20260425_3]
* 浮层类组件(下拉框、菜单等): 父容器链不能有overflow:hidden, 宽度不能超出所在容器, 失焦收起时需检查焦点是否转移到浮层内部子元素 [新增于 it#37 20260427_1]
* 备忘录的tips浮层向下展示 [新增于 it#38 20260427_2]
* 循环小图标需要与当前背景色（浅色或深色）有撞色效果，快速闪动，提示用户可点击，同小太阳需求：./iteration/20260422_4/REQUIREMENT.md [新增于 it#39 20260427_3]
* 点击确定则重新加载，点击取消或其他任何非浮层位置则自动收起 [新增于 it#43 20260428-11]
* 浮层宽度：150px [新增于 it#43 20260428-11]
* 努力工作中的浮层，需要在所有取消完成（包括3s等待）后才收起 [新增于 it#45 20260428-13]
* 取消等待3s的浮层样式同待发送时，但颜色变为淡红色 [新增于 it#45 20260428-13]
* 鼠标悬浮在列表上向左下展开浮层，内容为备忘录内容，移开则自动收起 [新增于 it#48 20260428-3]
* 鼠标悬浮在列表上向左下展开浮层，内容为备忘录元数据，移开则自动收起 [新增于 it#50 20260428-5]
* 浮层风格参考：./iteration/20260427_4/REQUIREMENT.md [新增于 it#56 20260430-2]
* 备忘录元数据列表展示在向下展开的浮层上，先展示Agent名称，然后换行符后再展示任务内容 [新增于 it#59 20260430-5]
* 备忘录明细列表展示在现有浮层上，增加一行 [新增于 it#59 20260430-5]
* 气泡高度自适应内容，不设固定高度限制 [新增于 it#63 20260430-9]
* 气泡字体12px，代码块字体继承一致 [新增于 it#63 20260430-9]
* 气泡之间gap:16px [新增于 it#63 20260430-9]
* CLI子任务气泡保留科技荧光绿色外，其余背景统一使用浅色模式和深色模式的通用背景色 [新增于 it#64 20260501-1]
* 右侧CLI子面板的每一条SSE响应都必须独立渲染为单独气泡，在鼠标悬停在气泡左上角展示一个5x5像素的闪光黏贴小图标，禁止多个响应共用一个外层复制图标 [新增于 it#66 20260501-3]
* CLI子面板气泡需求：/20260430-8/REQUIREMENT.md [新增于 it#66 20260501-3]
* 悬停的判断区域为独立气泡而非内部的文字 [新增于 it#66 20260501-3]
* 独立气泡间间隔3px [新增于 it#66 20260501-3]
* 黏贴到粘贴板失败也不要阻止打开浮层 [新增于 it#66 20260501-3]
* 每条SSE响应气泡的黏贴小图标和黏贴内容必须一一对应、相互独立 [新增于 it#66 20260501-3]
* 点击某条气泡的小图标时，只允许复制当前气泡对应的SSE响应内容 [新增于 it#66 20260501-3]
* 不允许复制相邻气泡内容 [新增于 it#66 20260501-3]
* 浮层样式同备忘录保存浮层：./iteration/20260427_4/REQUIREMENT.md [新增于 it#68 20260501-5]
* 无论执行成功或失败，都需要收起浮层 [新增于 it#68 20260501-5]
* 复制图标与气泡垂直右对齐（不是水平，在气泡之下），与气泡间隔垂直3px [新增于 it#73 20260502-5]
* 气泡需求：./iteration/20260419_10/REQUIREMENT.md [新增于 it#76 20260502-8]
* 扇形菜单样式参考：style.jpeg [新增于 it#79 20260504-1]
* 返回数组中每个元素的name就是扇形菜单展示的名字 [新增于 it#79 20260504-1]
* 浮层需求：./iteration/20260504-1/REQUIREMENT.md [新增于 it#80 20260504-2]
*  第三部分：展示：重启、关闭、日志、取消，四个按钮，点击浮层或按ESC则自动取消 [新增于 it#80 20260504-2]
* 任何浮层外的操作（点击、拖动等）都要立即收起浮层（要有动画效果） [新增于 it#80 20260504-2]
* 点击扇形菜单展开后的插件浮层中的启动按钮，立即更新指定插件的配置（meta） [新增于 it#84 20260504-6]
* 扇形菜单需求：./iteration/20260504-2/REQUIREMENT.md [新增于 it#84 20260504-6]
* 右上角备忘录明细列表在鼠标悬停时弹出的详情浮层中新增一行“类型”字段展示 [新增于 it#85 20260505-1]
* 扇形菜单点击展开时，已经启动的插件应用要展示为绿色闪动 [新增于 it#90 20260505-6]
* 扇形菜单：./iteration/20260504-2/REQUIREMENT.md [新增于 it#90 20260505-6]

### Agent管理
* 动画效果使用animation而非transition实现，切换状态时使用classList.add/remove保留元素基础class，避免className整体替换导致样式丢失 [新增于 it#13 20260419_20]
* 切换状态时必须先remove所有状态class（hb-ok、hb-fail、hb-task）再add目标class，包括错误分支和catch分支，避免旧状态class残留 [新增于 it#13 20260419_20]
* 动画期间禁止切换浅色或深色模式 [新增于 it#21 20260421_1]
* 根据时间自动切换深色模式和浅色模式，每60秒检查一次 [新增于 it#22 20260421_2]
* 如果当前用户自己指定的模式，则在不关闭的情况下不进行自动切换 [新增于 it#22 20260421_2]
* 如果用户没有自己指定，即使用户在使用中，也需要切换 [新增于 it#22 20260421_2]
* 在设置中新增Agent图标的右侧增加一个删除图标，点击后弹出二次确认框，确认后删除Agent并立即刷新下拉列表 [新增于 it#24 20260421_4]
* 删除成功后需要同步清除本地存储中的agentId，避免再次打开设置时下拉列表出现空行 [新增于 it#24 20260421_4]
* 在设置Agent后的图标的都调整为15x15px [新增于 it#25 20260421_5]
* 每个Agent的小蜜蜂图标和蜂群配置是独立的 [新增于 it#33 20260425_2]
* 空列表或Agent不存在或已删除则不展示小蜜蜂图标 [新增于 it#33 20260425_2]
* 开启蜂群后，设置中当前Agent后的小蜜蜂图标展示色彩，关闭时展示灰色。但无论开启或关闭，每5秒需要展示一层顺时针环绕的光圈，增加提示度 [新增于 it#33 20260425_2]
* 在设置中切换Agent时，需要关闭蜂群配置，每次打开时需要读取实时数据 [新增于 it#33 20260425_2]
* Config的需求：../agent/REQUIREMENT.md，Thinking时属性thinking=true，Auto则为false [新增于 it#35 20260425_4]
* Thinking和Auto的切换要有动画效果，使用滑动按钮 [新增于 it#35 20260425_4]
* 每个Agent的Thinking配置是独立的，与蜂巢需求一致 [新增于 it#35 20260425_4]
* 空列表或Agent不存在或已删除则不展示Thinking图标 [新增于 it#35 20260425_4]
* 在设置中切换Agent时，需要读取实时数据 [新增于 it#35 20260425_4]
* 每个Agent的Thinking配置是独立的 [新增于 it#36 20260426_1]
* 提示文案：切换Agent会丢弃尚未保存的备忘录 [新增于 it#41 20260428-1]
* 完成所有终止操作完成后切换回待发送状态和界面，并Tips取消成功 [新增于 it#45 20260428-13]
* 在左上备忘录的信封小图标后增加一个切换小图标，点击后备忘录模块的展示切换为已经创建的、非仅一次的备忘录元数据 [新增于 it#48 20260428-3]
* 下半部分右侧展示切换小图标，点击则切换回原来的备忘录布局 [新增于 it#48 20260428-3]
* 切换和悬浮展示/收起内容需要有淡入淡出的效果 [新增于 it#48 20260428-3]
* 右上角的备忘录任务明细列表和备忘录元数据列表增加Agent名称 [新增于 it#59 20260430-5]
* 启动成功则延迟2秒后切换到日志查看（延迟期间所要锁定界面，防止操作，有动画切换渐变） [新增于 it#84 20260504-6]

### 日志系统
* 开场动画中心图标需仅保留icon.png中间logo主体的透明有效区域，去除底图与半透明背景纹理，避免播放时出现额外图层感或非主体方块残影 [新增于 it#78 20260503-2]
* 日志读取的展示风格需要与右侧CMD子面板中的样式相同，但不需要复制和终止按钮 [新增于 it#83 20260504-5]
* 日志查看需求：./iteration/20260504-5/REQUIREMENT.md [新增于 it#84 20260504-6]
* 涉及弹窗、日志面板、确认框、遮罩层、透明点击层、复制按钮、全局选区逻辑，默认都必须遵守“交互域隔离 + 隐藏视图彻底失活”原则 [新增于 it#91 20260505-7]

### 提交与保存
* 设置面板需要禁止浏览器自动填充和保存密码提示，使用autocomplete="off"和autocomplete="new-password" [新增于 it#23 20260421_3]
* 蜂群开关（左对齐）和保存按钮（右对齐）并排，减少空间占用，保存后自动收起蜂群配置 [新增于 it#33 20260425_2]
* 仅当保存后配置才生效 [新增于 it#35 20260425_4]
* 点击备忘录的保存按钮，保存数据 [新增于 it#40 20260427_4]
* 等待发送需求：./iteration/20260428-12/REQUIREMENT.md [新增于 it#45 20260428-13]
* 待发送样式需求：./iteration/20260428-12/REQUIREMENT.md [新增于 it#45 20260428-13]
* 删除前需要二次确认，样式保存备忘录的样式：./iteration/20260427_4/REQUIREMENT.md [新增于 it#48 20260428-3]
* 备忘录元数据样式：./iteration/20260427_1/REQUIREMENT.md，但不需要保存和取消按钮 [新增于 it#50 20260428-5]
* 保存成功后立即刷新展示新的任务明细 [新增于 it#51 20260428-6]
* 保存备忘录需求：./iteration/20260427_4/REQUIREMENT.md [新增于 it#52 20260428-7]
* 保存方式同：./iteration/20260502-1/REQUIREMENT.md [新增于 it#70 20260502-2]
* 备忘录保存需求：./iteration/20260427_4/REQUIREMENT.md [新增于 it#75 20260502-7]
* 实际提交内容：[FILE:图片绝对路径] [新增于 it#76 20260502-8]

### 展示与预览
* 居中对话框需要支持HTML片段渲染，同时兼容现有的Markdown和Latex公式 [新增于 it#98 20260517-3]
* 支持的HTML结构：div/section/article/p/span/br/hr、h1-h6、strong/em/u/del/sub/sup、ul/ol/li、blockquote/details/summary、pre/code、table/thead/tbody/tr/th/td、img/iframe/video/audio/source、figure/figcaption、a [新增于 it#98 20260517-3]
* 支持的HTML属性：class/title/aria-label/aria-hidden/role、href/target/rel、src/loading/allow/allowfullscreen/referrerpolicy、src/alt/loading/width/height [新增于 it#98 20260517-3]
* 居中对话框需要支持Latex公式渲染，同时兼容现有的Markdown [新增于 it#97 20260517-2]
* HTML：预览方式为浏览器渲染内容，可编辑，编辑后重新刷新渲染 [新增于 it#8 20260419_16]
* 图片：预览方式为浏览器渲染二进制流，不可编辑 [新增于 it#8 20260419_16]
* PDF：使用pdf.js预览 [新增于 it#8 20260419_16]
* 不支持的格式仅提示，不展示编辑 [新增于 it#8 20260419_16]
* HTML预览：iframe需要允许脚本、表单、弹窗等交互 [新增于 it#8 20260419_16]
* PDF预览：使用pdf.js [新增于 it#8 20260419_16]
* 预览需求：./iteration/20260419_16/REQUIREMENT.md [新增于 it#10 20260419_18]
* 如果有多个则用空格分隔，超出展示则省略 [新增于 it#27 20260422_1]
* 下半部分展示今天按时间排序的滚动备忘录明细列表，样式为时间轮，先用模拟数据填充 [新增于 it#37 20260427_1]
* 上半部分占7/8，以列表的形式展示备忘录元数据，需要支持数据过多时的滚动 [新增于 it#48 20260428-3]
* 列表各列固定宽度，整体宽度不要超过容器宽度，超出内容以省略号截断；时间列格式为 MM-DD HH:mm，不展示年份 [新增于 it#48 20260428-3]
* 右上角备忘录第一格日历下半部分的备忘录明细列表，用真实数据填充，样式保持时间轮，高度要先保证日历完全展示（不需要滚动）再间隔5px，然后才是列表 [新增于 it#50 20260428-5]
* 列表展示：时间、备忘录内容（超过列宽则用...替代），每30秒刷新一次（重新读取），临近30分钟内待执行的需要有淡橙色的闪烁框 [新增于 it#50 20260428-5]
* 如果当前任务明细为空，使用--无待执行任务--的灰色字体展示空列表，不需要其他内容 [新增于 it#50 20260428-5]
* 列表各列固定宽度（时间展示不能换行），整体宽度不要超过容器宽度；时间列格式为 MM-DD HH:mm，不展示年份，超出备忘录内容以省略号截断，保证单行不超出不换行 [新增于 it#50 20260428-5]
* 备忘录明细在展示时如果有多条且需要加载滚动条时需要将相对于当前时间，下一个待执行任务明细（淡橙色）滚动到列表当前展示的第一条 [新增于 it#53 20260428-8]
* 右上角备忘录明细处于完成状态时，在最前方展示小放大镜图标 [新增于 it#56 20260430-2]
* 刷新备忘录元数据列表和备忘录明细列表的同时，滚动到列表当前展示的第一条 [新增于 it#58 20260430-4]
* 代码块保留水平滚动，滚动条仅悬停时显示（默认隐藏） [新增于 it#63 20260430-9]
* 右侧分割带标题后仅新增一个CMD风格小图标作视觉展示： [新增于 it#64 20260501-1]
* 右侧CLI子面板的每一条SSE响应的黏贴图标后展示一个终止小图标，与黏贴小图标使用不用色差 [新增于 it#68 20260501-5]
* 点击后展示浮动确认是否需要终止命令，点击确认后执行终止 [新增于 it#68 20260501-5]
* 图片的预览方式修改为可编辑，增加橡皮擦功能，将涂抹的区域都变为透明度为0（Alpha=0） [新增于 it#69 20260502-1]
* 图片预览需求：./iteration/20260419_16/REQUIREMENT.md:17 [新增于 it#69 20260502-1]
* 图片的预览方式增加使用不同色笔和线粗（边框粗细）给指定区域画圈 [新增于 it#70 20260502-2]
* 图片的预览方式的修改编辑能力增加后退上N步操作，最多后退10步 [新增于 it#71 20260502-3]
* 例如已经展示，但http://xxxx/image.png地址已经404 [新增于 it#72 20260502-4]
* 扇形按钮的数量和展示文字从api/plugins/meta获取 [新增于 it#79 20260504-1]
* 第二部分：展示该plugin的可选meta和已填meta（默认值） [新增于 it#80 20260504-2]

### 布局与尺寸
* 左右侧边栏收起或展开时，自动调整居中对话框文字/Markdown/HTML的展示宽度，保持最大化可用宽度 [新增于 it#106 20260517-11]
* 以上图标需要自动调整布局，保证隐藏时不产生错位 [新增于 it#28 20260422_2]
* 布局（右上）保持原来的上下两部分：./iteration/20260427_1/REQUIREMENT.md [新增于 it#48 20260428-3]
* 备忘录明细列表和备忘录元数据列表都是固定高，如果数据过多需要滚动加载，而不是超出容器高度 [新增于 it#53 20260428-8]
* Grid布局的子元素（如田字格的四个格子）默认min-height:auto，内容过多时会撑破格子高度而非触发滚动。需要在grid容器及其子元素上设置min-height:0，确保overflow-y:auto生效，内容在固定高度内滚动 [新增于 it#53 20260428-8]

### 其他
* Proxy相关需求：../proxy/iteration/20260419_4/REQUIREMENT.md [新增于 it#2 20260419_10]
* 样式和功能参考：./iteration/20260419_4/REQUIREMENT.md [新增于 it#2 20260419_10]
* Proxy相关需求：../proxy/iteration/20260419_5/REQUIREMENT.md [新增于 it#4 20260419_12]
* Markdown风格：./iteration/20260419_11/REQUIREMENT.md [新增于 it#4 20260419_12]
* Proxy相关需求：../proxy/iteration/20260419_6/REQUIREMENT.md [新增于 it#4 20260419_12]
* Proxy相关需求：../proxy/iteration/20260419_9/REQUIREMENT.md [新增于 it#5 20260419_13]
* 努力工作中和暂停键需求：./REQUIREMENT.md [新增于 it#7 20260419_15]
* 如果还在等待，则恢复边缘锐化闪光、"努力工作中"及暂停键效果 [新增于 it#7 20260419_15]
* 文本格式：MD、TXT、XML、JSON等依旧使用Markdown：./iteration/20260419_12/REQUIREMENT.md [新增于 it#8 20260419_16]
* Proxy需求：../proxy/iteration/20260419_10/REQUIREMENT.md [新增于 it#8 20260419_16]
* 禁止后退、转跳等所有会离开页面本身的操作或脚本执行 [新增于 it#9 20260419_17]
* 禁止后退、跳转等所有会离开页面本身的操作或脚本执行 [新增于 it#9 20260419_17]
* 使用replaceState替换history条目而非pushState压栈，确保后退按钮无可退记录 [新增于 it#9 20260419_17]
* 配合hashchange监听兜底，防止任何导航触发页面重载或断开SSE连接 [新增于 it#9 20260419_17]
* /A/REQUIREMENT.md，则直接使用绝对路径查找 [新增于 it#10 20260419_18]
* Proxy需求：../proxy/iteration/20260419_5/REQUIREMENT.md [新增于 it#10 20260419_18]
* 前后标点、空格、空行分隔的文字 [新增于 it#11 20260419_19]
* 链接、超链接 [新增于 it#11 20260419_19]
* 一整句中文描述 [新增于 it#11 20260419_19]
* 带标点的说明文字 [新增于 it#11 20260419_19]
* 混合说明和路径的长句 [新增于 it#11 20260419_19]
* 多行一起选中的路径块 [新增于 it#11 20260419_19]
* Proxy需求：../proxy/iteration/20260419_11/REQUIREMENT.md [新增于 it#13 20260419_20]
* 每3秒扫描一次最近心跳： [新增于 it#13 20260419_20]
* 如果心跳成功则使用当前SVG并进行节奏的闪动 [新增于 it#13 20260419_20]
* 如果心跳失败则`灰色`当前SVG并进行快节奏的闪动 [新增于 it#13 20260419_20]
* 如果执行任务则`绿色`当前SVG并进行超快节奏的闪动，饱和度降低20% [新增于 it#13 20260419_20]
* 连续5次心跳失败才算失败 [新增于 it#13 20260419_20]
* hue-rotate角度需要根据原色（accent约217°）精确计算到目标色相，蓝色到绿色（120°）应为-97deg而非+90deg [新增于 it#13 20260419_20]
* 二次提醒时按下回车，则表示删除 [新增于 it#15 20260419_4]
* 关联需求参考：../proxy/iteration/20260419_3/REQUIREMENT.md [新增于 it#17 20260419_6]
* 动画参考实现：algalon_intro.html，中心圆球边缘发亮，中心透明，转速由极快到慢 [新增于 it#21 20260421_1]
* 使用八角星（The Octagram） [新增于 it#21 20260421_1]
* 动画中心文案：修改为`DeepRight` [新增于 it#21 20260421_1]
* 需要考虑浅色背景和深色背景的配色 [新增于 it#21 20260421_1]
* 动画结束后要有向中心收拢的效果 [新增于 it#21 20260421_1]
* 浅色模式：系统时间7点-19点 [新增于 it#22 20260421_2]
* 深色模式：系统时间19点-7点 [新增于 it#22 20260421_2]
* Proxy需求：../proxy/iteration/20260420_1/REQUIREMENT.md [新增于 it#23 20260421_3]
* 名称要符合操作系统命名规范，只允许英文，不允许特殊字符和空格 [新增于 it#23 20260421_3]
* Proxy需求：../proxy/iteration/20260420_2/REQUIREMENT.md [新增于 it#24 20260421_4]
* 前端调用API时需要处理非JSON响应（如404、405等纯文本错误），解析前先检查Content-Type或用try-catch包裹resp.json()，避免JSON解析异常 [新增于 it#24 20260421_4]
* Proxy需求：../proxy/iteration/20260420_3/REQUIREMENT.md [新增于 it#26 20260421_6]
* Proxy需求：../proxy/iteration/20260419_3/REQUIREMENT.md [新增于 it#26 20260421_6]
* Proxy需求：../proxy/iteration/20260422_1/REQUIREMENT.md [新增于 it#27 20260422_1]
* 上传失败则内容为失败原因 [新增于 it#27 20260422_1]
* 上传取消则提示上传取消 [新增于 it#27 20260422_1]
* 当访问的Host不是localhost或127.0.0.1（本地访问时）隐藏： [新增于 it#28 20260422_2]
* 悬浮框需求：./iteration/20260422_3/REQUIREMENT.md [新增于 it#30 20260422_4]
* 提示框风格参考：./iteration/20260419_2/REQUIREMENT.md [新增于 it#32 20260425_1]
* 样式参考小太阳图标：./iteration/20260422_4/REQUIREMENT.md [新增于 it#33 20260425_2]
* 蜂群（swarm）配置项：开启/关闭，蜂群描述 [新增于 it#33 20260425_2]
* Config需求：../proxy/iteration/20260425-1/REQUIREMENT.md [新增于 it#33 20260425_2]
* 蜂巢需求：./iteration/20260425_2/REQUIREMENT.md [新增于 it#35 20260425_4]
* 读取服务端实时数据的fetch必须带时间戳参数防缓存，读取失败时UI重置为默认值 [新增于 it#35 20260425_4]
* 多个异步调用共享同一全局状态时，必须用序列号丢弃过期回调结果 [新增于 it#35 20260425_4]
* Thinking按钮需求：./iteration/20260425_4/REQUIREMENT.md [新增于 it#36 20260426_1]
* 上半部分切分为田字四方格 [新增于 it#37 20260427_1]
* 左上第一个格子水平切分为上下两部分 [新增于 it#37 20260427_1]
* 上下两部分以一条透明度70%的细线分割 [新增于 it#37 20260427_1]
* 右上第一个格子切分为上下两部分 [新增于 it#37 20260427_1]
* 如果用户焦点不在时间轮，则每10秒同步当前系统时间 [新增于 it#37 20260427_1]
* 以单页HTML编写以上代码，要求： [新增于 it#37 20260427_1]
* 拖入备忘录需求：./iteration/20260427_1/REQUIREMENT.md [新增于 it#38 20260427_2]
* 光圈需求：./iteration/20260425_3/REQUIREMENT.md [新增于 it#38 20260427_2]
* 对周期任务（工作日/自然日）在创建后会立即生成后5天内的所有任务明细，不用等定时器，对一次性任务如果执行时间早于当前时间（比如当前时间是10点01分，10点禁止执行，10点01分可以执行，10点02分可以执行），禁止创建并提示Tips（等于当前时间可以创建） [新增于 it#40 20260427_4]
* SSE响应可能分为多条，每次渲染时都需要更新时间和状态 [新增于 it#42 20260428-10]
* SSE还未结束：状态=等待 [新增于 it#42 20260428-10]
* SSE已经结束：状态=完成 [新增于 it#42 20260428-10]
* 使用浅色字体，字号10px [新增于 it#42 20260428-10]
* 转圈等待需要有淡入淡出的动画效果 [新增于 it#44 20260428-12]
* 如果在终止前有任何一条SSE响应被渲染，则标记其为为取消，并更新时间 [新增于 it#45 20260428-13]
* 不允许出现前端已经取消，但下游转发连接仍继续执行的情况 [新增于 it#45 20260428-13]
* 如果有连接则保持原逻辑 [新增于 it#46 20260428-14]
* SSE渲染状态需求：./iteration/20260428-10/REQUIREMENT.md [新增于 it#46 20260428-14]
* Proxy需求：../proxy/iteration/20260427-10/REQUIREMENT.md [新增于 it#46 20260428-14]
* 对话状态需求：./REQUIREMENT.md [新增于 it#46 20260428-14]
* 轮询恢复过程需用户无感，体验应与连续等待SSE返回一致；非终态轮询不得出现明显闪屏、反复提示或整页重绘 [新增于 it#46 20260428-14]
* Proxy需求：../proxy/iteration/20260427-3/REQUIREMENT.md [新增于 it#47 20260428-2]
* 在列表最后增加一个大叉的按钮，用于表示删除 [新增于 it#48 20260428-3]
* Proxy需求：../proxy/iteration/20260427-2/REQUIREMENT.md [新增于 it#48 20260428-3]
* 删除后立即刷新备忘录列表和左上日历的备忘录明细列表 [新增于 it#48 20260428-3]
* 备忘录需求：./iteration/20260427_1/REQUIREMENT.md [新增于 it#50 20260428-5]
* Proxy需求：../proxy/iteration/20260427-5/REQUIREMENT.md [新增于 it#50 20260428-5]
* 如果存在多条备忘录明细超过滚动条时，自动滚到离开当前时间最近的待执行明细 [新增于 it#50 20260428-5]
* 备忘录元数据：./iteration/20260428-3/REQUIREMENT.md [新增于 it#53 20260428-8]
* 列表自动滚动定位到指定元素时，使用getBoundingClientRect计算相对于滚动容器的实际偏移量，不能依赖offsetTop（受offsetParent层级影响导致定位不准），且需在DOM渲染完成后延迟执行 [新增于 it#53 20260428-8]
* 备忘录明细需求：./iteration/20260428-5/REQUIREMENT.md [新增于 it#54 20260428-9]
* Proxy需求：../proxy/iteration/20260427-6/REQUIREMENT.md [新增于 it#54 20260428-9]
* 任务明细列表刷新： [新增于 it#54 20260428-9]
* 指定状态成功后立即刷新任务明细列表 [新增于 it#54 20260428-9]
* 如果鼠标不在任务明细列表中，每30秒刷新一次任务明细列表 [新增于 it#54 20260428-9]
* 滚动需求：./iteration/20260428-8/REQUIREMENT.md:14 [新增于 it#58 20260430-4]
* 轮询恢复需求：./iteration/20260428-14/REQUIREMENT.md:20 [新增于 it#62 20260430-8]
* 分割带左对齐标题：正在执行的系统指令 [新增于 it#63 20260430-9]
* 右侧第二栏CLI子任务面板样式调整： [新增于 it#63 20260430-9]
* 代码块内padding和margin压缩（padding:2px 4px，margin:8px 0），代码块之间保持可辨识间距 [新增于 it#63 20260430-9]
* 以Golang编写以上代码，要求： [新增于 it#63 20260430-9]
* 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写 [新增于 it#63 20260430-9]
* CLI子板板：./iteration/20260430-9/REQUIREMENT.md [新增于 it#64 20260501-1]
* 分割带需求：./iteration/20260430-9/REQUIREMENT.md [新增于 it#64 20260501-1]
* CMD小图标需求：./iteration/20260501-1/REQUIREMENT.md [新增于 it#65 20260501-2]
* Proxy需求：../proxy/iteration/20260501-1/REQUIREMENT.md [新增于 it#65 20260501-2]
* 执行成功后需要有成功提示 [新增于 it#65 20260501-2]
* 每次重新打开都要重置为空，不保留之前的执行记录 [新增于 it#65 20260501-2]
* 离开悬停后3秒小图标才消失 [新增于 it#66 20260501-3]
* CMD小图标需求：./iteration/20260501-2/REQUIREMENT.md [新增于 it#66 20260501-3]
* SSE响应由markdown ```标记包裹，需要去除包裹 [新增于 it#66 20260501-3]
* SSE响应原内容： [新增于 it#66 20260501-3]
* 黏贴的内容：HELLO WORLD [新增于 it#66 20260501-3]
* 不允许复制整组聚合内容 [新增于 it#66 20260501-3]
* CLI子面板需求：./iteration/20260501-3/REQUIREMENT.md [新增于 it#67 20260501-4]
* 黏贴小图标需求：./iteration/20260501-3/REQUIREMENT.md [新增于 it#68 20260501-5]
* Proxy终止需求：../proxy/iteration/20260501-2/REQUIREMENT.md [新增于 it#68 20260501-5]
* 命令取值逻辑同黏贴小图标：./iteration/20260501-3/REQUIREMENT.md [新增于 it#68 20260501-5]
* 新建图片需求：../proxy/iteration/20260502-1/REQUIREMENT.md [新增于 it#69 20260502-1]
* 需要支持的拖拉图像：圆、椭圆、矩形、三角形 [新增于 it#70 20260502-2]
* 需要支持以上图像的拖拉放大缩小 [新增于 it#70 20260502-2]
* 拖拉图像需求：./iteration/20260502-2/REQUIREMENT.md [新增于 it#71 20260502-3]
* 透明图片需求：./iteration/20260502-1/REQUIREMENT.md [新增于 it#71 20260502-3]
* 复制图标需求：./iteration/20260502-5/REQUIREMENT.md [新增于 it#74 20260502-6]
* 需要当前无SSE连接时才可以点击重试图标 [新增于 it#74 20260502-6]
* Proxy需求：../proxy/iteration/20260502-3/REQUIREMENT.md [新增于 it#75 20260502-7]
* Proxy需求：../proxy/iteration/20260502-1/REQUIREMENT.md [新增于 it#76 20260502-8]
* 删除按钮保持可见 [新增于 it#76 20260502-8]
* Proxy需求：../proxy/iteration/20260503-2/REQUIREMENT.md [新增于 it#77 20260503-1]
* 开场动画的中心需要围绕./icon.png，仅使用以图的中心，半径600px的圆 [新增于 it#78 20260503-2]
* 需要有由小变大的闪动效果，最后渐变消失，要震撼、爆发、科幻史诗 [新增于 it#78 20260503-2]
* 动画整体过程控制在8s [新增于 it#78 20260503-2]
* 需要区分浅色模式和深色模式 [新增于 it#79 20260504-1]
* Proxy需求：../proxy/iteration/20260503-9/REQUIREMENT.md [新增于 it#79 20260504-1]
* 浮动分三部分 [新增于 it#80 20260504-2]
* 获取meta需求：../proxy/iteration/20260503-9/REQUIREMENT.md [新增于 it#80 20260504-2]
* 如果定义参数过多，保持窗口大小，添加滚动条 [新增于 it#80 20260504-2]
* 最后一个参数与按钮的分隔线距离15px [新增于 it#80 20260504-2]
* 点击重启，立即通过/api/plugins/config更新配置，成功后通过/api/plugins/start启动插件 [新增于 it#80 20260504-2]
* Config需求：../proxy/iteration/20260503-11/REQUIREMENT.md [新增于 it#80 20260504-2]
* 开启关闭需求：../proxy/iteration/20260503-14/REQUIREMENT.md [新增于 it#80 20260504-2]
* 已关闭则关闭状态变灰，重启按钮始终可用 [新增于 it#80 20260504-2]
* 三个按钮右下对齐，离开底边5px，风格与设置中按钮相同 [新增于 it#80 20260504-2]
* Proxy需求：../proxy/iteration/20260503-10/REQUIREMENT.md [新增于 it#83 20260504-5]
* 更新配置需求需求：../proxy/iteration/20260503-11/REQUIREMENT.md [新增于 it#84 20260504-6]
* 配置更新成功后启动插件 [新增于 it#84 20260504-6]
* 启动插件需求：../proxy/iteration/20260503-13/REQUIREMENT.md [新增于 it#84 20260504-6]
* Proxy需求：../proxy/iteration/20260503-13/REQUIREMENT.md [新增于 it#85 20260505-1]
* Cron需求：../cron/iteration/20260502-6/REQUIREMENT.md [新增于 it#85 20260505-1]
* Proxy上传需求：../proxy/iteration/20260422_1/REQUIREMENT.md [新增于 it#86 20260505-2]
* 上传时需要锁定界面，防止操作，效果同：./iteration/20260504-6/REQUIREMENT.md [新增于 it#86 20260505-2]
* 关联需求的参考实现： [新增于 it#86 20260505-2]
* ./iteration/20260422_1/REQUIREMENT.md [新增于 it#86 20260505-2]
* ./iteration/20260427_2/REQUIREMENT.md [新增于 it#86 20260505-2]
* ./iteration/20260425_3/REQUIREMENT.md [新增于 it#86 20260505-2]
* ./iteration/20260428-4/REQUIREMENT.md [新增于 it#86 20260505-2]
* 备忘录上传需求： [新增于 it#86 20260505-2]
* ./iteration/20260427_1/REQUIREMENT.md [新增于 it#86 20260505-2]
* 需要使用/api/plugins/status判断是否可关闭 [新增于 it#90 20260505-6]
* 是否已启动需求：../proxy/iteration/20260503-16/REQUIREMENT.md [新增于 it#90 20260505-6]
* 需要有同样的背景模糊处理 [新增于 it#91 20260505-7]
* 能用开源包的就用开源包 [新增于 it#91 20260505-7]
* 代码简洁，包体积越小越好 [新增于 it#91 20260505-7]

### 历史记录（已被覆盖，不再适用）
* ~~以单页HTML编写以上代码，要求：~~
* ~~能用开源包的就用开源包~~
* ~~代码简洁，包体积越小越好~~
* ~~Proxy相关需求：../proxy/iteration/20260419_5/REQUIREMENT.md~~
* ~~查找明确的单一路径，如/a/b/c.md或./iteration/20260510-1/wiki_md/index.md、./iteration/20260510-1/wiki_md/index.md或单个像文件名的内容，如REQUIREMENT.md~~
* ~~一整句中文描述~~
* ~~带标点的说明文字~~
* ~~混合说明和路径的长句~~
* ~~多行一起选中的路径块~~
* ~~关联需求参考：../proxy/iteration/20260419_3/REQUIREMENT.md~~
* ~~打开前需要二次提醒，风格为浮层，参考：./iteration/20260419_3/REQUIREMENT.md~~
* ~~前端调用API时需要处理非JSON响应（如404、405等纯文本错误），解析前先检查Content-Type或用try-catch包裹resp.json()，避免JSON解析异常~~
* ~~读取服务端实时数据的fetch必须带时间戳参数防缓存，读取失败时UI重置为默认值~~
* ~~多个异步调用共享同一全局状态时，必须用序列号丢弃过期回调结果~~
* ~~当文件或目录拖动至对话框输入区正确位置时，对话框需要圈绕光圈提示，直到正确拖入、移出或取消~~
* ~~Proxy需求：../proxy/iteration/20260427-2/REQUIREMENT.md~~
* ~~所有Cron相关前端高频触发的数据刷新（如切换会话、切换Agent）需要防抖处理，避免短时间内重复请求~~
* ~~浮层风格参考：./iteration/20260427_4/REQUIREMENT.md~~
* ~~Proxy需求：../proxy/iteration/20260427-10/REQUIREMENT.md~~
* ~~SSE渲染状态需求：./iteration/20260428-10/REQUIREMENT.md~~
* ~~气泡需求：./iteration/20260419_10/REQUIREMENT.md~~
* ~~备忘录元数据：./iteration/20260428-3/REQUIREMENT.md~~
* ~~备忘录明细需求：./iteration/20260428-5/REQUIREMENT.md~~
* ~~以Golang编写以上代码，要求：~~
* ~~作为其他模块可以调用的子模块和可独立运行的CLI命令来编写~~
* ~~前端弹层层级规范：左侧Sidebar、居中操作区、右侧Sidebar的overlay必须分域管理，禁止跨区域全屏遮挡；所有透明层在非激活态必须pointer-events:none，避免误拦截底层操作~~
* ~~CMD面板的全屏遮罩层不能覆盖右侧Sidebar区域，避免遮罩层拦截CLI子任务面板的复制/终止按钮点击事件~~
* ~~所有固定定位遮罩层、确认框、弹窗必须统一管理层级、显示状态和pointer-events，按左侧Sidebar、居中会话区、右侧Sidebar分域隔离，禁止未激活浮层跨区域遮挡或拦截其他可操作控件点击~~
* ~~需要使用/api/plugins/status判断是否可关闭~~
* ~~是否已启动需求：../proxy/iteration/20260503-16/REQUIREMENT.md~~
* ~~扇形菜单：./iteration/20260504-2/REQUIREMENT.md~~
* ~~需要有同样的背景模糊处理~~
* ~~涉及弹窗、日志面板、确认框、遮罩层、透明点击层、复制按钮、全局选区逻辑，默认都必须遵守“交互域隔离 + 隐藏视图彻底失活”原则~~

### 对话框与输入框
> 新增自 iteration/20260419_10/REQUIREMENT.md
+ 对话框中输入`@`触发可以使用键盘控制的菜单浮层。浮层的左下角紧贴@字符向右上展开，有2个选项：文件和技能，可用鼠标或键盘选择
> 新增自 iteration/20260419_10/REQUIREMENT.md
+ 选择具体文件或目录，将在对话框中`待提交内容的光标处`展示特殊反差色系气泡，气泡展示内容为`[FILE:文件路径的最后一级文件或目录名称]`，气泡在提交请求是内容替换为`[FILE:文件或目录绝对路径]`
> 新增自 iteration/20260419_10/REQUIREMENT.md
+ 选择具体技能，将在对话框中`待提交内容的光标处`展示特殊反差色系气泡，气泡展示内容和提交请求的内容均为`[SKILL:技能名称]`
> 新增自 iteration/20260419_10/REQUIREMENT.md
+ 当前光标待输入位置：已输入ABC，选择文件/DEV/X，再输入DEF，则展示内容为ABC[FILE:X]DEF，提交内容为ABC[FILE:/DEV/X]DEF，删除气泡则为ABCDEF
> 新增自 iteration/20260419_10/REQUIREMENT.md
+ 当前光标待输入位置：已输入ABC，选择技能X，再输入DEF，则展示和提交内容为ABC[SKILL:X]DEF，删除气泡则为ABCDEF

**编码约束：**
> 新增自 iteration/20260419_11/REQUIREMENT.md
+ 对话框使用contenteditable实现，提取用户输入内容时需要递归遍历DOM节点，正确处理换行（BR元素和DIV/P块级元素转为\n），确保多行内容（如Markdown表格）的换行符不丢失
> 新增自 iteration/20260419_11/REQUIREMENT.md
+ 使用contenteditable输入框粘贴内容时必须剥离HTML格式只保留纯文本, 防止带格式内容撑破布局
> 新增自 iteration/20260419_14/REQUIREMENT.md
+ 最后（最新）的提问气泡在渲染响应报文前，边框以5px边缘锐化闪光，响应报文开始渲染后停止锐化闪光
> 新增自 iteration/20260419_15/REQUIREMENT.md
+ 对话框用户提问气泡的边缘锐化闪光效果、"努力工作中"、暂停键效果与会话绑定，仅在当前会话正在等待消息时触发
> 新增自 iteration/20260419_15/REQUIREMENT.md
+ 用户提问气泡的边缘锐化闪光需求：./iteration/20260419_14/REQUIREMENT.md
> 新增自 iteration/20260419_15/REQUIREMENT.md
+ 切换会话后，尚未发送的用户输入需要以会话纬度暂存，切换回后重新填充（包括@文件或技能的气泡）
> 新增自 iteration/20260419_15/REQUIREMENT.md
+ 不同会话的输入框、暂停键（或其他中断）以会话纬度相互独立，互不影响
> 新增自 iteration/20260419_15/REQUIREMENT.md
+ 如果已经完成，则不用恢复，保持再次等待发送消息的效果

**编码约束：**
> 新增自 iteration/20260419_18/REQUIREMENT.md
+ 在对话框中双击、拖拉、框选范围文字时，模糊查找在当前Agent工作目录（workspace）下是否存在该文件且可预览，如果可预览则展示气泡提示
> 新增自 iteration/20260419_18/REQUIREMENT.md
+ 在对话框中双击、拖拉、框选范围链接或HTML超链接（<a>）时，直接展示气泡提示
> 新增自 iteration/20260419_18/REQUIREMENT.md
+ 气泡风格：./iteration/20260419_10/REQUIREMENT.md
> 新增自 iteration/20260419_18/REQUIREMENT.md
+ 对话框中点击文件路径气泡时，应基于相对路径或是绝对路径解析并打开实际存在的文件，而不是错误拼接到无关目录后提示文件不存在
> 新增自 iteration/20260419_19/REQUIREMENT.md
+ 在对话框中双击文字时，自动框选双击范围内：
> 新增自 iteration/20260419_4/REQUIREMENT.md
+ 二次提醒时按下回车，则表示删除
> 新增自 iteration/20260419_6/REQUIREMENT.md
+ 在对话框发送按钮右侧（对话框外），添加15x15像素，透明度50%的小文件夹图标
> 新增自 iteration/20260419_8/REQUIREMENT.md
+ 对话框中输入文字（包括placeholder）需要水平居中对齐，尤其在单行时不要有错位感

**编码约束：**
> 新增自 iteration/20260419_8/REQUIREMENT.md
+ contenteditable换行产生的子元素（div/p）需要继承居中对齐，使用通配符选择器强制所有子元素text-align:center
> 新增自 iteration/20260419_9/REQUIREMENT.md
+ 对话框中展示用户已发送请求和已接收响应的气泡<div>在左右各留空50px，避免展示过于紧凑
> 新增自 iteration/20260421_1/REQUIREMENT.md
+ 用户开始输入或其他任何操作时，立即触发结束动画，然后开始执行操作
> 新增自 iteration/20260421_3/REQUIREMENT.md
+ 在设置中的`打开Agent目录`右侧增加一个新增图标，点击后弹出名称输入框，确认后创建Agent并立即刷新下拉列表
> 新增自 iteration/20260421_4/REQUIREMENT.md
+ 二次输入框内容：确定删除`名称`？

**编码约束：**
> 新增自 iteration/20260421_6/REQUIREMENT.md
+ 新建：显示浮层，输入名称，选择文件或目录
> 新增自 iteration/20260422_2/REQUIREMENT.md
+ 对话框输入栏右侧`打开Agent目录`
> 新增自 iteration/20260422_3/REQUIREMENT.md
+ 对话输入框上展示高度为25px的悬浮框，居中对齐，展示当前正在使用的模型和Agent，字号需要自动适配
> 新增自 iteration/20260422_4/REQUIREMENT.md
+ 在对话输入框上提示当前Agent和模型的悬浮框右侧，添加一个1.5倍px大小的小太阳图标，用来打开设置页面，为当前会话切换Agent
> 新增自 iteration/20260422_5/REQUIREMENT.md
+ 对话框中输入`@`触发文件，如果是相对路径则以当前Agent目录为根目录开始查找
> 新增自 iteration/20260425_1/REQUIREMENT.md
+ 未选择Agent或Agent不存在均不允许发送请求
> 新增自 iteration/20260425_2/REQUIREMENT.md
+ 开启蜂群后，对话框输入框后小太阳图标后追加小蜜蜂图标，同节奏闪烁，关闭蜂群则消失
> 新增自 iteration/20260425_3/REQUIREMENT.md
+ 从虚拟文件系统或实际操作系统的文件或目录拖入对话框输入区域后自动转换为系统绝对路径
> 新增自 iteration/20260425_3/REQUIREMENT.md
+ 展示样式和插入区域参考@文件后的小气泡需求：./iteration/20260419_10/REQUIREMENT.md
> ~~覆盖自 iteration/20260425_3/REQUIREMENT.md，已被 20260427_1 覆盖~~
+ ~~当文件或目录拖动至对话框输入区正确位置时，对话框需要圈绕光圈提示，直到正确拖入、移出或取消~~
> 新增自 iteration/20260426_1/REQUIREMENT.md
+ 在输入对话框发送按钮正上方同步展示Thinking滑动按钮，状态需要与设置中同步
> 新增自 iteration/20260426_1/REQUIREMENT.md
+ 在对话框之外，与小蜜蜂图标水平对齐，与发送按钮垂直且右侧对齐
> 修正/扩展自 iteration/20260427_1/REQUIREMENT.md
+ 当文件或目录拖动至对话框输入区正确位置时，对话框需要圈绕光圈提示，直到正确拖入、移出或取消
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 上半部分占7/8，展示一个没有发送按钮的对话输入框，样式同居中对话框的输入框，占位符内容为`xxx的备忘录...`，字体大小为20px，高度为150px，其中xxx为当前Agent名称
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 使用同样的小气泡展示，包括光标和插入策略：./iteration/20260419_10/REQUIREMENT.md
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 备忘录菜单浮层也需要响应键盘确认（回车）或取消（ESC），输入完毕或离开备忘录输入框时要自动收起
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 备忘录菜单浮层的左下角紧贴@备忘录字符向右上展开，与居中输入框的浮层相互独立
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 备忘录菜单浮层不同于居中的输入框浮层，需要向下展开
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 需要支持选择模型，使用30px同样的小地球图标，选择后的模型字体大小和选择框内的待选择模型字体大小均为8px，需要一次全部展示，样式同输入框小地球，需要可以看到勾选的选择后模型，选择时需要有淡入淡出效果
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 点击取消时，重置输入框为`xxx的备忘录...`，重置时间为当前时间，重置模型为待选择
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 多个输入框共存时, 事件操作(气泡插入、拖入等)必须作用于触发事件的目标输入框, 占位符与输入内容的样式需独立控制
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ contenteditable输入框: 粘贴时剥离HTML只保留纯文本, 设置明确的max-height配合overflow-y:auto防止撑破布局
> 新增自 iteration/20260427_2/REQUIREMENT.md
+ 拖入对话框需求：./iteration/20260425_3/REQUIREMENT.md
> 新增自 iteration/20260427_2/REQUIREMENT.md
+ 禁止拖入时输入框光圈变为红色，并tips原因（居中输入框和备忘录输入框都需要）
> 新增自 iteration/20260427_2/REQUIREMENT.md
+ 居中对话框的tips浮层向上展示
> 新增自 iteration/20260427_4/REQUIREMENT.md
+ 提交前先渐变展开浮层，让用户确认除输入框内容之外的配置，再次点击浮层的保存后执行
> 新增自 iteration/20260427_4/REQUIREMENT.md
+ 如果当前未选择模型或未在输入框输入非空内容，使用Tips提示，同时不要展示该浮层，点击非浮层区域自动收起，再次点击保存后再次以最新数据展开

**编码约束：**
> 新增自 iteration/20260428-10/REQUIREMENT.md
+ 居中输入框在收到SSE响应并渲染时需要增加收到响应的时间日期和状态，以yyyy-MM-dd hh:mm:ss的格式展示在气泡的右下方，右对齐
> 新增自 iteration/20260428-11/REQUIREMENT.md
+ 居中输入框右侧文件夹图标上增加一个重新加载的图标，点击图标展开浮层，提示用户是否要重新加载会话
> 新增自 iteration/20260428-12/REQUIREMENT.md
+ 在居中输入框发送SSE请求时，在界面等待3s后再向后段发送请求，等待期间`整个页面`的垂直水平居中位置展示100x100px的转圈等待图标，发送成功后消失
> 新增自 iteration/20260428-13/REQUIREMENT.md
+ 点击居中对话框的输入框在等待SSE响应期间的终止会话图标
> 新增自 iteration/20260428-13/REQUIREMENT.md
+ 如果当前界面正在等待发送，则立即终止待发送消息
> 新增自 iteration/20260428-13/REQUIREMENT.md
+ 等待发送需求：./iteration/20260428-12/REQUIREMENT.md
> 新增自 iteration/20260428-13/REQUIREMENT.md
+ 如果已经发送请求
> 新增自 iteration/20260428-13/REQUIREMENT.md
+ 如果没有渲染任何SSE响应（包括取消还没发送的请求），则自动补充最后一条响应消息为已取消，标记为已取消，更新时间（样式与正常的SSE响应一致）
> 新增自 iteration/20260428-13/REQUIREMENT.md
+ 完成所有终止操作完成后切换回待发送状态和界面，并Tips取消成功
> 新增自 iteration/20260428-13/REQUIREMENT.md
+ 待发送样式需求：./iteration/20260428-12/REQUIREMENT.md

**编码约束：**
> 新增自 iteration/20260428-13/REQUIREMENT.md
+ 因为发送请求和发送取消是异步会有时间差，所有在发送终止请求时要等待3s，界面展示为转圈等待（同等待发送样式）
> 新增自 iteration/20260428-14/REQUIREMENT.md
+ 对话框状态在一次到多次到5秒轮询并确认收到完成状态的SSE响应后按原逻辑渲染并恢复为待发送状态，否则持续保持等待响应的样式
> 新增自 iteration/20260428-14/REQUIREMENT.md
+ 恢复会话时，空内容的SSE片段记录（如换行符、data: [DONE]终止标记）不应创建新的助手消息气泡，仅追加到当前助手消息的原始数据中用于终态检测
> 新增自 iteration/20260428-14/REQUIREMENT.md
+ 自动轮询恢复必须完整拉取本轮SSE的全部增量和终态；收到终态后立即恢复待输入，不允许出现只加载部分内容或结束后仍停留在等待状态
> 新增自 iteration/20260428-14/REQUIREMENT.md
+ 发送后立即刷新页面时，恢复逻辑不得重复渲染当前这次已发送请求；对已存在的本地请求应直接复用并继续恢复后续响应
> ~~覆盖自 iteration/20260428-4/REQUIREMENT.md，已被 20260502-8 覆盖~~
+ ~~气泡需求：./iteration/20260419_10/REQUIREMENT.md~~
> 新增自 iteration/20260428-6/REQUIREMENT.md
+ 在二次确认保存备忘录成功，在重置备忘录输入框、备忘录模型、备忘录思考模式后立即切换到备忘录元数据列表
> 新增自 iteration/20260428-7/REQUIREMENT.md
+ 如果任务元数据的输入框中无内容、空内容（trim后无内容）或仅为默认值时，需要立即以当前AgentId的占位符来刷新备忘录输入框的默认值并恢复展示输入框为默认值

**编码约束：**
> 新增自 iteration/20260430-1/REQUIREMENT.md
+ 下方操作栏（周期标签、时间选择、图标按钮）间距紧凑化，禁止换行（flex-wrap:nowrap），gap和padding缩小
> 新增自 iteration/20260430-2/REQUIREMENT.md
+ 居中对话框立即使用轮询来恢复对话内容（因为首次恢复时没有时间，使用备忘录明细的启动时间）、样式与普通会话一致：./iteration/20260428-14/REQUIREMENT.md
> 新增自 iteration/20260430-4/REQUIREMENT.md
+ 如果用户的光标/焦点没有停留在备忘录元数据、备忘录明细、备忘录日历，每30秒刷新一次最新数据
> 新增自 iteration/20260430-9/REQUIREMENT.md
+ 代码块内padding和margin压缩（padding:2px 4px，margin:8px 0），代码块之间保持可辨识间距
> 新增自 iteration/20260430-9/REQUIREMENT.md
+ 气泡高度自适应内容，不设固定高度限制
> 新增自 iteration/20260430-9/REQUIREMENT.md
+ 气泡字体12px，代码块字体继承一致
> 新增自 iteration/20260430-9/REQUIREMENT.md
+ 气泡之间gap:16px
> 新增自 iteration/20260501-2/REQUIREMENT.md
+ 点击右侧CMD小图标，居中展开用户输入CMD命令的浮层，宽度500px，高度110px，输入框内字体字号10px，输入指令后需要二次确认后执行
> 新增自 iteration/20260501-2/REQUIREMENT.md
+ 输入框高度调整为不低于35px，并适当增加上下内边距，要求在不遮挡Tip文案和底部按钮区域的前提下，使输入框视觉占比明显提升
> 新增自 iteration/20260501-2/REQUIREMENT.md
+ 切换到二次确认的浮层和二次确认的浮层返回到输入框保持用户无切换感觉，但需要高亮和闪烁危险提示
> 新增自 iteration/20260501-2/REQUIREMENT.md
+ 二次确认的提示文案与输入框危险提示文案使用同样式和位置
> 新增自 iteration/20260501-2/REQUIREMENT.md
+ 文案与输入框保持5px边距，水平右对齐
> 新增自 iteration/20260501-3/REQUIREMENT.md
+ CLI子面板气泡需求：/20260430-8/REQUIREMENT.md
> 新增自 iteration/20260501-3/REQUIREMENT.md
+ 悬停的判断区域为独立气泡而非内部的文字
> 新增自 iteration/20260501-3/REQUIREMENT.md
+ 独立气泡间间隔3px
> 新增自 iteration/20260501-3/REQUIREMENT.md
+ 点击黏贴小图标，自动复制气泡内的SSE响应至粘贴板，展开点击CMD小图标的浮层，并复制内容到输入框
> 新增自 iteration/20260501-3/REQUIREMENT.md
+ 每条SSE响应气泡的黏贴小图标和黏贴内容必须一一对应、相互独立
> 新增自 iteration/20260501-3/REQUIREMENT.md
+ 点击某条气泡的小图标时，只允许复制当前气泡对应的SSE响应内容
> 新增自 iteration/20260501-3/REQUIREMENT.md
+ 不允许复制相邻气泡内容
> 新增自 iteration/20260502-4/REQUIREMENT.md
+ 已经展示居中对话框中的图片不能因为后端链接下线而无法展示，需要进行浏览器缓存

**编码约束：**
> 新增自 iteration/20260502-5/REQUIREMENT.md
+ 在居中对话框所有的请求和SSE响应气泡右下角（气泡外，与气泡垂直右对齐）增加15x15px的复制图标
> 新增自 iteration/20260502-5/REQUIREMENT.md
+ 复制图标与气泡垂直右对齐（不是水平，在气泡之下），与气泡间隔垂直3px
> 新增自 iteration/20260502-6/REQUIREMENT.md
+ 在居中对话框所有的请求（不要响起SSE）的气泡右下角（气泡外，与复制图标水平右对齐）增加15x15px的重试图标
> 新增自 iteration/20260502-6/REQUIREMENT.md
+ 点击重发图标后复制请求内容，自动发送HTTP请求，同时新增并刷新居中对话框的请求气泡
> 新增自 iteration/20260502-6/REQUIREMENT.md
+ 如果消息正在发送状态，则禁止点击重试，并Tips消息正在发送中
> 修正/扩展自 iteration/20260502-8/REQUIREMENT.md
+ 气泡需求：./iteration/20260419_10/REQUIREMENT.md
> 新增自 iteration/20260502-8/REQUIREMENT.md
+ 在居中对话框输入框或右上角备忘录输入框进行系统粘贴板黏贴操作时，如果黏贴为图片数据，则使用/api/edit为图片在当前Agent的tmp目录下创建随机命名的图片，并在输入框中复制气泡样式的生成图片后的系统绝对路径
> 新增自 iteration/20260502-8/REQUIREMENT.md
+ 气泡样式需要与@文件时一致：
> 新增自 iteration/20260502-8/REQUIREMENT.md
+ 输入框里显示：[FILE:图片文件名]
> 新增自 iteration/20260502-8/REQUIREMENT.md
+ 右上角备忘录的气泡最大宽度不超过输入框，不要把输入框横向撑破
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 点击扇形菜单的按钮，打开居中对话框浮层
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 输入框一行一个，标题左对齐，参数名和参数输入框放一行
> 新增自 iteration/20260504-3/REQUIREMENT.md
+ 点击设置的浮层展示位置需要与扇形菜单的位置对齐（居中对话框水平垂直居中）
> 新增自 iteration/20260504-4/REQUIREMENT.md
+ 点击CLI子面板执行命令的浮层展示位置需要与扇形菜单的位置对齐（居中对话框水平垂直居中）
> 新增自 iteration/20260504-5/REQUIREMENT.md
+ 点击扇形菜单展开后的插件浮层中的日志按钮，浮层中间的参数输入框变为展示SSE流响应，日志按钮变为返回，点击后关闭读取日志的SSE流并转换为输入框
> 新增自 iteration/20260505-2/REQUIREMENT.md
+ 从本地文件系统拖入到居中输入框或左上角备忘录输入框的文件或文件夹（含图片），先上传至当前Agent的tmp目录下
> 新增自 iteration/20260505-2/REQUIREMENT.md
+ 然后使用上传后的文件系统绝对路径作为气泡内容（需要包括原逻辑的[FILE]前缀）
> 新增自 iteration/20260505-2/REQUIREMENT.md
+ 如果是目录，需要上传后保持目录结构，并将目录的文件系统路径作为气泡内容
> 新增自 iteration/20260505-7/REQUIREMENT.md
+ 如果居中输入框没有选择模型，使用覆层提醒：点击输入框左侧小地球选择模型

### 样式、主题与动画
> 新增自 iteration/20260419_10/REQUIREMENT.md
+ 选择文件的图标需要与对话框右侧小文件夹图标风格一致，相关需求：./iteration/20260419_6/REQUIREMENT.md
> 新增自 iteration/20260419_10/REQUIREMENT.md
+ 文件/目录或技能的气泡在左上角有一个10x10像素，透明度50%的小垃圾桶，点击后删除气泡
> 新增自 iteration/20260419_12/REQUIREMENT.md
+ 预览与浮层和对话框使用同色系小色差区分对比，编辑无论在浅色或是还是深色模式都使用白底黑字
> 新增自 iteration/20260419_12/REQUIREMENT.md
+ 浮层需要与对话框有同色系小色差区分对比
> 新增自 iteration/20260419_14/REQUIREMENT.md
+ 对话框用户提问气泡保持当前色系，对比LOGO的颜色，饱和度降低20%
> 新增自 iteration/20260419_2/REQUIREMENT.md
+ 浮层采用透明度50%，色系风格需要与对话框出现错误时的风格保持一致
> 新增自 iteration/20260419_20/REQUIREMENT.md
+ 如果心跳成功则使用当前SVG并进行节奏的闪动
> 新增自 iteration/20260419_20/REQUIREMENT.md
+ 如果心跳失败则`灰色`当前SVG并进行快节奏的闪动
> 新增自 iteration/20260419_20/REQUIREMENT.md
+ 如果执行任务则`绿色`当前SVG并进行超快节奏的闪动，饱和度降低20%
> 新增自 iteration/20260419_20/REQUIREMENT.md
+ 动画效果使用animation而非transition实现，切换状态时使用classList.add/remove保留元素基础class，避免className整体替换导致样式丢失
> 新增自 iteration/20260419_20/REQUIREMENT.md
+ SVG颜色变更使用CSS filter（如hue-rotate）实现，不要用class选择器覆盖内联stroke属性，因为内联属性优先级更高
> 新增自 iteration/20260419_3/REQUIREMENT.md
+ 按钮色系风格需要与侧边栏风格保持一致
> 新增自 iteration/20260419_5/REQUIREMENT.md
+ 小文件夹色系风格需要与设置风格保持一致
> 新增自 iteration/20260419_7/REQUIREMENT.md
+ 保持会话框风格：简洁、干练
> 新增自 iteration/20260421_1/REQUIREMENT.md
+ 首次初始化新会话时展示参考动画（观星者）10秒，然后渐变回现有的操作界面
> 新增自 iteration/20260421_1/REQUIREMENT.md
+ 动画参考实现：algalon_intro.html，中心圆球边缘发亮，中心透明，转速由极快到慢
> 新增自 iteration/20260421_1/REQUIREMENT.md
+ 动画中心文案：修改为`DeepRight`
> 新增自 iteration/20260421_1/REQUIREMENT.md
+ 需要考虑浅色背景和深色背景的配色
> 新增自 iteration/20260421_1/REQUIREMENT.md
+ 浅色背景使用原色系背景，蓝色星辰
> 新增自 iteration/20260421_1/REQUIREMENT.md
+ 深色背景使用原色系背景，白色星辰
> 新增自 iteration/20260421_1/REQUIREMENT.md
+ 动画结束后要有向中心收拢的效果
> 新增自 iteration/20260421_1/REQUIREMENT.md
+ 动画期间禁止切换浅色或深色模式
> 新增自 iteration/20260421_2/REQUIREMENT.md
+ 根据时间自动切换深色模式和浅色模式，每60秒检查一次
> 新增自 iteration/20260421_2/REQUIREMENT.md
+ 浅色模式：系统时间7点-19点
> 新增自 iteration/20260421_2/REQUIREMENT.md
+ 深色模式：系统时间19点-7点
> 新增自 iteration/20260422_1/REQUIREMENT.md
+ 上传文件图标风格与新建、打开一致，外围2px需要锐化发光和中等闪烁
> 新增自 iteration/20260422_2/REQUIREMENT.md
+ 以上图标需要自动调整布局，保证隐藏时不产生错位
> 新增自 iteration/20260422_4/REQUIREMENT.md
+ 小太阳图标需要与当前背景色（浅色或深色）有撞色效果，快速闪动，提示用户可点击
> 新增自 iteration/20260425_1/REQUIREMENT.md
+ 提示框风格参考：./iteration/20260419_2/REQUIREMENT.md
> 新增自 iteration/20260425_2/REQUIREMENT.md
+ 每个Agent的小蜜蜂图标和蜂群配置是独立的
> 新增自 iteration/20260425_2/REQUIREMENT.md
+ 样式参考小太阳图标：./iteration/20260422_4/REQUIREMENT.md
> 新增自 iteration/20260425_2/REQUIREMENT.md
+ 蜂群开关变更后立即检查当前会话的Agent以更新小蜜蜂图标

**编码约束：**
> 新增自 iteration/20260425_4/REQUIREMENT.md
+ Thinking和Auto的切换要有动画效果，使用滑动按钮
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 上半部分占2/3，居中展示可翻阅的日历，靠上对齐，以当前深色或浅色系标记今天，以淡红色标记点击选择的日期
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 上下两部分以一条透明度70%的细线分割
> 新增自 iteration/20260427_3/REQUIREMENT.md
+ 循环小图标需要与当前背景色（浅色或深色）有撞色效果，快速闪动，提示用户可点击，同小太阳需求：./iteration/20260422_4/REQUIREMENT.md
> 新增自 iteration/20260428-10/REQUIREMENT.md
+ 使用浅色字体，字号10px
> 新增自 iteration/20260428-11/REQUIREMENT.md
+ 与文件夹图标垂直对齐
> 新增自 iteration/20260428-9/REQUIREMENT.md
+ 鼠标悬停在备忘录明细列表明细时，在头部展示小垃圾桶图标，8x8像素，透明度50%的小垃圾桶
> 新增自 iteration/20260501-1/REQUIREMENT.md
+ CLI子任务气泡保留科技荧光绿色外，其余背景统一使用浅色模式和深色模式的通用背景色
> 新增自 iteration/20260501-1/REQUIREMENT.md
+ 右侧分割带标题后仅新增一个CMD风格小图标作视觉展示：
> ~~覆盖自 iteration/20260501-2/REQUIREMENT.md，已被 20260501-3 覆盖~~
+ ~~CMD小图标需求：./iteration/20260501-1/REQUIREMENT.md~~
> 修正/扩展自 iteration/20260501-3/REQUIREMENT.md
+ CMD小图标需求：./iteration/20260501-2/REQUIREMENT.md
> 新增自 iteration/20260501-3/REQUIREMENT.md
+ 离开悬停后3秒小图标才消失
> 新增自 iteration/20260501-5/REQUIREMENT.md
+ 黏贴小图标需求：./iteration/20260501-3/REQUIREMENT.md
> 新增自 iteration/20260501-5/REQUIREMENT.md
+ 命令取值逻辑同黏贴小图标：./iteration/20260501-3/REQUIREMENT.md
> 新增自 iteration/20260502-6/REQUIREMENT.md
+ 复制图标需求：./iteration/20260502-5/REQUIREMENT.md
> 新增自 iteration/20260503-2/REQUIREMENT.md
+ 开场动画的中心需要围绕./icon.png，仅使用以图的中心，半径600px的圆
> 新增自 iteration/20260503-2/REQUIREMENT.md
+ 需要有由小变大的闪动效果，最后渐变消失，要震撼、爆发、科幻史诗
> 新增自 iteration/20260503-2/REQUIREMENT.md
+ 动画整体过程控制在8s

**编码约束：**
> 新增自 iteration/20260503-2/REQUIREMENT.md
+ 开场动画中心图标需仅保留icon.png中间logo主体的透明有效区域，去除底图与半透明背景纹理，避免播放时出现额外图层感或非主体方块残影
> 新增自 iteration/20260504-1/REQUIREMENT.md
+ 需要区分浅色模式和深色模式
> 新增自 iteration/20260504-6/REQUIREMENT.md
+ 启动成功则延迟2秒后切换到日志查看（延迟期间所要锁定界面，防止操作，有动画切换渐变）

### 浮层、弹窗与遮罩
> 新增自 iteration/20260419_13/REQUIREMENT.md
+ 所有删除前都需要进行二次确认浮层，参考：./iteration/20260419_3/REQUIREMENT.md
> 新增自 iteration/20260419_16/REQUIREMENT.md
+ 保持浮层的方式，窗口大小自适应
> 新增自 iteration/20260419_2/REQUIREMENT.md
+ 提示采用对话栏水平和垂直居中宽250px,高100px的浮层，文字内容在覆层内垂直水平居中
> 新增自 iteration/20260419_4/REQUIREMENT.md
+ 删除需要二次提醒，风格为浮层，参考：./iteration/20260419_3/REQUIREMENT.md
> ~~覆盖自 iteration/20260419_5/REQUIREMENT.md，已被 20260419_6 覆盖~~
+ ~~打开前需要二次提醒，风格为浮层，参考：./iteration/20260419_3/REQUIREMENT.md~~
> 修正/扩展自 iteration/20260419_6/REQUIREMENT.md
+ 打开前需要二次提醒，风格为浮层，参考：./iteration/20260419_3/REQUIREMENT.md
> 新增自 iteration/20260425_3/REQUIREMENT.md
+ 提示方式同下方浮层：./iteration/20260422_1/REQUIREMENT.md
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 选择小时时需要有淡入淡出效果
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 复用其他组件的CSS class时, 必须检查其所有样式属性(包括布局约束和视觉样式), 确认不会与当前组件冲突, 必要时用更高优先级选择器隔离
> ~~覆盖自 iteration/20260428-11/REQUIREMENT.md，已被 20260428-9 覆盖~~
+ ~~浮层风格参考：./iteration/20260427_4/REQUIREMENT.md~~
> 新增自 iteration/20260428-11/REQUIREMENT.md
+ 浮层标题（标题居中，不需要内容）：确定重载当前会话？
> 新增自 iteration/20260428-11/REQUIREMENT.md
+ 浮层宽度：150px
> 新增自 iteration/20260428-12/REQUIREMENT.md
+ 转圈等待需要有淡入淡出的动画效果
> 新增自 iteration/20260428-13/REQUIREMENT.md
+ 努力工作中的浮层，需要在所有取消完成（包括3s等待）后才收起
> 新增自 iteration/20260428-13/REQUIREMENT.md
+ 取消等待3s的浮层样式同待发送时，但颜色变为淡红色
> 新增自 iteration/20260428-3/REQUIREMENT.md
+ 鼠标悬浮在列表上向左下展开浮层，内容为备忘录内容，移开则自动收起
> 新增自 iteration/20260428-5/REQUIREMENT.md
+ 鼠标悬浮在列表上向左下展开浮层，内容为备忘录元数据，移开则自动收起
> 新增自 iteration/20260428-9/REQUIREMENT.md
+ 点击小垃圾桶图标，需要展开浮层二次确认是否忽略任务，展示除任备忘录内容外的任务明细，
> ~~覆盖自 iteration/20260428-9/REQUIREMENT.md，已被 20260430-2 覆盖~~
+ ~~浮层风格参考：./iteration/20260427_4/REQUIREMENT.md~~
> 修正/扩展自 iteration/20260430-2/REQUIREMENT.md
+ 浮层风格参考：./iteration/20260427_4/REQUIREMENT.md
> 新增自 iteration/20260430-2/REQUIREMENT.md
+ 点击小放大镜图标，需要展开浮层二次确认是否展开会话，展示除任备忘录内容外的任务明细，
> 新增自 iteration/20260501-3/REQUIREMENT.md
+ 黏贴到粘贴板失败也不要阻止打开浮层
> 新增自 iteration/20260501-5/REQUIREMENT.md
+ 浮层样式同备忘录保存浮层：./iteration/20260427_4/REQUIREMENT.md
> 新增自 iteration/20260501-5/REQUIREMENT.md
+ 无论执行成功或失败，都需要收起浮层
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 浮层需求：./iteration/20260504-1/REQUIREMENT.md
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 任何浮层外的操作（点击、拖动等）都要立即收起浮层（要有动画效果）
> 新增自 iteration/20260504-6/REQUIREMENT.md
+ 点击扇形菜单展开后的插件浮层中的启动按钮，立即更新指定插件的配置（meta）
> 新增自 iteration/20260509-1/REQUIREMENT.md
+ 左上角扇形菜单圆按钮:中文插件名最多显示4个字,2/3/4个字自动放大圆按钮保持圆形,中文不强制大写和拉开字距

### X86架构节能渲染开关
> 新增自 ./iteration/20260603-1/REQUIREMENT.md
+ 所有X86的节能或降级渲染功能进行开关标记，所有标记开启关闭仅与节能开关相关，与操作系统无关
+ 仅MAC的X86架构启动时默认开启节能，默认使用浅色模式（并且关闭浅色模式/深色自动切换）

### 全屏切换按钮
> 新增自 ./iteration/20260604-1/REQUIREMENT.md
+ 在居中对话框左上（小地球垂直上方，与SWARM开关保持水平）增加全屏小图标
    + 非全屏时点击进入浏览器全屏（等同MAC FN+F效果），并收起左右侧边栏
    + 全屏时点击退出浏览器全屏，左右侧边栏保持状态

### 未读会话完成提示
> 新增自 ./iteration/20260604-2/REQUIREMENT.md
+ 当有不是当前会话的SSE响应完成（例如用户切到别的会话同时在等待该会话）时，在左侧边栏会话列表该会话的最右侧标记一个2x2px的闪光蓝色小星
    + 如果左侧边栏处于收起状态，则闪动左侧边栏展开的图标，闪动颜色要与当前浅色或深色模式具备反差色，提高醒目度
+ 点击或切换到该会话后，小星消失

### 备忘录任务完成闪烁通知
> 新增自 ./iteration/20260604-3/REQUIREMENT.md
+ 当有备忘录任务完成时，在右侧边栏备忘录任务明细列表整个区域闪烁
    + 如果右侧边栏处于收起状态，则闪动右侧边栏展开的图标，闪动颜色要与当前浅色或深色模式具备反差色，提高醒目度
+ 点击、切换或悬浮到备忘录明细列表后，闪烁消失

### 备忘录一次性任务时间修正
> 新增自 ./iteration/20260606-1/REQUIREMENT.md
+ 当创建备忘录一次性任务时，任务创建时间强制使用当前时间，并且不要提示创建时间晚于当前时间
+ 保证一次性任务必须可创建，如果时间晚于当前时间就使用当前时间

### 连续查询排队机制
> 新增自 ./iteration/20260606-2/REQUIREMENT.md
+ 当在正在执行请求时在居中对话框输入下一个查询并提交：
    + 如果当前执行请求处于等待发送状态（页面锁定，但尚未发送）则将居中对话输入框锁定置灰不可编辑，并立即取消当前请求，自动转而发送下一个查询后解锁居中对话输入框
    + 如果当前执行请求处于等待SSE响应（已连接服务器，等待结果）则将居中对话输入框锁定置灰不可编辑，并在内容左上角标记一个圆形的等待小图标
        + 当上一个SSE响应完全结束，自动转而发送下一个查询，并解锁居中对话输入框（去掉等待小图标）
+ 点击等待小图标则解锁居中对话输入框的锁定，关闭等待小图标，还原为可编辑状态，并且取消自动发送
+ 锁定置灰和等待小图标是Agent加Chat（会话）维度的，在切换会话时需要判断状态并重新渲染效果
+ 默认需要有动画效果，同时如果节能标签如果开启则不需要动画
+ 开始等待发送或取消等待，需要有Tips提示

### 浏览器插件快捷入口
> 新增自 ./iteration/20260606-3/REQUIREMENT.md
+ 如果当前启动了浏览器插件，则在左侧边栏插件展开按钮的右下方展示一个浏览器的小图标，图标3x3像素，实心荧光青绿色（按当前深色和浅色模式区分饱和度）慢速闪烁
+ 点击图标后调用/api/plugins/exec?key=browser&command=instance init&agentId=当前AgentID&chatId=当前会话ID，来加载浏览器CDP
+ 加载后锁定整个界面（左右侧边栏和居中对话框），提示正在进行浏览器插件登录，直到接口返回或报错后解锁
    + Integration需求：../integration/iteration/20260606-1/REQUIREMENT.md
    + Proxy需求：../proxy/iteration/20260606-1/REQUIREMENT.md
+ 在锁定界面的浮层上增加一个完成按钮，点击后调用/api/plugins/exec?key=browser&command=instance shutdown&agentId=当前AgentID&chatId=当前会话ID，来销毁浏览器CDP，解锁界面
+ 锁定界面的/api/plugins/exec不设置超时，持续等待直到用户点击完成
+ 完成按钮需要在锁定界面后30秒才出现

### SSE报文类型爆炸切换动画
> 新增自 ./iteration/20260606-4/REQUIREMENT.md
+ 居中对话框在接收和渲染响应报文时需要通过报文的biz和workflow区分两种类型：
    + 以下为结果报文：biz=main且workflow=base@close；biz=base且workflow=close；结束标记[DONE]
    + 不在此范围都为过程报文
+ SSE响应报文会在两种类型中切换输出
+ 当同类型的报文先后出现时，渲染效果保持现在的叠加。当不同类型的报文先后出现时，第一种类型的报文伴随爆炸效果并清空，重新以第二种类型的报文开始累加
+ 动画爆炸效果的范围需要包括最后一个SSE响应渲染的整个区域
+ 动画先爆炸效果，延迟2秒，加载新内容，需要有节能降级

### 屏保模式
> 新增自 ./iteration/20260606-5/REQUIREMENT.md
+ 超过5分钟没有执行任何任务、页面元素没有任何鼠标悬停（通常为电脑处于待机或无人状态）进入屏保模式，屏保内容为放大版的CLI鲸鱼动画
+ 屏保状态时进入全屏模式，整个屏幕锁定，循环播放鲸鱼动画，当页面出现任何鼠标、键盘事件时屏保以动画形式收起，缓慢收缩到CLI面板
+ 节能模式时，没有屏保

### 文件路径识别与下载快捷操作
> 新增自 ./iteration/20260606-6/REQUIREMENT.md
+ 居中对话框中双击链接时原本会查看是否存在文件，如果有则展示点击预览的小气泡，这个气泡展示名改为预览，功能相同
+ 在这个气泡右侧增加一个平行小气泡，展示名叫下载文件，点击后使用浏览器下载该文件
+ 下载小气泡右侧增加一个平行小气泡，展示名叫打开目录，点击后打开该文件所在的目录
+ 居中对话框渲染SSE响应时，识别所有文件路径（绝对路径和相对路径）并使用蓝色字体加下划线着重强调

### 编写代码
> 新增自 iteration/20260419_18/REQUIREMENT.md
+ 如果范围文字是一个文件相对路径链接，则先在Agent工作目录（workspace）查找文件名，不存在使用绝对路径查找
> 新增自 iteration/20260422_1/REQUIREMENT.md
+ 上传目录时，文件的相对路径不能依赖浏览器的文件名参数传递，需要通过独立的表单字段传递完整相对路径

### 虚拟文件系统
> 新增自 iteration/20260419_12/REQUIREMENT.md
+ 虚拟文件系统的文件可以打开，在右侧对话框水平垂直都居中，宽度为700px，高度为700px，透明度0%的浮层中，以Markdown格式预览或编辑内容
> 新增自 iteration/20260419_12/REQUIREMENT.md
+ 虚拟文件系统的目录可以点击，进入子孙目录
> 新增自 iteration/20260419_12/REQUIREMENT.md
+ 选择Agent后虚拟文件系统需要常驻显示，每10秒刷新一次`当前`展示的路径，仅在有变化时刷新内容
> 新增自 iteration/20260419_12/REQUIREMENT.md
+ 未选择Agent时（如刚初始化）时则不显示虚拟文件系统，但一旦选择并确定则立即显示
> 新增自 iteration/20260419_13/REQUIREMENT.md
+ 虚拟文件系统的文件或目录在水平对齐最后侧展示一个15x15像素的小垃圾桶，点击垃圾桶则删除对应文件或目录
> 新增自 iteration/20260419_16/REQUIREMENT.md
+ 虚拟文件系统的文件预览方式按文件类型（后缀名）区分：
> 新增自 iteration/20260419_16/REQUIREMENT.md
+ HTML：预览方式为浏览器渲染内容，可编辑，编辑后重新刷新渲染
> 新增自 iteration/20260419_16/REQUIREMENT.md
+ 图片：预览方式为浏览器渲染二进制流，不可编辑
> 新增自 iteration/20260419_16/REQUIREMENT.md
+ PDF：使用pdf.js预览
> 新增自 iteration/20260419_16/REQUIREMENT.md
+ HTML预览：iframe需要允许脚本、表单、弹窗等交互
> 新增自 iteration/20260419_16/REQUIREMENT.md
+ PDF预览：使用pdf.js
> 新增自 iteration/20260419_18/REQUIREMENT.md
+ 预览需求：./iteration/20260419_16/REQUIREMENT.md
> 新增自 iteration/20260419_18/REQUIREMENT.md
+ 链接或超链接：用文件编辑器浮层的iframe来预览链接，而不是打开新页面
> 新增自 iteration/20260421_6/REQUIREMENT.md
+ 虚拟文件系统的返回上级图标的水平位置，紧贴右侧靠拢展示2个按钮：新建和打开
> 新增自 iteration/20260422_1/REQUIREMENT.md
+ 虚拟文件系统的新建和打开的右侧添加一个新图标：上传
> 新增自 iteration/20260422_1/REQUIREMENT.md
+ 上传时在虚拟文件系统中`返回上级`的右侧展示上传进度图标直到完成后消失，点击上传进度图标则立即取消上传
> 新增自 iteration/20260422_1/REQUIREMENT.md
+ 上传后在虚拟文件系统最下方展开浮层
> 新增自 iteration/20260422_2/REQUIREMENT.md
+ 虚拟文件系统的`打开目录`
> 新增自 iteration/20260422_3/REQUIREMENT.md
+ 重新打开或切换会话后自动使用最后选择Agent和模型，同时刷新悬浮层和虚拟文件系统
> 新增自 iteration/20260425_3/REQUIREMENT.md
+ 虚拟文件系统需要支持一次选择拖拉多个文件或目录，在点击或选择任一文件或文件夹时提示如何选择多个
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 需要支持虚拟文件系统或实际操作系统的文件或目录拖入对话框 ：./iteration/20260425_3/REQUIREMENT.md
> 新增自 iteration/20260427_2/REQUIREMENT.md
+ 当访问的Host不是localhost或127.0.0.1（本地访问时）禁止向居中对话框和右上的备忘录拖入文件或目录，但允许从虚拟文件系统拖入
> 新增自 iteration/20260428-2/REQUIREMENT.md
+ 虚拟文件系统中小垃圾桶图标后增加一个下载图标，提供文件和目录的下载
> 新增自 iteration/20260428-4/REQUIREMENT.md
+ 通过虚拟文件系统拖入或@选择文件/目录时，气泡需展示完整绝对路径，而非仅文件名
> 新增自 iteration/20260428-4/REQUIREMENT.md
+ 虚拟文件系统拖入需求：./iteration/20260425_3/REQUIREMENT.md
> 新增自 iteration/20260428-4/REQUIREMENT.md
+ 通过操作系统拖入文件时，因浏览器安全策略无法获取绝对路径，需在插入后提示"浏览器安全策略仅展示文件名，建议使用虚拟文件系统"
> 新增自 iteration/20260501-2/REQUIREMENT.md
+ 提示位置在左侧虚拟文件系统的Tips位置，红字提示
> 新增自 iteration/20260501-3/REQUIREMENT.md
+ 复制时在左侧虚拟文件系统的Tips位置，红字提示已复制到粘贴板
> 新增自 iteration/20260501-5/REQUIREMENT.md
+ 执行成功后在左侧虚拟文件系统的Tips位置，红字提示已终止
> 新增自 iteration/20260502-1/REQUIREMENT.md
+ 图片的预览方式修改为可编辑，增加橡皮擦功能，将涂抹的区域都变为透明度为0（Alpha=0）
> 新增自 iteration/20260524-2/REQUIREMENT.md
+ 当原图格式不支持透明通道（如 `jpg/jpeg`）且使用了橡皮擦后，保存结果必须自动转为 `png`，禁止把透明区域压扁回不透明背景
> 新增自 iteration/20260502-1/REQUIREMENT.md
+ 图片预览需求：./iteration/20260419_16/REQUIREMENT.md:17
> 新增自 iteration/20260502-1/REQUIREMENT.md
+ 编辑并保存成功后将图片在文件系统的绝对路径复制到系统粘贴板，然后在虚拟文件系统的Tips位置提示路径已保存至黏贴板
> 新增自 iteration/20260502-2/REQUIREMENT.md
+ 图片的预览方式增加使用不同色笔和线粗（边框粗细）给指定区域画圈
> 新增自 iteration/20260502-3/REQUIREMENT.md
+ 图片的预览方式的修改编辑能力增加后退上N步操作，最多后退10步
> 新增自 iteration/20260502-5/REQUIREMENT.md
+ 点击复制图标后复制气泡内容到系统粘贴板，同时在虚拟文件系统的Tips位置提示黏贴成功
> 新增自 iteration/20260502-6/REQUIREMENT.md
+ 点击重发图标并发送请求成功后，在虚拟文件系统的Tips位置提示重试成功
> 新增自 iteration/20260502-7/REQUIREMENT.md
+ 在左上角创建备忘录的选择模型按钮左侧（紧贴，水平左对齐）增加一个无标题的复选框，选中后在虚拟文件系统的Tips位置提示该任务会复用当前会话（红字）
> 新增自 iteration/20260504-6/REQUIREMENT.md
+ 启动失败则在虚拟文件系统提示Tips启动失败，并延迟2秒后切换到日志查看（需要有动画切换渐变）
> 新增自 iteration/20260505-2/REQUIREMENT.md
+ 虚拟文件系统上传目录：./iteration/20260422_1/REQUIREMENT.md
> 新增自 iteration/20260505-2/REQUIREMENT.md
+ 虚拟文件系统上传需求：
> 新增自 iteration/20260505-5/REQUIREMENT.md
+ 在虚拟文件系统Tips位置提示：消息已恢复，删除会话可重建

### 使用设置
> 新增自 iteration/20260419_12/REQUIREMENT.md
+ 浮层右下展示2个按钮：保存、取消
> 新增自 iteration/20260419_12/REQUIREMENT.md
+ 保存：将修改后的内容保存至指定文件
> 新增自 iteration/20260419_12/REQUIREMENT.md
+ 取消：关闭浮层，点击ESC也可以关闭覆层
> 新增自 iteration/20260419_5/REQUIREMENT.md
+ 设置中`选择 Agent`下拉框的右侧，添加30x30像素，透明度30%的小文件夹图标
> 新增自 iteration/20260419_5/REQUIREMENT.md
+ 点击图标，调用/api/folder?agentId=xxx打开对应目录
> 新增自 iteration/20260419_6/REQUIREMENT.md
+ 小文件夹图标参考：./iteration/20260419_5/REQUIREMENT.md
> 新增自 iteration/20260419_6/REQUIREMENT.md
+ 点击图标，调用/api/folder?agentId=xxx打开当前Agent对应目录
> 新增自 iteration/20260421_3/REQUIREMENT.md
+ 设置面板需要禁止浏览器自动填充和保存密码提示，使用autocomplete="off"和autocomplete="new-password"
> 新增自 iteration/20260421_4/REQUIREMENT.md
+ 删除成功后需要同步清除本地存储中的agentId，避免再次打开设置时下拉列表出现空行
> 新增自 iteration/20260421_5/REQUIREMENT.md
+ 在设置Agent后的图标的都调整为15x15px
> 新增自 iteration/20260422_1/REQUIREMENT.md
+ 上传取消则提示上传取消

**编码约束：**
> 新增自 iteration/20260422_2/REQUIREMENT.md
+ 设置中选择Agent的`打开Agent目录`
> 新增自 iteration/20260422_3/REQUIREMENT.md
+ 会话与会话之间保持独立的Agent和模型，设置里的选择Agent和对话输入框左侧的`选择模型`绑定会话保存
> 新增自 iteration/20260425_1/REQUIREMENT.md
+ 在设置中删除Agent后，如果已经创建的会话使用了该Agent则提示选择Agent
> 新增自 iteration/20260425_1/REQUIREMENT.md
+ 提示框收起和提示时都是为当前会话选择Agent，需要注意实时性
> 新增自 iteration/20260425_2/REQUIREMENT.md
+ 在设置中选择Agent下拉框中Agent名字后增加小蜜蜂图标，点击后展开填写蜂群（swarm）配置
> 新增自 iteration/20260425_2/REQUIREMENT.md
+ 蜂群开关（左对齐）和保存按钮（右对齐）并排，减少空间占用，保存后自动收起蜂群配置
> 新增自 iteration/20260425_2/REQUIREMENT.md
+ 开启蜂群后，设置中当前Agent后的小蜜蜂图标展示色彩，关闭时展示灰色。但无论开启或关闭，每5秒需要展示一层顺时针环绕的光圈，增加提示度
> 新增自 iteration/20260425_2/REQUIREMENT.md
+ 在设置中切换Agent时，需要关闭蜂群配置，每次打开时需要读取实时数据
> 新增自 iteration/20260425_2/REQUIREMENT.md
+ Agent绑定关系变更时（新建会话、切换会话、浮层选择Agent）需要异步检查该Agent的config.json状态并更新蜜蜂图标，不能仅在设置面板操作时更新
> 新增自 iteration/20260425_4/REQUIREMENT.md
+ 在设置中选择Agent后，在紧贴下拉框左侧展开一个Checkbox框，在Thinking和Auto之间切换
> 新增自 iteration/20260425_4/REQUIREMENT.md
+ 在设置中切换Agent时，需要读取实时数据
> 新增自 iteration/20260425_4/REQUIREMENT.md
+ 仅当保存后配置才生效

**编码约束：**
> 新增自 iteration/20260425_4/REQUIREMENT.md
+ Agent绑定关系变更时（新建会话、切换会话、浮层选择Agent）需要异步检查该Agent的config.json状态并更新Thinking图标，不能仅在设置面板操作时更新
> 新增自 iteration/20260426_1/REQUIREMENT.md
+ 在设置更新Thinking配置并保存后，才同时同步Thinking在对话框上的按钮
> 新增自 iteration/20260426_1/REQUIREMENT.md
+ 在会话框按钮更新Thinking配置后，需要保存至当前Agent配置
> 新增自 iteration/20260426_1/REQUIREMENT.md
+ 设置面板内的操作（切换Agent、修改开关）只更新面板内DOM，禁止修改全局状态变量或输入框UI，全局状态仅在保存成功或关闭面板时从当前会话Agent重新加载
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 设置面板内的操作只更新面板内DOM, 全局状态仅在保存或关闭面板时从当前会话Agent重新加载
> 新增自 iteration/20260427_1/REQUIREMENT.md
+ 浮层类组件(下拉框、菜单等): 父容器链不能有overflow:hidden, 宽度不能超出所在容器, 失焦收起时需检查焦点是否转移到浮层内部子元素
> 新增自 iteration/20260427_3/REQUIREMENT.md
+ 在备忘录时间空间与取消（小垃圾桶图标）之间新增一个循环小图标，点击向下展开下拉框浮层，有三个选项：仅一次、工作日、自然日、每小时、每15分钟、每30分钟。默认选择（即使没打开）选择仅一次
> 新增自 iteration/20260427_4/REQUIREMENT.md
+ 点击备忘录的保存按钮，保存数据
> 新增自 iteration/20260428-1/REQUIREMENT.md
+ 在设置中切换Agent后，如果右上的备忘录有任何输入（包括备忘录输入框、备忘录选择模型、备忘录日历、备忘录时间、备忘录思考模式）在点击设置保存后，重置所有备忘录未保存的输入
> 新增自 iteration/20260428-1/REQUIREMENT.md
+ 如果备忘录有任何输入，在设置选择Agent上发展示Tips提示，红字，字号10px，切换Agent并保存会重置未保存备忘录，不保存不重置
> 新增自 iteration/20260428-1/REQUIREMENT.md
+ 提示文案：切换Agent会丢弃尚未保存的备忘录

**编码约束：**
> 新增自 iteration/20260428-11/REQUIREMENT.md
+ 点击确定则重新加载，点击取消或其他任何非浮层位置则自动收起
> 新增自 iteration/20260428-13/REQUIREMENT.md
+ 如果在终止前有任何一条SSE响应被渲染，则标记其为为取消，并更新时间
> 新增自 iteration/20260428-13/REQUIREMENT.md
+ 不允许出现前端已经取消，但下游转发连接仍继续执行的情况
> 新增自 iteration/20260428-14/REQUIREMENT.md
+ 如果无连接且最后响应不是完成或取消，则立即将页面状态切换为等待响应的样式（悬浮努力工作中，发送变为终止），并为该AgentID和Chat（会话ID）每5秒轮询重新加载并渲染最后一条SSE请求时间之后的对话数据（每次都要实时取最后一条），加载完后Tips已加载完成
> 新增自 iteration/20260428-2/REQUIREMENT.md
+ 前端通过JS触发文件下载时，必须使用隐藏iframe方式（设置iframe.src为下载URL），不能使用动态创建<a>标签.click()或window.open，因为浏览器安全策略会拦截非直接用户手势触发的下载和弹窗
> 新增自 iteration/20260428-3/REQUIREMENT.md
+ 删除前需要二次确认，样式保存备忘录的样式：./iteration/20260427_4/REQUIREMENT.md
> 新增自 iteration/20260428-5/REQUIREMENT.md
+ 备忘录列表使用当前会话的AgentID及日历选择日期来获取明细，如果更改了Agent（如调整了设置）或日期（如点击日历其他日期）需要立即刷新
> 新增自 iteration/20260428-5/REQUIREMENT.md
+ 备忘录元数据样式：./iteration/20260427_1/REQUIREMENT.md，但不需要保存和取消按钮
> 新增自 iteration/20260428-6/REQUIREMENT.md
+ 保存成功后立即刷新展示新的任务明细

**编码约束：**
> 新增自 iteration/20260428-7/REQUIREMENT.md
+ 首次打开、切换会话、设置中切换Agent、保存成功备忘录后，需要立即以日历指定日期和当前会话AgentId刷新备忘录明细列表，如果同时正在展示任务元数据列表那么也需要立即刷新
> 新增自 iteration/20260428-7/REQUIREMENT.md
+ 保存备忘录需求：./iteration/20260427_4/REQUIREMENT.md
> 新增自 iteration/20260428-8/REQUIREMENT.md
+ Grid布局的子元素（如田字格的四个格子）默认min-height:auto，内容过多时会撑破格子高度而非触发滚动。需要在grid容器及其子元素上设置min-height:0，确保overflow-y:auto生效，内容在固定高度内滚动
> 新增自 iteration/20260430-6/REQUIREMENT.md
+ 当前会话没有绑定Agent需要选择Agent时的浮层下拉框仅有一个可选项时，需要同时监听click事件作为补充，确保单选项场景下选择后也能正常绑定Agent并关闭浮层
> 新增自 iteration/20260502-2/REQUIREMENT.md
+ 保存方式同：./iteration/20260502-1/REQUIREMENT.md
> 新增自 iteration/20260502-7/REQUIREMENT.md
+ 如果开启了复选框，在保存浮层时增加当前提示（红字）：继续使用当前会话
> 新增自 iteration/20260502-7/REQUIREMENT.md
+ 备忘录保存需求：./iteration/20260427_4/REQUIREMENT.md
> 新增自 iteration/20260503-1/REQUIREMENT.md
+ 设置中的模型与密钥数据不在存储在页面，改为使用/api/token保存或获取
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 第一部分（展示在插件声明下3px）：是否复用会话（ChatID）、选择Agent、选择模型、思考模式
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 选择模型后下拉框需要勾选后收起
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 插件配置必须统一以key作为唯一标识，打开插件浮层时强制重新请求/api/plugins/meta回填最新meta，并保证保存后关闭重开及刷新页面都能稳定显示已填参数
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 第三部分：展示：重启、关闭、日志、取消，四个按钮，点击浮层或按ESC则自动取消
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 三个按钮右下对齐，离开底边5px，风格与设置中按钮相同


### 迭代 20260525-1
> 新增自 ./iteration/20260525-1/REQUIREMENT.md
+ 修正Site会话区SSE流式响应的Markdown渲染时机，禁止仅在响应结束后一次性做Markdown转换
+ 在流式过程中，每次收到新的SSE `delta.content` 后，都必须基于“当前完整累计文本”立即执行一次可见渲染，让用户实时看到已成形的标题、列表、引用、代码块、表格、链接、图片、HTML嵌入等结构，而不是长时间看到 `#`、`*`、``` 之类原始符号
+ 流式Markdown渲染必须兼容“中间态未闭合”的内容，例如未闭合代码块、未闭合列表、未闭合表格、未闭合HTML片段；中间态允许按降级样式展示，但不得出现整段内容持续以纯转义文本形式闪烁到结束
+ 流式渲染与最终完成态渲染必须使用同一套渲染链路和同一份安全策略，禁止流式阶段走纯文本、结束阶段再走另一套Markdown/HTML处理，避免前后展示结果不一致
+ Markdown渲染需要继续遵守site现有能力边界，保持对标准Markdown、LaTeX、图片、安全HTML嵌入的兼容，不得破坏既有消息展示、历史会话恢复、惰性加载、复制、停止生成、自动滚动等能力
+ 流式Markdown渲染前必须先对累计文本做轻量归一化，清理 ANSI/OSC 控制序列、`\r` 回车、零宽字符及其他会破坏行首 Markdown 识别的隐藏控制字符，避免 `#### 正在加载技能 weather`、引用、列表、序号等结构因为脏前缀而长期显示为原始符号
+ SSE消息的聚合逻辑必须稳定：
+ 同一条assistant消息在流式过程中只维护一个增量中的消息实体
+ 每次只在原有内容尾部追加新的 `delta.content`
+ 不得因重复重渲染导致消息拆分、重复插入、错序、跳动或丢字
+ 自动滚动逻辑需要与流式Markdown渲染兼容：
+ 当用户位于底部附近时，收到新内容后自动滚动到底部
+ 当用户主动上滑查看历史时，不得强制抢回滚动位置
+ 流式渲染期间如果用户点击暂停或请求异常结束，页面应保留当前已经成功收到并渲染的内容，不得回退为纯文本或清空本轮assistant消息
+ 对于包含代码块、表格、图片、HTML iframe/video 等可能引发布局抖动的内容，渲染层必须控制重排频率，避免每个字符级更新都造成明显卡顿；可以使用轻量节流、批量刷新或成熟开源流式Markdown方案，但最终效果必须是“流式过程中持续可读”
+ 如果实时 Markdown 渲染阶段仍发生异常，页面不得静默吞错；至少需要保留当前文本可见展示，并输出统一前缀的诊断日志，便于定位是哪类增量内容触发了降级路径
+ 如果引入开源包，必须满足：
+ 包体积尽可能小
+ 许可证可用于当前项目
+ 所有前端依赖资源必须可随站点一起发布
+ 发布链路必须校验资源存在性和正确MIME，禁止引用了未发布资源
+ 本次需求同时补强会话区浮层与流式区的交互隔离：
+ “努力工作中”浮层、停止按钮、错误提示、复制按钮、HTML预览、图片预览等不得因为固定定位层级或透明遮罩影响左右Sidebar操作
+ 会话区新增或调整的浮层必须接入统一浮层管理，不得在业务里直接散落控制 `z-index`、`pointer-events`、`left/top`
+ 隐藏态浮层必须彻底失活，不能残留点击拦截
+ 本次需求禁止改写integration的二进制收口边界，site仍必须作为integration统一静态站点能力的一部分被发布和访问，不能新增独立站点启动方式、独立资源目录协议或脱离integration的构建入口

### 迭代 20260525-2
> 新增自 ./iteration/20260525-2/REQUIREMENT.md
+ 在居中对话输入框是否开启HTML开关的左侧增加SWARM（蜂群）开关，与HTML开关相同是Agent+Chat（会话）绑定，切换会话时需要切换状态
    + 在居中对话框开启蜂群模式（SWARM）并发起对话时，同时写入body.metadata.router_disable = false，关闭时写入body.metadata.router_disable = true
        + 开启SWARM（蜂群）时router_disable=false，关闭时router_disable=true
        + false对应蜂群开启，true对应关闭
    + 转发/v1/chat/completions时需要带上metadata.router_disable = true/false
    ```
    {
        "metadata": {
        ...
        "router_disable": true
        }
    }
    ```

### 迭代 20260525-3
> 新增自 ./iteration/20260525-3/REQUIREMENT.md
+ 在右上角备忘录元数据思考模式开关的左侧增加SWARM（蜂群）开关，在创建时作为router_disable参数传入，默认true，关闭
+ 开启SWARM（蜂群）时router_disable=false，关闭时router_disable=true
    + Integration需求：../integration/iteration/20260524-3/REQUIREMENT.md
    + Proxy需求：../proxy/iteration/20260524-3/REQUIREMENT.md
    + Cron需求：../cron/iteration/20260524-1/REQUIREMENT.md
+ 备忘录任务创建时，右上角SWARM开关与实际转发/v1/chat/completions的metadata.router_disable必须全链路一致
    + 映射规则固定为：
        + 开启SWARM 时，router_disable=false
        + 关闭SWARM 时，router_disable=true

### 迭代 20260525-4
> 新增自 ./iteration/20260525-4/REQUIREMENT.md
+ 在插件展开的浮层右上角思考模式开关的左侧增加SWARM（蜂群）开关，在HTTP POST `/api/plugins/config`作为router_disable参数传入，默认true，关闭
+ 开启SWARM（蜂群）时router_disable=false，关闭时router_disable=true
    + Integration需求：../integration/iteration/20260524-3/REQUIREMENT.md
    + Proxy需求：../proxy/iteration/20260524-3/REQUIREMENT.md
    + Cron需求：../cron/iteration/20260524-1/REQUIREMENT.md

### 迭代 20260525-5
> 新增自 ./iteration/20260525-5/REQUIREMENT.md
+ 设置中蜂群开关参数swarm改为router_disable，类型不变，语意相反（router_disable=true表示关闭）
+ 开启SWARM（蜂群）时router_disable=false，关闭时router_disable=true
+ 页面展示开关名称不变
    + HTTP /api/edit中swarm改为router_disable，默认为true，语意相反
        + Proxy需求：../proxy/iteration/20260524-5/REQUIREMENT.md

### 迭代 20260525-6
> 新增自 ./iteration/20260525-6/REQUIREMENT.md
+ 首次使用（无任何浏览器记录）时增加选择Agent和配置模型的新手引导任务
    + 在`请选择 Agent`处增加新手引导的遮罩，高亮选择框，并除选择 Agent外暗淡除其他部分并不可点击
        + 新手引导文案：新手可以从DEF_AGENT开始。
    + 选择完Agent后，右侧菜单动画展开，在设置按钮处增加新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：点击设置，配置需要使用的模型。
    + 展开设置后，在模型和密钥处增加新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：选择模型，并输入密钥。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到模型客户化配置，高亮按钮，并暗淡其他部分并不可点击
    + 展开客户化配置后，在快速响应处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：快速响应模型（选填）：系统会自动将简单问题调度至此模型，以节约Token。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到深度思考，高亮按钮，并暗淡其他部分并不可点击
    + 在深度思考处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：深度思考模型（选填）：复杂规划问题调度至此模型，以获得更好效果。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到多模态输入，高亮按钮，并暗淡其他部分并不可点击
    + 在多模态输入处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：多模态输入模型（选填）：图片、PDF等多模态识别会调度至此模型。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到多模态输出，高亮按钮，并暗淡其他部分并不可点击
    + 在多模态输出处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：多模态输出模型（选填）：图片、视频等多模态生成会调度至此模型。
        + 提供一个`知道了`的新手引导按钮，点击后切换到新建Agent按钮的新手引导任务，高亮按钮，并暗淡其他部分并不可点击
    + 在新建Agent处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：点击此处新建Agent，配置独立的USER、SOUL和SKILL。
        + 提供一个`知道了`的新手引导按钮，点击后切换到虚拟文件系统的新手引导任务，高亮按钮，并暗淡其他部分并不可点击
    + 在整个虚拟文件系统处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：每个Agent都有自己的虚拟文件系统。
        + 提供一个`知道了`的新手引导按钮，点击后切换到虚拟文件系统新增文件或目录的新手引导任务，高亮按钮，并暗淡其他部分并不可点击
    + 在虚拟文件系统新增文件或目录处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：点击为Agent新增文件或目录。
        + 提供一个`知道了`的新手引导按钮，点击后切换到虚拟文件系统上传的新手引导任务，高亮按钮，并暗淡其他部分并不可点击
    + 在虚拟文件系统上传处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：点击上传或将直接将文件目录拖入虚拟文件系统。
        + 提供一个`完成`的新手引导按钮，点击后关闭新手引导任务的遮罩，所有功能可点击

### 迭代 20260525-7
> 新增自 ./iteration/20260525-7/REQUIREMENT.md
+ 首次点击左侧新建会话或居中对话输入框时（无任何点击输入框记录）时增加引导对话输入的新手引导任务
    + 在居中对话输入框左侧选择模型（小地球）增加新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：选择已经配置的模型。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到发送按钮，高亮按钮，并暗淡其他部分并不可点击
    + 在发送按钮处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：发送消息。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到切换回居中对话输入框的区域，高亮按钮，并暗淡其他部分并不可点击
    + 在居中对话输入框的区域处新手引导的遮罩，模拟@输入，弹起文件和技能菜单并高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：输入@来引入文件或强调需要使用的技能。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到整个居中对话框渲染响应的区域，高亮按钮，并暗淡其他部分并不可点击
    + 在居中对话框渲染响应的区域处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：等待响应结果。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到CLI子面板，高亮按钮，并暗淡其他部分并不可点击
    + 在切换到整个CLI子面板（右侧边栏第二栏）区域处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：执行的系统命令会在此显示。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到居中对话框的SWARN开关，高亮按钮，并暗淡其他部分并不可点击
    + 在SWARN区域处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：开启蜂群模式，会自动调用其他蜂群Agent。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到居中对话框的HTML区域，高亮按钮，并暗淡其他部分并不可点击
    + 在HTML区域处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：切换Markdown或HTML输出格式以获得更好视觉体验。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到居中对话框的思考模式区域，高亮按钮，并暗淡其他部分并不可点击
    + 在思考模式区域处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：深度思考模式。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到居中对话框的右侧刷新会话区域，高亮按钮，并暗淡其他部分并不可点击
    + 在刷新会话区域处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：刷新因网络中断的会话内容。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到左侧新建新会话区域，高亮按钮（如果收起需要自动展开），并暗淡其他部分并不可点击
    + 在左侧新建新会话处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：新建会话开启独立上下文。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到左侧会话列表区域，高亮按钮，并暗淡其他部分并不可点击
    + 在左侧会话列表处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：切换会话探讨不同话题。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到会话列表第一个会话删除按钮，高亮按钮，并暗淡其他部分并不可点击
    + 在会话删除按钮处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：删除不需要的会话。
        + 提供一个`完成`的新手引导按钮，点击后关闭新手引导任务的遮罩，所有功能可点击
+ 与首次使用（无任何浏览器记录）时新手引导相互独立，并且需要兼容
    + 首次使用需求：./iteration/20260525-6/REQUIREMENT.md

### 迭代 20260525-8
> 新增自 ./iteration/20260525-8/REQUIREMENT.md
+ 首次点击设置中Agent的蜂群小图标，（无任何点击蜂群开关的记录）时开启引导对话输入的新手引导任务
    + 在设置中Agent的蜂群开关增加新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：开启蜂群，让其他Agent可以自动调用。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到蜂群的思考模式，高亮按钮，并暗淡其他部分并不可点击
    + 在蜂群的思考模式增加新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：选择蜂群被调用时的思考模式。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到蜂群的模型选择，高亮按钮，并暗淡其他部分并不可点击
    + 在蜂群的模型选择增加新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：选择蜂群被调用时使用的模型。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到蜂群的描述，高亮按钮，并暗淡其他部分并不可点击
    + 在蜂群的描述增加新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：输入Agent详细的作用和描述，需要的输入信息和预计的输出内容，让蜂群调用更准确。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到蜂群的确认，高亮按钮，并暗淡其他部分并不可点击
    + 在蜂群的确认增加新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：配置完后需要点击确认。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到设置的保存，高亮按钮，并暗淡其他部分并不可点击
    + 在设置的保存增加新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：保存后让蜂群生效。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到设置中Agent右侧的蜂群小图标，开启闪动（蜂群开启时效果），高亮按钮，并暗淡其他部分并不可点击
    + 在蜂群小图标加新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：开启后，蜂群图标会进行闪动。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到居中对话框的蜂群小图标，高亮按钮，并暗淡其他部分并不可点击
    + 在居中对话框的蜂群小图标加新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：开启后，蜂群图标会进行闪动。
        + 提供一个`完成`的新手引导按钮，点击后关闭新手引导任务的遮罩，所有功能可点击（包括检查设置中因为新手引导点亮的蜂群小图标）
+ 与首次使用（无任何浏览器记录）时新手引导相互独立，并且需要兼容
    + 首次使用需求：./iteration/20260525-6/REQUIREMENT.md
+ 首次点击居中对话输入框时（无任何点击输入框记录）时增加引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击居中对话输入框需求：./iteration/20260525-7/REQUIREMENT.md

### 迭代 20260526-1
> 新增自 ./iteration/20260526-1/REQUIREMENT.md
+ 首次点击左侧边栏插件按钮（无任何点击插件按钮的记录）并且当前有可用插件时开启引导对话输入的新手引导任务
    + 在展开左侧边栏的扇形插件增加新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：插件会提供额外感知能力。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到蜂群的思考模式，高亮按钮，并暗淡其他部分并不可点击
    + 在展开后第一个插件图标处增加新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：选择需要启动的插件。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到插件浮层，高亮按钮，并暗淡其他部分并不可点击
    + 在插件浮层增加新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：填写需要提供的配置。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到插件的启动按钮，高亮按钮，并暗淡其他部分并不可点击
    + 在插件的启动按钮处新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：启动插件并接受外部感知。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到插件的日志按钮，高亮按钮，并暗淡其他部分并不可点击
    + 在插件的日志增加新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：可以通过日志查看插件状态。
        + 提供一个`完成`的新手引导按钮，点击后关闭新手引导任务的遮罩，所有功能可点击（包括检查设置中因为新手引导点亮的蜂群小图标）
+ 与首次使用（无任何浏览器记录）时新手引导相互独立，并且需要兼容
    + 首次使用需求：./iteration/20260525-6/REQUIREMENT.md
+ 首次点击居中对话输入框时（无任何点击输入框记录）时增加引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击居中对话输入框需求：./iteration/20260525-7/REQUIREMENT.md
+ 首次点击设置中Agent的蜂群小图标，（无任何点击蜂群开关的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击居中对话输入框需求：./iteration/20260525-8/REQUIREMENT.md

### 迭代 20260526-2
> 新增自 ./iteration/20260526-2/REQUIREMENT.md
+ 首次点击虚拟文件系统SOUL.md或USER.md的任意位置，如查看或编辑文件、小灯泡图标等（无任何点击SOUL.md或USER.md任意位置记录）时开启引导对话输入的新手引导任务
    + 在SOUL.md这整个文件增加新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：SOUL.md记录Agent的身份、性格、行为风格，每次对话后自动更新。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到SOUL.md的小灯泡按钮，高亮按钮，并暗淡其他部分并不可点击
    + 在SOUL.md的小灯泡按钮处增加新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：也可以主动整理。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到USER.md这整个文件，高亮按钮，并暗淡其他部分并不可点击
    + 在USER.md这整个文件增加新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：USER.md记录用户的姓名、称呼、时区、备注，每次对话后自动更新。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到SOUL.md的小灯泡按钮，高亮按钮，并暗淡其他部分并不可点击
    + 在SOUL.md的小灯泡按钮处新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：也可以主动整理。
        + 提供一个`完成`的新手引导按钮，点击后关闭新手引导任务的遮罩，所有功能可点击（包括检查设置中因为新手引导点亮的蜂群小图标）
+ 如果SOUL.md不存在则直接进入USER.md的新手引导，如果USER.md不存在则完成SOUL.md后结束新手引导
+ 与首次使用（无任何浏览器记录）时新手引导相互独立，并且需要兼容
    + 首次使用需求：./iteration/20260525-6/REQUIREMENT.md
+ 首次点击居中对话输入框时（无任何点击输入框记录）时增加引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击居中对话输入框需求：./iteration/20260525-7/REQUIREMENT.md
+ 首次点击设置中Agent的蜂群小图标，（无任何点击蜂群开关的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击设置中Agent的蜂群小图标：./iteration/20260525-8/REQUIREMENT.md
+ 首次点击左侧边栏插件按钮（无任何点击插件按钮的记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击左侧边栏插件按钮需求：./iteration/20260526-1/REQUIREMENT.md

### 迭代 20260526-3
> 新增自 ./iteration/20260526-3/REQUIREMENT.md
+ 首次点击右侧边栏的第一栏任何元素时（备忘录日历、明细或元数据）（无任何点击备忘录日历、明细或元数据任意位置记录）开启引导对话输入的新手引导任务
    + 在备忘录元数据（右侧边栏第一栏右侧）整个区域增加新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：备忘录可以制定延迟或周期任务。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到SOUL.md的小灯泡按钮，高亮按钮，并暗淡其他部分并不可点击
    + 在备忘录元数据的使用当前会话按钮处增加新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：勾选后共享当前会话上下文至备忘录任务。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到选择模型按钮，高亮按钮，并暗淡其他部分并不可点击
    + 在备忘录元数据选择模型按钮增加新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：选择备忘录需要使用的模型。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到SWARM和思考模式按钮（2个一起），高亮按钮，并暗淡其他部分并不可点击
    + 在备忘录SWARM和思考模式按钮处新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：为备忘录开启蜂群或选择思考模式。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到时间选择区域，高亮按钮，并暗淡其他部分并不可点击
     + 在备忘录时间选择区域处新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
         + 新手引导文案：选择延迟或周期任务的执行时间。
         + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到周期模式，高亮下拉框，并暗淡其他部分并不可点击
     + 在备忘录周期模式下拉框处新手引导的遮罩，展开下拉框，高亮仅一次选择框，并暗淡其他部分并不可点击
         + 新手引导文案：使用仅一次开启延迟任务。
         + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到仅一次外的所有选项，高亮下拉框，并暗淡其他部分并不可点击
     + 在备忘录周期模式下拉框仅一次外的所有选项处新手引导的遮罩，展开下拉框，高亮选择框，并暗淡其他部分并不可点击
         + 新手引导文案：选择其他选项开启周期任务。
         + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换信封小图标所有选项，高亮按钮，并暗淡其他部分并不可点击
     + 在备忘录信封小图标处新手引导的遮罩，展开下拉框，高亮选择框，并暗淡其他部分并不可点击
         + 新手引导文案：创建任务。
         + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到展示备忘录元数据列表按钮，高亮按钮，并暗淡其他部分并不可点击
     + 在备忘录展开列表按钮处新手引导的遮罩，展开下拉框，高亮选择框，并暗淡其他部分并不可点击
         + 新手引导文案：查看已经创建的周期任务。
         + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到备忘录周期任务列表，高亮任务列表，并暗淡其他部分并不可点击
     + 在备忘录展开列表处新手引导的遮罩，展开下拉框，里模拟一条"测试任务",高亮任务列表，并暗淡其他部分并不可点击
         + 新手引导文案：查看已经创建的周期任务。
         + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到备忘录模拟的"测试任务"删除按钮，高亮按钮，并暗淡其他部分并不可点击
     + 备忘录模拟的"测试任务"删除按钮处新手引导的遮罩,高亮按钮，并暗淡其他部分并不可点击
         + 新手引导文案：点击删除周期任务。
         + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到备忘录明细列表（并删除模拟"测试任务），高亮列表，并暗淡其他部分并不可点击
     + 备忘录明细列表处新手引导的遮罩,高亮按钮，模拟1条"已完成测试任务明细"，1条"待执行测试任务明细",高亮并已完成测试任务明细，暗淡其他部分并不可点击
         + 新手引导文案：查看已经完成的任务内容。
         + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到待执行测试任务明细的小垃圾桶图标，高亮图标，并暗淡其他部分并不可点击
     + 在备忘录任务明细的小垃圾桶图标处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
         + 新手引导文案：终止尚未执行的任务。
         + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到待执行测试任务明细的放大镜图标，高亮图标，并暗淡其他部分并不可点击
     + 在备忘录任务明细的放大镜图标处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
         + 新手引导文案：在对话框中查看过程。
         + 提供一个`完成`的新手引导按钮，点击后关闭新手引导任务的遮罩，所有功能可点击（包括检查设置中因为新手引导模拟的数据）
+ 与首次使用（无任何浏览器记录）时新手引导相互独立，并且需要兼容
    + 首次使用需求：./iteration/20260525-6/REQUIREMENT.md
+ 首次点击居中对话输入框时（无任何点击输入框记录）时增加引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击居中对话输入框需求：./iteration/20260525-7/REQUIREMENT.md
+ 首次点击设置中Agent的蜂群小图标，（无任何点击蜂群开关的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击设置中Agent的蜂群小图标：./iteration/20260525-8/REQUIREMENT.md
+ 首次点击左侧边栏插件按钮（无任何点击插件按钮的记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击左侧边栏插件按钮需求：./iteration/20260526-1/REQUIREMENT.md
+ 首次点击虚拟文件系统SOUL.md或USER.md的任意位置，如查看或编辑文件、小灯泡图标等（无任何点击SOUL.md或USER.md任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务任务相互独立，并且需要兼容
    + 首次点击左侧边栏插件按钮需求：./iteration/20260526-2/REQUIREMENT.md

### 迭代 20260526-4
> 新增自 ./iteration/20260526-4/REQUIREMENT.md
+ 鼠标悬停左上角备忘录元数据列表任意元数据时，在居中对话框水平垂直居中位置展示浮层，内容为该备忘录元数据的内容和配置
    + 同时模糊化除该浮层外居中对话框的所有背景
    + 鼠标离开时浮层自动消失，需要有动画效果
+ 鼠标悬停左上角备忘录任务明细列表任意任务明细数据时，在居中对话框水平垂直居中位置展示浮层，内容为该备忘录任务明细数据的内容和配置
    + 同时模糊化除该浮层外居中对话框的所有背景
    + 鼠标离开时浮层自动消失，需要有动画效果

### 迭代 20260526-5
> 新增自 ./iteration/20260526-5/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务
    + 在skills目录加新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：存放Agent技能目录。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到创建SKILL按钮，高亮按钮，并暗淡其他部分并不可点击
    + 在创建SKILL按钮处增加新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：成功经验也可以变成技能。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到创建技能的浮层，高亮浮层，并暗淡其他部分并不可点击
    + 在创建技能的浮层增加新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：选择上下文范围来创建技能。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到提炼最近N轮对话及其输入框（整体），高亮区域，并暗淡其他部分并不可点击
    + 在提炼最近N轮对话及其输入框（整体）处新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：每次提问和回答是一轮对话。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到提炼Skill的开始时间和结束时间（整体），高亮区域，并暗淡其他部分并不可点击
    + 在提炼Skill的开始时间和结束时间（整体）处新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：或选择期望的时间范围。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到提炼Skill目标浮层的提炼目标，高亮区域，并暗淡其他部分并不可点击
    + 在提炼Skill目标浮层的提炼目标处新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：精准的提炼目标是技能质量的关键。
        + 提供一个`完成`的新手引导按钮，点击后关闭新手引导任务的遮罩，所有功能可点击
+ 与首次使用（无任何浏览器记录）时新手引导相互独立，并且需要兼容
    + 首次使用需求：./iteration/20260525-6/REQUIREMENT.md
+ 首次点击居中对话输入框时（无任何点击输入框记录）时增加引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击居中对话输入框需求：./iteration/20260525-7/REQUIREMENT.md
+ 首次点击设置中Agent的蜂群小图标，（无任何点击蜂群开关的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击设置中Agent的蜂群小图标：./iteration/20260525-8/REQUIREMENT.md
+ 首次点击左侧边栏插件按钮（无任何点击插件按钮的记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击左侧边栏插件按钮需求：./iteration/20260526-1/REQUIREMENT.md
+ 首次点击虚拟文件系统SOUL.md或USER.md的任意位置，如查看或编辑文件、小灯泡图标等（无任何点击SOUL.md或USER.md任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统SOUL.md或USER.md的任意位置需求：./iteration/20260526-2/REQUIREMENT.md
+ 首次点击右侧边栏的第一栏任何元素时（备忘录日历、明细或元数据）（无任何点击备忘录日历、明细或元数据任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-3/REQUIREMENT.md

### 迭代 20260526-6
> 新增自 ./iteration/20260526-6/REQUIREMENT.md
+ 右侧边栏CLI子面板展示CLI命令的区域，如果当前为空（从未执行）则展示Chrome小飞鸟循环飞翔的动画

### 迭代 20260526-7
> 新增自 ./iteration/20260526-7/REQUIREMENT.md
+ 首次点击右侧边栏CLI子面板时（无任何点击右侧边栏CLI子面板任何元素的记录）时开启引导对话输入的新手引导任务
    + 在CLI子面板命令展示区域加新手引导的遮罩，在CLI命令展示区域模拟一条ls / ，高亮命令，并暗淡其他部分并不可点击
        + 新手引导文案：系统执行的命令会在这里显示。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到模拟CLI命令的终止按钮，高亮按钮，并暗淡其他部分并不可点击
    + 在模拟CLI命令的终止按钮处增加新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：终止预期外的系统命令。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到模拟CLI命令的复制按钮，高亮按钮，并暗淡其他部分并不可点击
    + 在模拟CLI命令的复制按钮增加新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：重复执行系统命令。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到CLI子面板执行执行命令的按钮（同时清除模拟CLI命令），高亮区域，并暗淡其他部分并不可点击
    + 在CLI子面板执行执行命令的按钮处新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：执行自定义系统命令。
        + 提供一个`完成`的新手引导按钮，点击后关闭新手引导任务的遮罩，所有功能可点击
+ 与首次使用（无任何浏览器记录）时新手引导相互独立，并且需要兼容
    + 首次使用需求：./iteration/20260525-6/REQUIREMENT.md
+ 首次点击居中对话输入框时（无任何点击输入框记录）时增加引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击居中对话输入框需求：./iteration/20260525-7/REQUIREMENT.md
+ 首次点击设置中Agent的蜂群小图标，（无任何点击蜂群开关的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击设置中Agent的蜂群小图标：./iteration/20260525-8/REQUIREMENT.md
+ 首次点击左侧边栏插件按钮（无任何点击插件按钮的记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击左侧边栏插件按钮需求：./iteration/20260526-1/REQUIREMENT.md
+ 首次点击虚拟文件系统SOUL.md或USER.md的任意位置，如查看或编辑文件、小灯泡图标等（无任何点击SOUL.md或USER.md任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统SOUL.md或USER.md的任意位置需求：./iteration/20260526-2/REQUIREMENT.md
+ 首次点击右侧边栏的第一栏任何元素时（备忘录日历、明细或元数据）（无任何点击备忘录日历、明细或元数据任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-3/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统skills目录或创建SKILL按钮需求：./iteration/20260526-5/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-7/REQUIREMENT.md

### 迭代 20260526-8
> 新增自 ./iteration/20260526-8/REQUIREMENT.md
+ 首次点击右侧边栏知识库任意元素时（无任何点击右侧边栏知识库任意元素的记录）时开启引导对话输入的新手引导任务
    + 在右侧边栏知识库整个区域加新手引导的遮罩 ，高亮区域，并暗淡其他部分并不可点击
        + 新手引导文案：使用时积累的知识会被自动归档为WIKI。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到展开知识库全屏按钮，高亮按钮，并暗淡其他部分并不可点击
    + 在展开知识库全屏按钮处增加新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：如果知识库内容丰富，可以点击展开。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到整理知识库按钮，高亮按钮，并暗淡其他部分并不可点击
    + 在整理知识库增加新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：也可以主动整理。
        + 提供一个`完成`的新手引导按钮，点击后关闭新手引导任务的遮罩，所有功能可点击
+ 与首次使用（无任何浏览器记录）时新手引导相互独立，并且需要兼容
    + 首次使用需求：./iteration/20260525-6/REQUIREMENT.md
+ 首次点击居中对话输入框时（无任何点击输入框记录）时增加引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击居中对话输入框需求：./iteration/20260525-7/REQUIREMENT.md
+ 首次点击设置中Agent的蜂群小图标，（无任何点击蜂群开关的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击设置中Agent的蜂群小图标：./iteration/20260525-8/REQUIREMENT.md
+ 首次点击左侧边栏插件按钮（无任何点击插件按钮的记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击左侧边栏插件按钮需求：./iteration/20260526-1/REQUIREMENT.md
+ 首次点击虚拟文件系统SOUL.md或USER.md的任意位置，如查看或编辑文件、小灯泡图标等（无任何点击SOUL.md或USER.md任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统SOUL.md或USER.md的任意位置需求：./iteration/20260526-2/REQUIREMENT.md
+ 首次点击右侧边栏的第一栏任何元素时（备忘录日历、明细或元数据）（无任何点击备忘录日历、明细或元数据任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-3/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统skills目录或创建SKILL按钮需求：./iteration/20260526-5/REQUIREMENT.md
+ 首次点击右侧边栏CLI子面板时（无任何点击右侧边栏CLI子面板任何元素的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-7/REQUIREMENT.md

### 迭代 20260526-9
> 新增自 ./iteration/20260526-9/REQUIREMENT.md
+ 首次点击浏览器插件重启按钮时（无任何点击浏览器插件重启按钮的记录）时开启引导对话输入的新手引导任务
    + 在浏览器插件重启按钮加新手引导的遮罩 ，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：浏览器插件内置Obscura内核[https://github.com/h4ckf0r0day/obscura]，与Chrome共享Cookie。
        + 提供一个`完成`的新手引导按钮，点击后关闭新手引导任务的遮罩，所有功能可点击
+ 与首次使用（无任何浏览器记录）时新手引导相互独立，并且需要兼容
    + 首次使用需求：./iteration/20260525-6/REQUIREMENT.md
+ 首次点击居中对话输入框时（无任何点击输入框记录）时增加引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击居中对话输入框需求：./iteration/20260525-7/REQUIREMENT.md
+ 首次点击设置中Agent的蜂群小图标，（无任何点击蜂群开关的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击设置中Agent的蜂群小图标：./iteration/20260525-8/REQUIREMENT.md
+ 首次点击左侧边栏插件按钮（无任何点击插件按钮的记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击左侧边栏插件按钮需求：./iteration/20260526-1/REQUIREMENT.md
+ 首次点击虚拟文件系统SOUL.md或USER.md的任意位置，如查看或编辑文件、小灯泡图标等（无任何点击SOUL.md或USER.md任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统SOUL.md或USER.md的任意位置需求：./iteration/20260526-2/REQUIREMENT.md
+ 首次点击右侧边栏的第一栏任何元素时（备忘录日历、明细或元数据）（无任何点击备忘录日历、明细或元数据任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-3/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统skills目录或创建SKILL按钮需求：./iteration/20260526-5/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-7/REQUIREMENT.md
+ 首次点击右侧边栏知识库任意元素时（无任何点击右侧边栏知识库任意元素的记录）时开启引导对话输入的新手引导任务
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-8/REQUIREMENT.md

### 迭代 20260526-10
> 新增自 ./iteration/20260526-10/REQUIREMENT.md
+ 首次点击远程插件重启按钮时（无任何点击远程插件重启按钮的记录）时开启引导对话输入的新手引导任务
    + 在远程插件重启按钮加新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：远程插件提供SSH操控远程主机的能力。
        + 提供一个`完成`的新手引导按钮，点击后关闭新手引导任务的遮罩，所有功能可点击
+ 与首次使用（无任何浏览器记录）时新手引导相互独立，并且需要兼容
    + 首次使用需求：./iteration/20260525-6/REQUIREMENT.md
+ 首次点击居中对话输入框时（无任何点击输入框记录）时增加引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击居中对话输入框需求：./iteration/20260525-7/REQUIREMENT.md
+ 首次点击设置中Agent的蜂群小图标，（无任何点击蜂群开关的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击设置中Agent的蜂群小图标：./iteration/20260525-8/REQUIREMENT.md
+ 首次点击左侧边栏插件按钮（无任何点击插件按钮的记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击左侧边栏插件按钮需求：./iteration/20260526-1/REQUIREMENT.md
+ 首次点击虚拟文件系统SOUL.md或USER.md的任意位置，如查看或编辑文件、小灯泡图标等（无任何点击SOUL.md或USER.md任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统SOUL.md或USER.md的任意位置需求：./iteration/20260526-2/REQUIREMENT.md
+ 首次点击右侧边栏的第一栏任何元素时（备忘录日历、明细或元数据）（无任何点击备忘录日历、明细或元数据任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-3/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统skills目录或创建SKILL按钮需求：./iteration/20260526-5/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-7/REQUIREMENT.md
+ 首次点击右侧边栏知识库任意元素时（无任何点击右侧边栏知识库任意元素的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-8/REQUIREMENT.md
+ 首次点击浏览器插件重启按钮时（无任何点击浏览器插件重启按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-9/REQUIREMENT.md

### 迭代 20260526-11
> 新增自 ./iteration/20260526-11/REQUIREMENT.md
+ 首次左侧边栏，最近右侧的小灯泡按钮时（无任何点击最近右侧的小灯泡按钮的记录）时开启引导对话输入的新手引导任务
    + 在最近右侧的小灯泡按钮加新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
        + 新手引导文案：如果缺失重要的本地应用，会提示进行安装。
        + 提供一个`完成`的新手引导按钮，点击后关闭新手引导任务的遮罩，所有功能可点击
+ 与首次使用（无任何浏览器记录）时新手引导相互独立，并且需要兼容
    + 首次使用需求：./iteration/20260525-6/REQUIREMENT.md
+ 首次点击居中对话输入框时（无任何点击输入框记录）时增加引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击居中对话输入框需求：./iteration/20260525-7/REQUIREMENT.md
+ 首次点击设置中Agent的蜂群小图标，（无任何点击蜂群开关的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击设置中Agent的蜂群小图标：./iteration/20260525-8/REQUIREMENT.md
+ 首次点击左侧边栏插件按钮（无任何点击插件按钮的记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击左侧边栏插件按钮需求：./iteration/20260526-1/REQUIREMENT.md
+ 首次点击虚拟文件系统SOUL.md或USER.md的任意位置，如查看或编辑文件、小灯泡图标等（无任何点击SOUL.md或USER.md任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统SOUL.md或USER.md的任意位置需求：./iteration/20260526-2/REQUIREMENT.md
+ 首次点击右侧边栏的第一栏任何元素时（备忘录日历、明细或元数据）（无任何点击备忘录日历、明细或元数据任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-3/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统skills目录或创建SKILL按钮需求：./iteration/20260526-5/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-7/REQUIREMENT.md
+ 首次点击右侧边栏知识库任意元素时（无任何点击右侧边栏知识库任意元素的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-8/REQUIREMENT.md
+ 首次点击浏览器插件重启按钮时（无任何点击浏览器插件重启按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-9/REQUIREMENT.md
+ 首次点击浏览器插件重启按钮时（无任何点击浏览器插件重启按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-10/REQUIREMENT.md

### 迭代 20260527-1
> 新增自 ./iteration/20260527-1/REQUIREMENT.md
+ 点击左侧边栏最左上的`logoSvg`，在居中对话框展示Token用量的浮层，浮层需要可以输入开始时间、结束时间、最大记录条数（都是选填，默认开始时间为7天前，结束时间为此时此刻，最大记录条数默认500，上限也是500）
    + API: /api/consume
    + Proxy需求：../proxy/iteration/20260527-1/REQUIREMENT.md
    + Integration需求：../integration/iteration/20260527-1/REQUIREMENT.md
+ 浮层展开时需要使用默认时间进行首次查询，每次查询开始时都需要锁定界面直到数据查询完毕
+ 查询结果需要分2部分展示：
    + 第一部分：按AgentId汇总的数据（AgentId以响应数据为准，即使当前Agent已经被删除也需要展示）
        + 按比例展示（显示在一行）
            + THINKING = THINKING / TOTAL（百分比，小数点最多2位）
            + INPUT = INPUT / TOTAL（百分比，小数点最多2位）
            + CACHE = CACHE / TOTAL（百分比，小数点最多2位）
            + TOTAL（原始值除以1000，以k结尾为单位）
        + 数据较多时，展开时需要支持滚动条
        + 需要提供足够的可视高度
    + 第二部分：按AgentId + 模型维度汇总数据（显示在一行），明细数据可能很多，需要滚动而不能撑破覆层
        + 格式同第一部分
        + 数据较多时，展开时需要支持滚动条
        + 需要提供足够的可视高度
    + 第三部分：每条日志明细（默认收起以让出展示空间给第一第二部分，点击后展开后第一第二部分折叠起）
        + 原始数据，不需要转为小数，一行一条，以表格形式
        + 数据较多时，展开时需要支持滚动条
        + 需要提供足够的可视高度
    + 底部标记本次查询的最大记录条数（不要被遮挡），需要提供足够的可视高度
+ 浮层展开期间背景模糊，点击其他任何非浮层位置则关闭浮层

### 迭代 20260527-2
> 新增自 ./iteration/20260527-2/REQUIREMENT.md
+ 在新手引导`在居中对话框渲染响应的区域处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击`和`在切换到整个CLI子面板（右侧边栏第二栏）区域处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击`修改为：
    +（原流程）在居中对话框渲染响应的区域处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
           + 新手引导文案：等待响应结果。
           + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到左侧边栏的`logoSvg`，高亮按钮，并暗淡其他部分并不可点击
    +（新流程）在左侧边栏的`logoSvg`处新手引导的遮罩，高亮按钮，并暗淡其他部分并不可点击
           + 新手引导文案：查看所消耗的Token明细
           + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换到整个CLI子面板（右侧边栏第二栏）区域，高亮区域，并暗淡其他部分并不可点击
    +（原流程）在CLI子面板执行执行命令的按钮处新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
+ 新手引导需求：./iteration/20260525-7/REQUIREMENT.md
+ Token消耗需求：./iteration/20260527-1/REQUIREMENT.md

### 迭代 20260527-3
> 新增自 ./iteration/20260527-3/REQUIREMENT.md
+ 由`整理 USER.md`或`整理 SOUL.md`浮层触发的请求时写入body.metadata.profile_commit = true
+ 转发/v1/chat/completions时需要带上metadata.profile_commit = true
```
{
    "metadata": {
    ...
    "profile_commit": true
    }
}
```

### 迭代 20260527-4
> 新增自 ./iteration/20260527-4/REQUIREMENT.md
+ 左侧边栏`logoSvg`在无任务状态时，变为报表小图标（注意搭配深色和浅色背景）配合慢速闪动效果，开始执行任务时切换回原样
+ Token消费的触发浮层需要同时兼容2种图标：./iteration/20260527-1/REQUIREMENT.md

### 迭代 20260527-5
> 新增自 ./iteration/20260527-5/REQUIREMENT.md
+ 在设置中增加新模型配置：
    + xiaomi：
        + __url：https://api.xiaomimimo.com/v1/chat/completions
        + __model：mimo-v2-flash
        + __model_fast：mimo-v2-flash
        + __model_thinking：mimo-v2.5-pro
        + __model_multi_input：mimo-v2.5
        + __model_multi_output：没有，不支持配置
+ 模型配置：./iteration/20260520-1/REQUIREMENT.md

### 迭代 20260529-1
> 新增自 ./iteration/20260529-1/REQUIREMENT.md
+ 设置中模型与密钥，选择模型deepright时需要隐藏客户化配置图标（五角星）
+ 如果将已配置模型切换到deepright，需要隐藏客户化配置图标
+ 如果将已配置模型切换到deepright，保存时需要清空原本配置的：模型URL、基础模型、快速响应模型、深度思考模型、多模态输入模型、多模态输出模型
    + 对已有模型的的切换是修改，不是新增
    + 仅在点击保存后真实生效

### 迭代 20260529-2
> 新增自 ./iteration/20260529-2/REQUIREMENT.md
+ 设置中模型与密钥的客户化配置中，如果鼠标悬停在基础模型、快速响应、深度思考、多模态输入、多模态输出时，如果输入框内容大于输入框宽度（有文字被遮挡）则在下方悬浮输入内容
    + 悬浮的输入框内容宽度等于实际文字宽度，需要展示全文字
    + 鼠标移开后悬浮消失

### 迭代 20260530-1
> 新增自 ./iteration/20260530-1/REQUIREMENT.md
+ 在右侧边栏知识库的整理知识库WIKI左侧增加一个自动整理的开关，默认为关闭
    + 开关对应参数属性为knowledge_disable，开关开启时knowledge_disable=false，关闭时knowledge_disable=true（开false，关true）
    + 默认为开关开启状态（knowledge_disable=false）
+ 该开关是全局的，与Agent和Chat（当前会话）不绑定
+ 转发/v1/chat/completions时需要带上metadata.knowledge_disable = true/false，并一路传递至--host指定的最终处理服务器
```
{
  "metadata": {
  ...
  "knowledge_disable": true
  }
}
```
+ 本需求只新增了knowledge_disable属性，不要删除或修改其他属性
+ 需要通过原有的所有测试

### 迭代 20260530-2
> 新增自 ./iteration/20260530-2/REQUIREMENT.md
+ 首次点击右侧边栏知识库的自动整理开关（无任何点击右侧边栏知识库的自动整理开关的记录）时开启引导对话输入的新手引导任务
    + 在自动整理开关处新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：知识库会自动整理，点击关闭后切换为手动模式。
        + 提供一个`完成`的新手引导按钮，点击后关闭新手引导任务的遮罩，所有功能可点击
+ 与首次使用（无任何浏览器记录）时新手引导相互独立，并且需要兼容
    + 首次使用需求：./iteration/20260525-6/REQUIREMENT.md
+ 首次点击居中对话输入框时（无任何点击输入框记录）时增加引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击居中对话输入框需求：./iteration/20260525-7/REQUIREMENT.md
+ 首次点击设置中Agent的蜂群小图标，（无任何点击蜂群开关的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击设置中Agent的蜂群小图标：./iteration/20260525-8/REQUIREMENT.md
+ 首次点击左侧边栏插件按钮（无任何点击插件按钮的记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击左侧边栏插件按钮需求：./iteration/20260526-1/REQUIREMENT.md
+ 首次点击虚拟文件系统SOUL.md或USER.md的任意位置，如查看或编辑文件、小灯泡图标等（无任何点击SOUL.md或USER.md任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统SOUL.md或USER.md的任意位置需求：./iteration/20260526-2/REQUIREMENT.md
+ 首次点击右侧边栏的第一栏任何元素时（备忘录日历、明细或元数据）（无任何点击备忘录日历、明细或元数据任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-3/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统skills目录或创建SKILL按钮需求：./iteration/20260526-5/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-7/REQUIREMENT.md
+ 首次点击右侧边栏知识库任意元素时（无任何点击右侧边栏知识库任意元素的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-8/REQUIREMENT.md
+ 首次点击浏览器插件重启按钮时（无任何点击浏览器插件重启按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-9/REQUIREMENT.md
+ 首次点击浏览器插件重启按钮时（无任何点击浏览器插件重启按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-10/REQUIREMENT.md

### 迭代 20260530-3
> 新增自 ./iteration/20260530-3/REQUIREMENT.md
+ 首次点击设置模型与密钥中任意模型客户化配置的模型URL（无任何点击客户化配置的模型URL记录）时开启引导对话输入的新手引导任务
    + 在自动整理开关处新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：注意数据安全，谨慎使用三方代理。
        + 提供一个`完成`的新手引导按钮，点击后关闭新手引导任务的遮罩，所有功能可点击
+ 与首次使用（无任何浏览器记录）时新手引导相互独立，并且需要兼容
    + 首次使用需求：./iteration/20260525-6/REQUIREMENT.md
+ 首次点击居中对话输入框时（无任何点击输入框记录）时增加引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击居中对话输入框需求：./iteration/20260525-7/REQUIREMENT.md
+ 首次点击设置中Agent的蜂群小图标，（无任何点击蜂群开关的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击设置中Agent的蜂群小图标：./iteration/20260525-8/REQUIREMENT.md
+ 首次点击左侧边栏插件按钮（无任何点击插件按钮的记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击左侧边栏插件按钮需求：./iteration/20260526-1/REQUIREMENT.md
+ 首次点击虚拟文件系统SOUL.md或USER.md的任意位置，如查看或编辑文件、小灯泡图标等（无任何点击SOUL.md或USER.md任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统SOUL.md或USER.md的任意位置需求：./iteration/20260526-2/REQUIREMENT.md
+ 首次点击右侧边栏的第一栏任何元素时（备忘录日历、明细或元数据）（无任何点击备忘录日历、明细或元数据任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-3/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统skills目录或创建SKILL按钮需求：./iteration/20260526-5/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-7/REQUIREMENT.md
+ 首次点击右侧边栏知识库任意元素时（无任何点击右侧边栏知识库任意元素的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-8/REQUIREMENT.md
+ 首次点击浏览器插件重启按钮时（无任何点击浏览器插件重启按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-9/REQUIREMENT.md
+ 首次点击浏览器插件重启按钮时（无任何点击浏览器插件重启按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-10/REQUIREMENT.md
+ 首次点击浏览器插件重启按钮时（无任何点击浏览器插件重启按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-10/REQUIREMENT.md
+ 首次点击右侧边栏知识库的自动整理开关（无任何点击右侧边栏知识库的自动整理开关的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260530-2/REQUIREMENT.md

### 迭代 20260530-4
> 新增自 ./iteration/20260530-4/REQUIREMENT.md
+ 在设置中`设置`文案后增加一个点击后在粘贴板复制deviceId的复制小图标，低频转动
    + DeviceId需求：../agent/REQUIREMENT.md
+ 点击后将deviceId复制到系统粘贴板，并在虚拟文件系统的Tips位置提示：注意唯一ID的安全性
+ 在选择 Agent蜂群开关后增加一个R的小图标，和蜂群同频转动，点开后在蜂群配置的位置展示添加外部设备连接
    + 每行仅一个输入框，限制长度50字符，每个输入框需要有小眼睛展示和显示内容
    + 可以有多个输入框（多行），滚动展示，高度不超过蜂群配置的高度
    + 如果再次点击R的小图标则自动收起，如果点击蜂群配置则自动切换
    + 同蜂群配置一样，有一个确认按钮，点击确认后收起
+ 点击设置的保存后同其他配置一起保存并持久化到数据库
+ 如果该配置存在值，则所有转发/v1/chat/completions时需要带上metadata.router_remote = [第一个值,第二个值,...]
```
{
    "metadata": {
    ...
    "router_remote": ["remote_1","remote_2"]
    }
}
```
    + 包括居中对话框、备忘任务明细（包括通过插件生成的备忘录任务明细）

### 迭代 20260530-5
> 新增自 ./iteration/20260530-5/REQUIREMENT.md
+ 首次点击设置中deviceId复制小图标或选择 Agent蜂群的R小图标（无任何点击复制小图标或R小图标记录）时开启引导对话输入的新手引导任务
    + 在R小图标处新手引导的遮罩，高亮选择框，并暗淡其他部分并不可点击
        + 新手引导文案：开启蜂群远程Agent。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换（展开外部设备连接），高亮区域，并暗淡其他部分并不可点击
    + 在外部设备连接处新手引导的遮罩，高亮区域，并暗淡其他部分并不可点击
        + 新手引导文案：填写远程Agent的唯一标识。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换外部设备连接确认小图标，高亮按钮，并暗淡其他部分并不可点击
    + 在外部设备连接确认小图标处新手引导的遮罩，高亮区域，并暗淡其他部分并不可点击
        + 新手引导文案：点击确认。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换设置的保存，高亮按钮，并暗淡其他部分并不可点击
    + 在设置的保存小图标处新手引导的遮罩，高亮区域，并暗淡其他部分并不可点击
        + 新手引导文案：保存后让蜂群生效。
        + 提供一个`知道了`的新手引导按钮，点击后新手引导的遮罩切换deviceId复制小图标，高亮按钮，并暗淡其他部分并不可点击
    + 在deviceId复制小图标处新手引导的遮罩，高亮区域，并暗淡其他部分并不可点击
        + 新手引导文案：远程Agent点击此处获取唯一标识。
        + 提供一个`完成`的新手引导按钮，点击后关闭新手引导任务的遮罩，所有功能可点击
+ 与首次使用（无任何浏览器记录）时新手引导相互独立，并且需要兼容
    + 首次使用需求：./iteration/20260525-6/REQUIREMENT.md
+ 首次点击居中对话输入框时（无任何点击输入框记录）时增加引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击居中对话输入框需求：./iteration/20260525-7/REQUIREMENT.md
+ 首次点击设置中Agent的蜂群小图标，（无任何点击蜂群开关的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击设置中Agent的蜂群小图标：./iteration/20260525-8/REQUIREMENT.md
+ 首次点击左侧边栏插件按钮（无任何点击插件按钮的记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击左侧边栏插件按钮需求：./iteration/20260526-1/REQUIREMENT.md
+ 首次点击虚拟文件系统SOUL.md或USER.md的任意位置，如查看或编辑文件、小灯泡图标等（无任何点击SOUL.md或USER.md任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统SOUL.md或USER.md的任意位置需求：./iteration/20260526-2/REQUIREMENT.md
+ 首次点击右侧边栏的第一栏任何元素时（备忘录日历、明细或元数据）（无任何点击备忘录日历、明细或元数据任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-3/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统skills目录或创建SKILL按钮需求：./iteration/20260526-5/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-7/REQUIREMENT.md
+ 首次点击右侧边栏知识库任意元素时（无任何点击右侧边栏知识库任意元素的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-8/REQUIREMENT.md
+ 首次点击浏览器插件重启按钮时（无任何点击浏览器插件重启按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-9/REQUIREMENT.md
+ 首次点击浏览器插件重启按钮时（无任何点击浏览器插件重启按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-10/REQUIREMENT.md
+ 首次点击浏览器插件重启按钮时（无任何点击浏览器插件重启按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-10/REQUIREMENT.md
+ 首次点击右侧边栏知识库的自动整理开关（无任何点击右侧边栏知识库的自动整理开关的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库的自动整理需求：./iteration/20260530-2/REQUIREMENT.md
+ 首次点击设置模型与密钥中任意模型客户化配置的模型URL（无任何点击客户化配置的模型URL记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击设置模型与密钥中任意模型客户化配置的模型URL需求：./iteration/20260530-3/REQUIREMENT.md

### 迭代 20260530-6
> 新增自 ./iteration/20260530-6/REQUIREMENT.md
+ /install_app接口每60秒自动刷新一次

### 迭代 20260530-7
> 新增自 ./iteration/20260530-7/REQUIREMENT.md
+ 在左侧边栏的深色/浅色模式水平位置，仅贴侧边栏最右侧内，新增一个节能开关（默认关闭）
+ 开启后：
    + 关闭开屏动画
    + 关掉欢迎页无限动画
    + 毛玻璃改成普通半透明背景
    + 降低日常使用的持续消耗
        + pageTaskSchedulerStep改为15秒1次
            + heartbeat 改成 30s
            + vfs 改成 30s
            + wiki / quickPlugins 改成 60s
+ 仅作为开关，和bool判断，不要破坏现有代码

### 迭代 20260531-1
> 新增自 ./iteration/20260531-1/REQUIREMENT.md
+ X86架构启动时默认开启节能，默认使用浅色模式（并且关闭浅色模式/深色自动切换）

### 迭代 20260613-6
> 新增自 ./iteration/20260613-6/REQUIREMENT.md
+ 在居中输入框或备忘录输入框首次通过@唤起__internal_cron技能并且当前选择模型已经配置了密钥（无任何居中输入框或备忘录输入框首次通过@唤起__internal_cron技能并且当前选择模型已经配置了密钥记录）时开启新手引导任务
    + 在居中对话输入框增加新手引导，询问用户是否需要了解技能使用方法（是绑定回车，否绑定ESC），并暗淡其他部分并不可点击
        + 新手引导文案：需要了解__internal_cron如何使用吗？
    + 点击是使用动画为用户自动在居中对话框输入：给我几个用自然语言描述[SKILL:__internal_cron]的案例
        + [SKILL:__internal_cron]在居中对话框输入时的动画需要使用气泡样式
    + 动画最后为用户自动发送，关闭新手引导任务的遮罩，所有功能可点击
+ 首次使用（无任何浏览器记录）时新手引导相互独立，并且需要兼容
    + 首次使用需求：./iteration/20260525-6/REQUIREMENT.md
+ 首次点击居中对话输入框时（无任何点击输入框记录）时增加引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击居中对话输入框需求：./iteration/20260525-7/REQUIREMENT.md
+ 首次点击设置中Agent的蜂群小图标，（无任何点击蜂群开关的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击设置中Agent的蜂群小图标：./iteration/20260525-8/REQUIREMENT.md
+ 首次点击左侧边栏插件按钮（无任何点击插件按钮的记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击左侧边栏插件按钮需求：./iteration/20260526-1/REQUIREMENT.md
+ 首次点击虚拟文件系统SOUL.md或USER.md的任意位置，如查看或编辑文件、小灯泡图标等（无任何点击SOUL.md或USER.md任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统SOUL.md或USER.md的任意位置需求：./iteration/20260526-2/REQUIREMENT.md
+ 首次点击右侧边栏的第一栏任何元素时（备忘录日历、明细或元数据）（无任何点击备忘录日历、明细或元数据任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-3/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统skills目录或创建SKILL按钮需求：./iteration/20260526-5/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-7/REQUIREMENT.md
+ 首次点击右侧边栏知识库任意元素时（无任何点击右侧边栏知识库任意元素的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-8/REQUIREMENT.md
+ 首次点击浏览器插件重启按钮时（无任何点击浏览器插件重启按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-9/REQUIREMENT.md
+ 首次点击浏览器插件重启按钮时（无任何点击浏览器插件重启按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-10/REQUIREMENT.md
+ 首次点击设置，任意模型的客户化配置（无任何点击客户化配置的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击设置，任意模型的客户化配置需求：./iteration/20260607-1/REQUIREMENT.md
+ 首次出现任意模型客户化配置按钮（五角星）闪烁（无任何客户化配置按钮（五角星）闪烁的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次出现任意模型客户化配置按钮（五角星）闪烁需求：./iteration/20260608-4/REQUIREMENT.md
+ 首次配置完模型密钥并保存后（无任何配置完模型密钥并保存后的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次配置完模型密钥并保存需求：./iteration/20260608-5/REQUIREMENT.md
+ 首次鼠标点击、悬停在备忘录元数据输入框且已完成备忘录新手引导且已配置任意模型（无任何鼠标点击、悬停在备忘录元数据输入框的记录且已完成备忘录新手引导且已配置任意模型）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次鼠标点击、悬停在备忘录元数据输入框且已完成备忘录新手引导且已配置任意模型需求：./iteration/20260609-2/REQUIREMENT.md
+ 首次备忘录任务执行完成（状态为已完成）且当前备忘录明细列表仅此一条（包括其他状态的）（无任何备忘录任务执行完成（状态为已完成）的记录）且当前备忘录明细列表仅此一条（包括其他状态的）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次备忘录任务执行完成（状态为已完成）且当前备忘录明细列表仅此一条（包括其他状态的）需求：./iteration/20260609-3/REQUIREMENT.md
+ 首次点击沙盒按钮（无任何点击沙盒按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击沙盒按钮需求：./iteration/20260609-4/REQUIREMENT.md
+ 首次点击全屏按钮（无任何点击全屏按钮的记录）时开启新手引导任务相互独立，并且需要兼容
    + 首次点击全屏按钮需求：./iteration/20260609-5/REQUIREMENT.md
+ 首次点击虚拟文件系统新建按钮或鼠标首次悬浮在虚拟文件系统除SOUL.md和USER.md外任意可编辑文件（无任何点击虚拟文件系统新建按钮或鼠标首次悬浮在虚拟文件系统除SOUL.md和USER.md外任意可编辑文件的记录）时开启新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统新建按钮或鼠标首次悬浮在虚拟文件系统除SOUL.md和USER.md外任意可编辑文件需求：./iteration/20260610-3/REQUIREMENT.md
+ 首次出现跨会话完成提醒（会话有一个小×，或因为左侧边栏收起时左上收起菜单闪动）（无任何跨会话完成提醒（会话有一个小×，或因为左侧边栏收起时左上收起菜单闪动）的记录）时开启新手引导任务相互独立，并且需要兼容
    + 首次出现跨会话完成提醒（会话有一个小×，或因为左侧边栏收起时左上收起菜单闪动）需求：./iteration/20260610-4/REQUIREMENT.md
+ 首次成功启动任意插件后（无任何成功启动任意插件的记录）时开启新手引导任务相互独立，并且需要兼容
    + 首次成功启动任意插件需求：./iteration/20260611-1/REQUIREMENT.md
+ 在居中对话框存在正确且结束的SSE响应后，首次在居中对话输入框中切换到不同模型（无任何居中对话框存在正确且结束的SSE响应后，在居中对话输入框中切换到不同模型记录）时开启新手引导任务相互独立，并且需要兼容
    + 在居中对话框存在正确且结束的SSE响应后，首次在居中对话输入框中切换到不同模型需求：./iteration/20260611-7/REQUIREMENT.md
+ 设置中首次成功开启并保存蜂群配置（无任何设置中成功开启并保存蜂群配置时记录）时开启新手引导任务相互独立，并且需要兼容
    + 设置中首次成功开启并保存蜂群配置需求：./iteration/20260611-8/REQUIREMENT.md

### 迭代 20260613-7
> 新增自 ./iteration/20260613-7/REQUIREMENT.md
+ 在居中输入框或备忘录输入框首次通过@唤起__internal_browser技能并且当前选择模型已经配置了密钥（无任何居中输入框或备忘录输入框首次通过@唤起__internal_browser技能并且当前选择模型已经配置了密钥记录）时开启新手引导任务
    + 在居中对话输入框增加新手引导，询问用户是否需要了解技能使用方法（是绑定回车，否绑定ESC），并暗淡其他部分并不可点击
        + 新手引导文案：需要了解__internal_browser如何使用吗？
    + 点击是使用动画为用户自动在居中对话框输入：给我几个用自然语言描述[SKILL:__internal_browser]的案例
        + [SKILL:__internal_browser]在居中对话框输入时的动画需要使用气泡样式
    + 动画最后为用户自动发送，关闭新手引导任务的遮罩，所有功能可点击
+ 首次使用（无任何浏览器记录）时新手引导相互独立，并且需要兼容
    + 首次使用需求：./iteration/20260525-6/REQUIREMENT.md
+ 首次点击居中对话输入框时（无任何点击输入框记录）时增加引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击居中对话输入框需求：./iteration/20260525-7/REQUIREMENT.md
+ 首次点击设置中Agent的蜂群小图标，（无任何点击蜂群开关的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击设置中Agent的蜂群小图标：./iteration/20260525-8/REQUIREMENT.md
+ 首次点击左侧边栏插件按钮（无任何点击插件按钮的记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击左侧边栏插件按钮需求：./iteration/20260526-1/REQUIREMENT.md
+ 首次点击虚拟文件系统SOUL.md或USER.md的任意位置，如查看或编辑文件、小灯泡图标等（无任何点击SOUL.md或USER.md任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统SOUL.md或USER.md的任意位置需求：./iteration/20260526-2/REQUIREMENT.md
+ 首次点击右侧边栏的第一栏任何元素时（备忘录日历、明细或元数据）（无任何点击备忘录日历、明细或元数据任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-3/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统skills目录或创建SKILL按钮需求：./iteration/20260526-5/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-7/REQUIREMENT.md
+ 首次点击右侧边栏知识库任意元素时（无任何点击右侧边栏知识库任意元素的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-8/REQUIREMENT.md
+ 首次点击浏览器插件重启按钮时（无任何点击浏览器插件重启按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-9/REQUIREMENT.md
+ 首次点击浏览器插件重启按钮时（无任何点击浏览器插件重启按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-10/REQUIREMENT.md
+ 首次点击设置，任意模型的客户化配置（无任何点击客户化配置的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击设置，任意模型的客户化配置需求：./iteration/20260607-1/REQUIREMENT.md
+ 首次出现任意模型客户化配置按钮（五角星）闪烁（无任何客户化配置按钮（五角星）闪烁的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次出现任意模型客户化配置按钮（五角星）闪烁需求：./iteration/20260608-4/REQUIREMENT.md
+ 首次配置完模型密钥并保存后（无任何配置完模型密钥并保存后的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次配置完模型密钥并保存需求：./iteration/20260608-5/REQUIREMENT.md
+ 首次鼠标点击、悬停在备忘录元数据输入框且已完成备忘录新手引导且已配置任意模型（无任何鼠标点击、悬停在备忘录元数据输入框的记录且已完成备忘录新手引导且已配置任意模型）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次鼠标点击、悬停在备忘录元数据输入框且已完成备忘录新手引导且已配置任意模型需求：./iteration/20260609-2/REQUIREMENT.md
+ 首次备忘录任务执行完成（状态为已完成）且当前备忘录明细列表仅此一条（包括其他状态的）（无任何备忘录任务执行完成（状态为已完成）的记录）且当前备忘录明细列表仅此一条（包括其他状态的）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次备忘录任务执行完成（状态为已完成）且当前备忘录明细列表仅此一条（包括其他状态的）需求：./iteration/20260609-3/REQUIREMENT.md
+ 首次点击沙盒按钮（无任何点击沙盒按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击沙盒按钮需求：./iteration/20260609-4/REQUIREMENT.md
+ 首次点击全屏按钮（无任何点击全屏按钮的记录）时开启新手引导任务相互独立，并且需要兼容
    + 首次点击全屏按钮需求：./iteration/20260609-5/REQUIREMENT.md
+ 首次点击虚拟文件系统新建按钮或鼠标首次悬浮在虚拟文件系统除SOUL.md和USER.md外任意可编辑文件（无任何点击虚拟文件系统新建按钮或鼠标首次悬浮在虚拟文件系统除SOUL.md和USER.md外任意可编辑文件的记录）时开启新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统新建按钮或鼠标首次悬浮在虚拟文件系统除SOUL.md和USER.md外任意可编辑文件需求：./iteration/20260610-3/REQUIREMENT.md
+ 首次出现跨会话完成提醒（会话有一个小×，或因为左侧边栏收起时左上收起菜单闪动）（无任何跨会话完成提醒（会话有一个小×，或因为左侧边栏收起时左上收起菜单闪动）的记录）时开启新手引导任务相互独立，并且需要兼容
    + 首次出现跨会话完成提醒（会话有一个小×，或因为左侧边栏收起时左上收起菜单闪动）需求：./iteration/20260610-4/REQUIREMENT.md
+ 首次成功启动任意插件后（无任何成功启动任意插件的记录）时开启新手引导任务相互独立，并且需要兼容
    + 首次成功启动任意插件需求：./iteration/20260611-1/REQUIREMENT.md
+ 在居中对话框存在正确且结束的SSE响应后，首次在居中对话输入框中切换到不同模型（无任何居中对话框存在正确且结束的SSE响应后，在居中对话输入框中切换到不同模型记录）时开启新手引导任务相互独立，并且需要兼容
    + 在居中对话框存在正确且结束的SSE响应后，首次在居中对话输入框中切换到不同模型需求：./iteration/20260611-7/REQUIREMENT.md
+ 设置中首次成功开启并保存蜂群配置（无任何设置中成功开启并保存蜂群配置时记录）时开启新手引导任务相互独立，并且需要兼容
    + 设置中首次成功开启并保存蜂群配置需求：./iteration/20260611-8/REQUIREMENT.md

### 迭代 20260614-1
> 新增自 ./iteration/20260614-1/REQUIREMENT.md
+ 在设置，新增Agent小图标后增加导入和导出小图标
    + 导出：点击后执行/api/agent/export?agent_id=xxx，导出当前Agent配置，并自动使用浏览器下载
        + 需要提示用户已下载
    + 导出：点击后打开上传文件或目录的浏览器窗口，选择后执行/api/agent/import，导入当前Agent配置
        + 需要检查同名Agent不能被覆盖
+ Integration需求：../integration/iteration/20260614-1/REQUIREMENT.md

### 迭代 20260614-2
> 新增自 ./iteration/20260614-2/REQUIREMENT.md
+ 在首次设置中点击导出或导入按钮（无任何设置中点击导出或导入的记录）时开启新手引导任务
    + 使用一个动画先高亮导出按钮，然后高亮导入按钮（中间间隔2秒，如果当前没有导出按钮需要展示），并暗淡其他部分并不可点击
        + 新手引导文案：导出或导入Agent配置，方便备份与协作。
        + 提供一个`完成`的新手引导按钮，点击后关闭新手引导任务的遮罩（如果原本没有导出按钮也需要隐藏），所有功能可点击
+ 首次使用（无任何浏览器记录）时新手引导相互独立，并且需要兼容
    + 首次使用需求：./iteration/20260525-6/REQUIREMENT.md
+ 首次点击居中对话输入框时（无任何点击输入框记录）时增加引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击居中对话输入框需求：./iteration/20260525-7/REQUIREMENT.md
+ 首次点击设置中Agent的蜂群小图标，（无任何点击蜂群开关的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击设置中Agent的蜂群小图标：./iteration/20260525-8/REQUIREMENT.md
+ 首次点击左侧边栏插件按钮（无任何点击插件按钮的记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击左侧边栏插件按钮需求：./iteration/20260526-1/REQUIREMENT.md
+ 首次点击虚拟文件系统SOUL.md或USER.md的任意位置，如查看或编辑文件、小灯泡图标等（无任何点击SOUL.md或USER.md任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统SOUL.md或USER.md的任意位置需求：./iteration/20260526-2/REQUIREMENT.md
+ 首次点击右侧边栏的第一栏任何元素时（备忘录日历、明细或元数据）（无任何点击备忘录日历、明细或元数据任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-3/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统skills目录或创建SKILL按钮需求：./iteration/20260526-5/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-7/REQUIREMENT.md
+ 首次点击右侧边栏知识库任意元素时（无任何点击右侧边栏知识库任意元素的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-8/REQUIREMENT.md
+ 首次点击浏览器插件重启按钮时（无任何点击浏览器插件重启按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-9/REQUIREMENT.md
+ 首次点击浏览器插件重启按钮时（无任何点击浏览器插件重启按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-10/REQUIREMENT.md
+ 首次点击设置，任意模型的客户化配置（无任何点击客户化配置的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击设置，任意模型的客户化配置需求：./iteration/20260607-1/REQUIREMENT.md
+ 首次出现任意模型客户化配置按钮（五角星）闪烁（无任何客户化配置按钮（五角星）闪烁的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次出现任意模型客户化配置按钮（五角星）闪烁需求：./iteration/20260608-4/REQUIREMENT.md
+ 首次配置完模型密钥并保存后（无任何配置完模型密钥并保存后的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次配置完模型密钥并保存需求：./iteration/20260608-5/REQUIREMENT.md
+ 首次鼠标点击、悬停在备忘录元数据输入框且已完成备忘录新手引导且已配置任意模型（无任何鼠标点击、悬停在备忘录元数据输入框的记录且已完成备忘录新手引导且已配置任意模型）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次鼠标点击、悬停在备忘录元数据输入框且已完成备忘录新手引导且已配置任意模型需求：./iteration/20260609-2/REQUIREMENT.md
+ 首次备忘录任务执行完成（状态为已完成）且当前备忘录明细列表仅此一条（包括其他状态的）（无任何备忘录任务执行完成（状态为已完成）的记录）且当前备忘录明细列表仅此一条（包括其他状态的）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次备忘录任务执行完成（状态为已完成）且当前备忘录明细列表仅此一条（包括其他状态的）需求：./iteration/20260609-3/REQUIREMENT.md
+ 首次点击沙盒按钮（无任何点击沙盒按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击沙盒按钮需求：./iteration/20260609-4/REQUIREMENT.md
+ 首次点击全屏按钮（无任何点击全屏按钮的记录）时开启新手引导任务相互独立，并且需要兼容
    + 首次点击全屏按钮需求：./iteration/20260609-5/REQUIREMENT.md
+ 首次点击虚拟文件系统新建按钮或鼠标首次悬浮在虚拟文件系统除SOUL.md和USER.md外任意可编辑文件（无任何点击虚拟文件系统新建按钮或鼠标首次悬浮在虚拟文件系统除SOUL.md和USER.md外任意可编辑文件的记录）时开启新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统新建按钮或鼠标首次悬浮在虚拟文件系统除SOUL.md和USER.md外任意可编辑文件需求：./iteration/20260610-3/REQUIREMENT.md
+ 首次出现跨会话完成提醒（会话有一个小×，或因为左侧边栏收起时左上收起菜单闪动）（无任何跨会话完成提醒（会话有一个小×，或因为左侧边栏收起时左上收起菜单闪动）的记录）时开启新手引导任务相互独立，并且需要兼容
    + 首次出现跨会话完成提醒（会话有一个小×，或因为左侧边栏收起时左上收起菜单闪动）需求：./iteration/20260610-4/REQUIREMENT.md
+ 首次成功启动任意插件后（无任何成功启动任意插件的记录）时开启新手引导任务相互独立，并且需要兼容
    + 首次成功启动任意插件需求：./iteration/20260611-1/REQUIREMENT.md
+ 在居中对话框存在正确且结束的SSE响应后，首次在居中对话输入框中切换到不同模型（无任何居中对话框存在正确且结束的SSE响应后，在居中对话输入框中切换到不同模型记录）时开启新手引导任务相互独立，并且需要兼容
    + 在居中对话框存在正确且结束的SSE响应后，首次在居中对话输入框中切换到不同模型需求：./iteration/20260611-7/REQUIREMENT.md
+ 在设置中首次成功开启并保存蜂群配置（无任何设置中成功开启并保存蜂群配置时记录）时开启新手引导任务相互独立，并且需要兼容
    + 在设置中首次成功开启并保存蜂群配置的需求：./iteration/20260611-8/REQUIREMENT.md

### 迭代 20260614-3
> 新增自 ./iteration/20260614-3/REQUIREMENT.md
+ 在居中输入框或备忘录输入框首次通过@唤起__internal_token技能并且当前选择模型已经配置了密钥（无任何居中输入框或备忘录输入框首次通过@唤起__internal_token技能并且当前选择模型已经配置了密钥记录）时开启新手引导任务
    + 在居中对话输入框增加新手引导，询问用户是否需要了解技能使用方法（是绑定回车，否绑定ESC），并暗淡其他部分并不可点击
        + 新手引导文案：需要了解__internal_token如何使用吗？
    + 点击是使用动画为用户自动在居中对话框输入：给我几个用自然语言描述[SKILL:__internal_token]的案例
        + [SKILL:__internal_token]在居中对话框输入时的动画需要使用气泡样式
    + 动画最后为用户自动发送，关闭新手引导任务的遮罩，所有功能可点击
+ 首次使用（无任何浏览器记录）时新手引导相互独立，并且需要兼容
    + 首次使用需求：./iteration/20260525-6/REQUIREMENT.md
+ 首次点击居中对话输入框时（无任何点击输入框记录）时增加引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击居中对话输入框需求：./iteration/20260525-7/REQUIREMENT.md
+ 首次点击设置中Agent的蜂群小图标，（无任何点击蜂群开关的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击设置中Agent的蜂群小图标：./iteration/20260525-8/REQUIREMENT.md
+ 首次点击左侧边栏插件按钮（无任何点击插件按钮的记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击左侧边栏插件按钮需求：./iteration/20260526-1/REQUIREMENT.md
+ 首次点击虚拟文件系统SOUL.md或USER.md的任意位置，如查看或编辑文件、小灯泡图标等（无任何点击SOUL.md或USER.md任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统SOUL.md或USER.md的任意位置需求：./iteration/20260526-2/REQUIREMENT.md
+ 首次点击右侧边栏的第一栏任何元素时（备忘录日历、明细或元数据）（无任何点击备忘录日历、明细或元数据任意位置记录）并且当前有可用插件时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-3/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统skills目录或创建SKILL按钮需求：./iteration/20260526-5/REQUIREMENT.md
+ 首次点击虚拟文件系统skills目录或创建SKILL按钮时（无任何点击skills目录或创建SKILL按钮记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏的第一栏任何元素需求：./iteration/20260526-7/REQUIREMENT.md
+ 首次点击右侧边栏知识库任意元素时（无任何点击右侧边栏知识库任意元素的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-8/REQUIREMENT.md
+ 首次点击浏览器插件重启按钮时（无任何点击浏览器插件重启按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-9/REQUIREMENT.md
+ 首次点击浏览器插件重启按钮时（无任何点击浏览器插件重启按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击右侧边栏知识库任意元素需求：./iteration/20260526-10/REQUIREMENT.md
+ 首次点击设置，任意模型的客户化配置（无任何点击客户化配置的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击设置，任意模型的客户化配置需求：./iteration/20260607-1/REQUIREMENT.md
+ 首次出现任意模型客户化配置按钮（五角星）闪烁（无任何客户化配置按钮（五角星）闪烁的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次出现任意模型客户化配置按钮（五角星）闪烁需求：./iteration/20260608-4/REQUIREMENT.md
+ 首次配置完模型密钥并保存后（无任何配置完模型密钥并保存后的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次配置完模型密钥并保存需求：./iteration/20260608-5/REQUIREMENT.md
+ 首次鼠标点击、悬停在备忘录元数据输入框且已完成备忘录新手引导且已配置任意模型（无任何鼠标点击、悬停在备忘录元数据输入框的记录且已完成备忘录新手引导且已配置任意模型）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次鼠标点击、悬停在备忘录元数据输入框且已完成备忘录新手引导且已配置任意模型需求：./iteration/20260609-2/REQUIREMENT.md
+ 首次备忘录任务执行完成（状态为已完成）且当前备忘录明细列表仅此一条（包括其他状态的）（无任何备忘录任务执行完成（状态为已完成）的记录）且当前备忘录明细列表仅此一条（包括其他状态的）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次备忘录任务执行完成（状态为已完成）且当前备忘录明细列表仅此一条（包括其他状态的）需求：./iteration/20260609-3/REQUIREMENT.md
+ 首次点击沙盒按钮（无任何点击沙盒按钮的记录）时开启引导对话输入的新手引导任务相互独立，并且需要兼容
    + 首次点击沙盒按钮需求：./iteration/20260609-4/REQUIREMENT.md
+ 首次点击全屏按钮（无任何点击全屏按钮的记录）时开启新手引导任务相互独立，并且需要兼容
    + 首次点击全屏按钮需求：./iteration/20260609-5/REQUIREMENT.md
+ 首次点击虚拟文件系统新建按钮或鼠标首次悬浮在虚拟文件系统除SOUL.md和USER.md外任意可编辑文件（无任何点击虚拟文件系统新建按钮或鼠标首次悬浮在虚拟文件系统除SOUL.md和USER.md外任意可编辑文件的记录）时开启新手引导任务相互独立，并且需要兼容
    + 首次点击虚拟文件系统新建按钮或鼠标首次悬浮在虚拟文件系统除SOUL.md和USER.md外任意可编辑文件需求：./iteration/20260610-3/REQUIREMENT.md
+ 首次出现跨会话完成提醒（会话有一个小×，或因为左侧边栏收起时左上收起菜单闪动）（无任何跨会话完成提醒（会话有一个小×，或因为左侧边栏收起时左上收起菜单闪动）的记录）时开启新手引导任务相互独立，并且需要兼容
    + 首次出现跨会话完成提醒（会话有一个小×，或因为左侧边栏收起时左上收起菜单闪动）需求：./iteration/20260610-4/REQUIREMENT.md
+ 首次成功启动任意插件后（无任何成功启动任意插件的记录）时开启新手引导任务相互独立，并且需要兼容
    + 首次成功启动任意插件需求：./iteration/20260611-1/REQUIREMENT.md
+ 在居中对话框存在正确且结束的SSE响应后，首次在居中对话输入框中切换到不同模型（无任何居中对话框存在正确且结束的SSE响应后，在居中对话输入框中切换到不同模型记录）时开启新手引导任务相互独立，并且需要兼容
    + 在居中对话框存在正确且结束的SSE响应后，首次在居中对话输入框中切换到不同模型需求：./iteration/20260611-7/REQUIREMENT.md
+ 设置中首次成功开启并保存蜂群配置（无任何设置中成功开启并保存蜂群配置时记录）时开启新手引导任务相互独立，并且需要兼容
    + 设置中首次成功开启并保存蜂群配置需求：./iteration/20260611-8/REQUIREMENT.md

### 迭代 20260614-4
> 新增自 ./iteration/20260614-4/REQUIREMENT.md
+ 每次切换会话（包括首次打开应用进入会话）后，如果当前Agent+ChatID（会话）没有正在等待SSE响应（没有未完成任务，处于结束态）则自动调用一次[重载当前会话]以同步最新数据
+ 每30秒检查一次当前Agent+ChatID（会话）如果没有正在等待SSE响应（没有未完成任务，处于结束态）则自动调用一次[重载当前会话]以同步最新数据
+ 总结：切换会话时主动调用刷新切换的会话，每30秒主动调用刷新当前的会话
+ 调用刷新后需要在Tips提示（原逻辑）

### 迭代 20260618-1
> 新增自 ./iteration/20260618-1/REQUIREMENT.md
+ 区分系统（MAC或是Windows/WSL）调用不同沙盒方案（包括目录选择）
+ 严格隔离MAC系统的实现路径，完全保持原样
+ WSL沙盒需求：../cli-get/sandbox/wsl/REQUIREMENT.md
+ Proxy需求：../proxy/iteration/20260618-1/REQUIREMENT.md
+ Integration需求：../integration/iteration/20260618-1/REQUIREMENT.md

### 迭代 20260618-2
> 新增自 ./iteration/20260618-2/REQUIREMENT.md
+ SSE响应中choices[0].metadata.__PROCESS__不为空时不渲染正文，展示到当前会话最后一个assistant气泡底部footer预留槽位
+ 展示内容取choices[0].delta.content，按前端收口规则做文本归一化（去掉\r、trim、连续换行折叠为空格、连续空白折叠为单个空格）
+ __PROCESS__提示仅按当前ChatId匹配，不再按AgentId::ChatId隔离
+ __PROCESS__临时提示展示5秒，下一轮收到新的__PROCESS__则立即替换并刷新5秒计时
+ 纯__PROCESS__报文在实时SSE和历史恢复时都不进入居中对话框正文
+ "努力工作中"位置逻辑保持原样，仅跟当前会话busy状态显示/隐藏，不被__PROCESS__替换
+ __WARN__红框重试提示只作用于当前这段SSE响应，不影响下一段
+ biz=cli且workflow=sub或__sub时__WARN__不进入红框提示
+ finish_reason=error保持现有错误展示逻辑，不被__PROCESS__和__WARN__覆盖

### 迭代 20260619-1
> 新增自 ./iteration/20260619-1/REQUIREMENT.md
+ __WARN__不再插入独立红框，改为展示到footer预留槽位（与__PROCESS__共用展示位）
+ 展示内容取choices[0].delta.content，按前端收口规则做文本归一化
+ __WARN__提示仅按当前ChatId匹配，不再按AgentId::ChatId隔离
+ __WARN__展示时间优先取choices[0].metadata.__DELAY__（毫秒），否则默认30秒
+ __WARN__与__PROCESS__共用同一footer展示位，后到覆盖，各自按时长重新计时
+ __WARN__展示样式改为荧光红色警告风格；__PROCESS__原有样式保持不变
+ 原本__WARN__的红框展示逻辑整体删除，不保留兼容代码
+ __WARN__不再使用前端兜底文案，仅使用服务端实际下发的delta.content
+ 红框错误展示仅保留错误场景：HTTP非200、code不在200-299范围内、finish_reason=error
+ 首次收到__WARN__时引导的高亮目标改为footer中__WARN__提示

### 迭代 20260620-2
> 新增自 ./iteration/20260620-2/REQUIREMENT.md
+ SSE响应里choices[0].metadata.__TARGET__为非空标记时，__PROCESS__和__WARN__改为展示在当前等待响应assistant气泡内侧左下
+ __TARGET__只影响展示位置，不参与提示作用域匹配
+ 无论是否携带__TARGET__，提示匹配和清理都只按当前ChatId处理

### 迭代 20260620-3
> 新增自 ./iteration/20260620-3/REQUIREMENT.md
+ SSE响应里choices[0].metadata.__TASK_START__为非空时，在当前等待响应assistant气泡右上角展示任务气泡；气泡正文取choices[0].delta.content按前端收口规则归一化后的首字符并转大写
+ 只有当前报文归一化后仍能取到首字符时，才新增或更新对应任务气泡
+ 同一__TASK_START__ key重复到达时更新现有气泡内容；新的key追加到最右侧，并推动已有气泡整体左移
+ choices[0].metadata.__TASK_CLOSE__为非空时按key移除任务气泡，剩余气泡自动右移补位
+ 任务气泡匹配和清理仅按当前ChatId处理，不参与AgentId和__TARGET__
+ 纯__TASK_START__/__TASK_CLOSE__控制包不进入居中对话框正文，历史恢复和restore不重复回放
+ 若当前整轮SSE直到结束都未收到非空__TARGET__，收尾时清空残留任务气泡
+ 任务气泡产生与消失都带过渡动画

### 编写代码
    + 所有固定定位遮罩层、确认框、弹窗必须统一管理层级、显示状态和pointer-events，按左侧Sidebar、居中会话区、右侧Sidebar分域隔离，禁止未激活浮层跨区域遮挡或拦截其他可操作控件点击
    + 涉及弹窗、日志面板、确认框、遮罩层、透明点击层、复制按钮、全局选区逻辑，默认都必须遵守"交互域隔离 + 隐藏视图彻底失活"原则
    + 设计上先统一弹层挂哪、用哪套坐标系，并避免把会溢出的弹层放进overflow:hidden 容器里
    + 代码上把定位收口成一个公共函数或portal 机制，不要在业务里各自手算 left/top
    + 点击CLI子面板执行CLI命令时，如果执行出现异常（如超时），需要先关闭CLI子命令执行浮层后再展示错误浮层，避免两个浮层重叠
    + 所有固定定位遮罩层、确认框、弹窗必须统一管理层级、显示状态和pointer-events，按左侧Sidebar、居中会话区、右侧Sidebar分域隔离，禁止未激活浮层跨区域遮挡或拦截其他可操作控件点击
    + 涉及弹窗、日志面板、确认框、遮罩层、透明点击层、复制按钮、全局选区逻辑，默认都必须遵守"交互域隔离 + 隐藏视图彻底失活"原则
        + 设计上先统一弹层挂哪、用哪套坐标系，并避免把会溢出的弹层放进overflow:hidden 容器里
        + 代码上把定位收口成一个公共函数或portal 机制，不要在业务里各自手算 left/top
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 相关迭代：iteration/日期/REQUIREMENT.md
> 新增自 iteration/20260509-3/REQUIREMENT.md
> 合并截止：./iteration/20260620-3/REQUIREMENT.md，下次合并从此之后的新迭代开始
