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
+ 当前会话沙盒UI的状态查询、状态恢复、切换会话后的自动刷新都需要改为仅按`chatId`处理
+ 前端读取会话沙盒状态时不再依赖`agentId`
    + 即使当前未选中`agentId`，只要存在`chatId`也需要能够查询并恢复当前会话沙盒状态
    + 前端不允许因为当前没有选中`agentId`就直接跳过当前会话沙盒状态恢复
+ 写入会话沙盒状态时继续调用`/api/sandbox=沙盒模式?agentId=x&chatId=y`
    + 其中`chatId`用于定位当前会话沙盒状态
    + `agentId`仅作为服务端日志输入
+ 前端读取会话沙盒状态时改为使用仅按`chatId`查询的`/api/sandbox_status`
    + 不需要为了状态查询额外兜底拼接`agentId + chatId`
+ 前端沙盒缓存、恢复和切换逻辑需要从“Agent+Chat绑定”改为“Chat绑定”
    + 沙盒状态缓存key需要改为`chatId`
    + 沙盒请求中的并发保护、恢复快照、切换会话刷新都需要改为按`chatId`作用域处理
+ 浮层中的沙盒状态、更新时间和恢复逻辑都需要以当前`chatId`为唯一作用域
    + 浮层仍可展示当前选中的`agentId`
    + 但该展示信息不允许参与沙盒状态查询、恢复、缓存命中和刷新判断
+ 前端会话沙盒交互协议需要与最新跨系统沙盒方案保持一致
    + 仍保留 `目录权限` / `离线执行` / `双重限制` / `关闭沙盒` 四个入口
    + `目录权限`=`filepick`，`离线执行`=`net`，`双重限制`=`filepick_net`，`关闭沙盒`=`off`
    + 写接口继续使用`/api/sandbox=沙盒模式?agentId=x&chatId=y`
    + 前端不需要区分 macOS 与 Windows/WSL；后端会自动选择对应沙盒 helper
+ 涉及目录白名单的前端交互需要继续兼容系统差异
    + `filepick`与`filepick_net`继续复用现有目录授权/目录选择交互
    + Windows/WSL 下由后端拉起对应的 WSL `bubblewrap` 目录授权与执行流程
    + 如果后端返回 WSL helper / `bubblewrap` 不可用或目录授权失败，前端需要展示失败提示，但不能破坏当前页面其余会话功能
+ 需要同步更新与proxy / integration最新沙盒接口语义一致的前端逻辑
    + Proxy需求：../../../proxy/iteration/20260709-1/REQUIREMENT.md
    + Integration需求：../../../integration/iteration/20260707-1/REQUIREMENT.md
+ 需要补充前端验证
    + 切换不同`agentId`但保持同一`chatId`时，沙盒状态需要保持一致
    + 当前未选择`agentId`时，只要存在`chatId`仍可恢复沙盒状态
    + 写入后重新进入同一`chatId`时能够恢复正确的沙盒状态和更新时间
    + 目录权限/双重限制在不同系统下继续复用同一前端协议与浮层交互的验证
    + WSL 目录授权失败时能够展示错误提示且不影响当前会话其它交互的验证

### 编写代码
+ 以现有site页面技术栈编写以上代码，要求：
    + 在../../index.html中按现有HTML/CSS/JavaScript组织方式实现
    + 不引入新的构建流程和额外运行时依赖
    + 代码简洁，尽量复用现有SSE解析、assistant气泡渲染、思考片段展示、历史恢复和轮询状态同步逻辑
    + 冷历史模式与现有实时/刷新恢复模式必须显式分流，避免在现有 `appendRestoredRecords`、pending 请求恢复、CLI 子任务恢复逻辑里硬塞条件导致语义串线
    + 若为冷历史/向上恢复补充新的查询参数或新的取数模式，必须保持现有 `/api/restore` 实时/刷新恢复语义完全兼容
    + 向上恢复、新增提示文案、历史窗口裁切、滚动加载都必须保持现有滚动位置稳定，不允许因为插入更早历史导致当前阅读位置大幅跳动
    + 能用现有开源包和浏览器能力的就不要重复造轮子
+ 最小范围更新

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
