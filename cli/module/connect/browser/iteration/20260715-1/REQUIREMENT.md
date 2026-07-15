### 第一性原则
+ 仅可以新增/更新/删除browser（ ../../..）目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../../DESIGN.md
+ 本模块设计文档：../../../DESIGN.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 需求介绍
+ 处理Browser插件中 `--connect-bin` 的使用边界，目标是让 `browser` 在运行时插件目录下稳定找到 integration，并避免除 `start` 外其他命令继续透传或偷偷兼容 `--connect-bin`
+ 新增运行时文件 `browser_runtime.json`，放在 Browser 插件运行目录下（与 `browser_instance.json`、`browser.log` 同级）
+ 只有 `browser start` 命令允许接收 `--connect-bin`
+ `browser start` 成功后，才将本次 `--connect-bin` 写入或覆盖 `browser_runtime.json`
+ `browser start` 失败时，不写入 `browser_runtime.json`，也不覆盖旧文件
+ 除 `browser start` 外，所有 Browser 命令如果传入 `--connect-bin`，都必须立即判定为不合法参数并报错，不能继续执行，不能静默忽略，不能做兼容处理
+ 后续所有 Browser 内部需要用到 integration 路径或运行时根目录的地方，都统一从 `browser_runtime.json` 读取
+ `browserResolveRuntimeConfigPath` 改为优先从 `browser_runtime.json` 读取，不再直接消费命令行传入的 `--connect-bin`
+ `browserRuntimeRoot` 改为优先从 `browser_runtime.json` 读取，不再直接消费命令行传入的 `--connect-bin`
+ 如果 `browser_runtime.json` 不存在，则按当前 `browser` 二进制路径推断运行时目录作为回退逻辑
+ `browser stop` 成功后，需要 best-effort 删除 `browser_runtime.json`
+ 删除 `browser_runtime.json` 失败时，只记录日志，不导致 `browser stop` 失败
+ `browser instance shutdown` 不删除 `browser_runtime.json`
+ 如果其他内部逻辑仍然需要 integration 路径，也统一改为从 `browser_runtime.json` 读取，禁止保留新的命令行 `--connect-bin` 读取入口

### 最终效果
+ 当 Browser 已从运行时插件目录执行时，`browser start --connect-bin ...` 会把 integration 路径写入 `browser_runtime.json`
+ 后续 `browser instance create/init/restart/get/list/shutdown` 以及其他 Browser 命令，都不再接收 `--connect-bin`，而是从 `browser_runtime.json` 读取需要的 integration 路径
+ 当用户或调用链误传 `--connect-bin` 到非 `start` 命令时，会第一时间报错暴露异常
+ `browser stop` 会清理 `browser_runtime.json`，但清理失败不会影响 stop 主流程

#### 插件日志
+ 必须在插件同目录下提供以browser.log的日志文件

### 撰写手册
+ 编写USER_GUIDE.md

### 同步代码
+ ../../../browser/REQUIREMENT.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 编写代码
+ 以Golang编写以上代码，要求：
    + 需要将browser后台daemon启动为真正独立于前台命令生命周期的进程，不能继承父进程的临时stdout/stderr管道，避免后续命令执行时因输出链路失效导致daemon断开
    + browser在插件配置和命令链路中正确识别并校验cookie_path，确保fetch、store和start都按该 Cookie 文件工作，出错时终止启动并写明browser.log
    + 代码简洁，包体积越小越好
    + 能用开源包的就用开源包
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 验收测试
+ 不能只看端口通不通，必须校验daemon归属，避免误连旧版本
+ 后台daemon必须脱离父会话，不能出现start后几秒自灭
+ 所有后台进程功能都要补，父命令退出后仍存活的验收测试
+ 严格遵守指纹需求：../../CHECK.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 同步代码：../../../../integration/REQUIREMENT.md（每次都要同步更新代码）
