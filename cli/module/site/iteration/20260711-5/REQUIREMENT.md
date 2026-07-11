### 第一性原则
+ 仅可以新增/更新/删除site（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Site介绍：../../REQUIREMENT.md
+ Site手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../REQUIREMENT.md
+ 所以设计/编译都需要遵守site页面的现有前端收口原则

### 需求介绍
+ 本次需要继续增强 site 页面的 onboarding 体系，让用户可以显式控制“是否还需要新手引导”，并在引导过程中随时整条跳过：
    + 设置面板底部左侧需要新增一个 `新手引导` 总开关
    + 默认状态为开启，即显示 `✓`
    + 点击后切换为 `×`，表示不再触发任何新手引导
    + 再次点击则切回 `✓`
+ 这个总开关的正式交互语义收口为“修改后需要保存才生效”：
    + 在设置面板里点击 `✓ / ×` 时，只修改当前面板内的草稿状态
    + 只有点击设置面板右下角 `保存` 后，新的 onboarding 开关状态才真正生效
    + 点击 `取消`、点遮罩关闭设置或其它未保存退出路径时，这次切换必须作废
    + 检查点：
        + 用户在设置里把 `✓` 改成 `×` 后，如果没有点击 `保存`，页面上的 onboarding 行为不得改变
        + 用户改完后点击 `保存`，新的开关状态必须立即对后续 onboarding 入口生效
+ 当总开关被保存为关闭态 `×` 后，页面需要统一禁止任何 onboarding 再被触发：
    + 已排队等待播放的 onboarding 不得继续补播
    + 各类同步启动入口、异步启动入口和 queue/pending 入口都必须统一尊重该状态
    + 如果当前页面上已经有 onboarding 正在播放，保存关闭后应立即停止该引导
+ 除了设置中的总开关，还需要给各条 onboarding 增加“整条跳过”能力：
    + 在每个 onboarding 的图中标记位置新增一个 `×`
    + 点击该 `×` 时，要直接跳过“当前整个 onboarding”，不是只关闭当前一步
    + 这套 `×` 要覆盖所有复用通用 `.onboarding-card` 模板的引导卡片
+ onboarding 卡片上的 `×` 不是所有步骤都显示，最后一步必须隐藏：
    + 当当前步骤的主按钮文案是 `完成` 时，不显示右上角 `×`
    + 只有中间讲解步骤、可继续推进的步骤才显示 `×`
    + 检查点：
        + 非最后一步：显示 `×`
        + 最后一步按钮为 `完成`：不显示 `×`
+ 点击 onboarding 卡片右上角 `×` 后，不同类型引导要保留现有业务语义：
    + 首次使用主引导链路如果被手动 `×` 跳过，需要整体标记为已完成，避免下次刷新又从头自动开始
    + 某些已有特殊语义的独立引导，例如 standalone mode、保存模型校验引导，也要继续保留各自已有的关闭后状态记录
    + 检查点：
        + 跳过后不会只关闭当前一步却继续弹后续步骤
        + 特殊引导的原有点击记录、pending 清理、补播抑制语义不能被改坏

### 最新实现收口
+ 当前最新实现已经把本次需求正式收口为“设置内总开关 + 引导卡片整条跳过”两套能力：
    + 设置面板底部左侧新增 `新手引导` 开关，视觉上显示为 `✓ / ×`
    + 默认状态为 `✓`
    + 切换为 `×` 后，表示关闭后续 onboarding 触发
    + 再切回 `✓` 时，恢复后续 onboarding 触发能力
+ 当前总开关正式采用“草稿修改，保存生效”的语义：
    + 打开设置时，先把当前真实 onboarding 状态复制为设置草稿
    + 点击开关只修改草稿，不立刻写 `localStorage`
    + 点击 `保存` 后，草稿才真正写回运行态和持久化状态
    + 点击 `取消` 或未保存关闭设置时，草稿作废
+ 当前保存关闭 onboarding 后的正式收口行为如下：
    + 立即写入 `deepright_onboarding_enabled`
    + 页面内所有 onboarding 启动入口立即被总开关统一拦截
    + 已排队的 onboarding pending 状态会立即清空
    + 当前正在播放的 onboarding 会立即结束
+ 当前所有 onboarding 卡片模板都统一支持右上角 `×`，但最终完成步骤不显示：
    + 非最后一步显示 `×`
    + 最后一步主按钮文案为 `完成` 时隐藏 `×`
    + 点击 `×` 后，直接结束当前整条 onboarding，而不是只推进或关闭当前一步
