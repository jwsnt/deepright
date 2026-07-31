### 第一性原则
+ 仅可以新增/更新/删除browser（ ../../..）目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../../DESIGN.md
+ 本模块设计文档：../../../DESIGN.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 需求介绍
+ Browser 插件成功执行 `browser start` 后，必须从本次主应用（Integration）对应的 `config/config.json` 读取嵌套配置 `browser.clear` 与 `browser.scan`。
+ `browser.clear` 和 `browser.scan` 均为正整数，单位为小时：
```json
{
  "browser": {
    "init_timeout": 300,
    "clear": 72,
    "scan": 2
  }
}
```
+ `browser.clear` 表示 Chrome profile 目录的过期时长；目录最后修改时间距离当前时间严格超过该时长时，视为可清理目录。
+ `browser.scan` 表示后台扫描周期。插件启动成功后必须立即异步执行首次扫描，之后每隔 `browser.scan` 小时异步执行一次；扫描与删除不得阻塞 `browser start`、Playwright daemon 启动或其他 Browser CLI 命令。
+ Browser 的 `start` 前台命令会退出，因此周期任务必须由独立后台进程承载；重复执行 `browser start` 时必须复用或替换已有清理进程，不能并行创建多个扫描循环。`browser stop` 必须停止该后台清理进程。
+ macOS：扫描主应用配置的 `agent-dir` 下每个 Agent 工作目录的直接子目录；只处理名称满足 `chrome_` 加非空后缀、且不区分大小写的目录。
+ Windows WSL / WSL2：只扫描 Windows 宿主机 `C:\\ProgramData\\deepright` 下满足同一名称规则的目录。Browser 在 WSL 中运行时，允许转换为对应的 Linux 挂载路径访问，但日志中必须保留 Windows 目标目录语义。
+ 原生 Linux 与 Windows 原生环境不执行本次 profile 清理任务。
+ 清理时只能删除已经判定过期的目录；不得删除文件、符号链接或不匹配 `chrome_` 前缀规则的目录。
+ 主应用配置不存在、`browser` 配置不存在、`browser.clear` 或 `browser.scan` 缺失、不是正整数或超出可表示时长时，Browser 必须跳过清理任务且不得影响 `browser start` 成功；原因必须记录在插件同目录 `browser.log`。

#### 插件日志
+ 必须在插件同目录下提供以browser.log的日志文件
+ 必须记录后台清理进程的启动、停止、配置跳过、每次扫描、删除成功与删除失败事件；日志需包含目标根目录、扫描/删除数量及错误信息（如有）。

### 撰写手册
+ 编写USER_GUIDE.md

### 同步代码
+ ../../../browser/REQUIREMENT.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 编写代码
+ 在 Browser 二进制中实现独立的 profile 清理后台命令与生命周期管理，不新增用户可见二进制或用户可见 CLI 命令。
+ 清理配置必须从 `browser start` 成功后写入的 Browser runtime 记录定位主应用 `config/config.json`；不得从当前工作目录猜测配置文件。
+ 使用目录自身的最后修改时间作为过期依据；比较时使用当前时间与 `browser.clear` 的小时数。
+ 目录枚举、过期判断、删除、定时等待和错误处理均应位于后台清理进程中。
+ `browser start` 仅负责在成功后异步拉起后台清理进程；启动清理进程或读取清理配置失败只能写日志，不能回滚已经成功启动的 Browser 插件。

### 验收测试
+ 验证 `browser.clear=72`、`browser.scan=2` 能正确解析为小时级时长；缺失、非正整数、错误类型与超大值均跳过任务并记录原因。
+ 验证首次扫描在清理后台进程启动后立即执行，后续周期使用 `browser.scan`。
+ 验证 macOS 仅扫描 `agent-dir/<agent>/chrome_*` 目录，大小写不敏感，并只删除最后修改时间超过 `browser.clear` 的目录。
+ 验证 WSL 仅扫描 `C:\\ProgramData\\deepright` 对应目录，且不会扫描原生 Linux 路径。
+ 验证匹配目录中的未过期目录、普通文件、符号链接和非 `chrome_` 前缀目录均保留。
+ 验证清理扫描与目录删除不阻塞 `browser start`，重复 `start` 不产生多个清理循环，`browser stop` 能终止清理后台进程。

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 同步代码：../../../../integration/REQUIREMENT.md（每次都要同步更新代码）