+ 当前整条跳过的特殊语义也已完成收口：
    + 首次使用主引导链路点击 `×` 时，会整体标记为已完成，避免后续自动补播
    + `standalone mode` 引导点击 `×` 时，会沿用原有“已说明但未切换”的记录语义
    + `model save verify` 引导点击 `×` 时，会同步写入点击记录并清理 pending

### 技术实现
+ 当前实现全部收口在 `../../index.html` 单文件内，没有新增额外依赖：
    + 新增持久化 key：`ONBOARDING_ENABLED_KEY = 'deepright_onboarding_enabled'`
    + 新增读取函数：`getStoredOnboardingEnabled()`
    + `onboardingState` 新增 `enabled` 字段，并同步到根节点属性 `data-onboarding-enabled`
+ 设置面板里的总开关采用“真实值 + 草稿值”双状态实现：
    + 真实值使用 `onboardingState.enabled`
    + 草稿值使用 `settingsOnboardingDraftEnabled`
    + `openSettings()` 中调用 `initSettingsOnboardingDraft()`，把当前真实值复制到草稿
    + `closeSettings()` 中调用 `resetSettingsOnboardingDraft()`，确保未保存退出时丢弃草稿
    + `renderSettingsOnboardingToggle()` 统一根据草稿值渲染 `✓ / ×`
    + 设置里的点击事件收口到 `toggleSettingsOnboardingEnabled()`，只改草稿，不直接生效
    + `saveSettingsAsync()` 在已有设置保存逻辑成功后，再比对草稿值和真实值；只有发生变化时才调用 `setOnboardingEnabled(...)`
+ 真正的开关提交逻辑统一收口在 `setOnboardingEnabled(enabled)`：
    + 写入 `localStorage`
    + 更新 `onboardingState.enabled`
    + 更新 `data-onboarding-enabled`
    + 当保存为关闭态时，调用 `clearDeferredOnboardingStarts({ clearPersistent: true })` 清理待播状态
    + 如果当前已有 onboarding 正在播放，则立即 `stopOnboarding(false)`
+ onboarding 卡片右上角 `×` 采用统一模板化实现：
    + 所有复用 `.onboarding-card` 的模板都新增 `.onboarding-card-head` 与 `.onboarding-card-close`
    + `syncOnboardingCardContent(scope, config)` 里统一控制右上角 `×` 显隐
    + `shouldShowOnboardingDismiss(config)` 当前规则为：
        + `config.hideDismiss === true` 时隐藏
        + 主按钮文案为 `完成` 时隐藏
        + 其它步骤默认显示
+ 点击 `×` 的整条跳过行为统一收口在 `handleOnboardingDismiss(event)`：
    + 默认直接 `stopOnboarding(false)`
    + 首次使用主引导链路通过 `isFirstRunOnboardingStep(step)` 识别，并转为 `stopOnboarding(true)`
    + `standalone mode` 引导会补写 `completedWithoutSwitch = true`
    + `model save verify` 引导会补写点击记录并清掉 pending
+ 所有 onboarding 入口统一通过 `isOnboardingEnabled()` 做总开关兜底：
    + `isOnboardingStartLocked()` 会把“总开关关闭”视为启动锁
    + 各类 `start...Onboarding()` / `queue...Onboarding()` 入口都已补齐总开关短路
    + 异步引导入口在 `await` 之后也增加了二次检查，避免等待过程中关闭总开关后仍然弹出

### 编写代码
+ 以现有site页面技术栈编写以上代码，要求：
    + 在../../index.html中按现有HTML/CSS/JavaScript组织方式实现
    + 不引入新的构建流程和额外运行时依赖
    + 代码简洁，尽量复用现有 onboarding 模板、设置面板底部 action 区、状态持久化和 queue/pending 清理逻辑
    + 总开关的“草稿态”和“真实生效态”必须显式分离，禁止在设置内点击时直接改动真实运行态
    + onboarding 右上角 `×` 的显隐规则必须模板统一，禁止为单个引导逐个手写特殊 DOM 分支
    + 各类 onboarding 启动入口必须统一尊重总开关，不能只拦一部分 start 逻辑而遗漏 queue/pending 或异步分支
    + 引导整条跳过后，不得留下会反复补播的 pending 状态
    + 能用现有开源包和浏览器能力的就不要重复造轮子
+ 最小范围更新

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
