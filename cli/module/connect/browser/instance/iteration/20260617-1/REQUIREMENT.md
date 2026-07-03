### 第一性原则
+ 本迭代需求文档以同目录实现文件 `browser_instance_wsl.go` 的实际行为为准，用于细化、补充并修正文档，不再保留与实现不一致的设想性描述
+ 仅允许新增、更新、删除 `browser` 目录及其子目录内的文件和目录

### 技术规范
+ 严格遵守整体设计文档：`../../../../../DESIGN.md`
+ 本模块设计文档：`../../../../DESIGN.md`
+ 设计、编译、调用方式均需遵守 browser 模块的二进制与 CLI 收口原则
+ 严格遵守指纹约束：`../../../CHECK.md`

### 需求目标
+ 在当前目录提供一个独立的 WSL Browser Instance 管理程序：`browser_instance_wsl.go`
+ 该程序负责按 `agentId + chatId` 维度复用、恢复或新建一个可用的 Chrome CDP 实例，并以 JSON 输出结果

### 运行前提
+ 程序必须运行在 WSL 环境内
    + 仅 `runtime.GOOS == linux` 不足以判定，还要满足 `WSL_DISTRO_NAME`、`WSL_INTEROP` 或 `/proc/sys/kernel/osrelease`、`/proc/version` 中存在 WSL/Microsoft 特征
    + 若不在 WSL 内运行，返回 `{"status":1,"message":"browser_instance_wsl must run inside WSL"}`
+ 本机必须存在可用的 Chrome 可执行文件：
    + 若传入 `--chrome` 且去除首尾空白后非空，则校验该路径
    + 否则回退到默认路径：`/mnt/c/Program Files/Google/Chrome/Application/chrome.exe`
    + 若目标路径不存在，返回 `{"status":1,"message":"chrome not found: <实际路径>"}` 

### CLI 约定
+ 入参：
    + `--agentId`：必填，按 `trim + lower-case` 归一化后参与查询和写入
    + `--chatId`：必填，按 `trim + lower-case` 归一化后参与查询和写入
    + `--headless`：可选，默认 `true`
    + `--chrome`：可选，用于指定 Chrome 可执行文件路径；未传或仅包含空白字符时回退到默认路径 `/mnt/c/Program Files/Google/Chrome/Application/chrome.exe`
+ 参数缺失或解析失败时，统一输出 JSON 错误：
    + `agentId` 为空时返回 `{"status":1,"message":"agentId is required"}`
    + `chatId` 为空时返回 `{"status":1,"message":"chatId is required"}`
+ 当前实现不提供 `help` 子命令，也不会输出 flag 默认帮助文案；所有失败都走 JSON 错误输出

### 数据存储
+ 使用当前目录下名为 `browser_data` 的 SQLite 数据库；若当前工作目录本身像 browser 程序目录，则优先使用 `cwd/browser_data`，否则回退到 `browser_instance_wsl.go` 所在目录下的 `browser_data`
+ 数据表名固定为 `browser_instance_wsl`
+ 表结构包含以下字段：
    + `agent_id`、`chat_id`：联合主键
    + `pid`
    + `port`
    + `ws`
    + `http`
    + `user_data_dir`
    + `updated_at`
+ 启动时若表不存在，程序会自动建表
+ 每次成功拿到可用实例后，都会执行 upsert，`updated_at` 使用 `RFC3339` 时间格式

### 主流程
+ 程序按如下顺序获取实例：

#### 1. 优先复用已登记且仍然存活的实例
+ 用归一化后的 `agentId + chatId` 查询 `browser_instance_wsl` 记录
+ 若记录存在且 `port > 0`，则调用：
    + `curl -s http://localhost:<port>/json/version`
+ 解析返回 JSON 中的 `webSocketDebuggerUrl`
+ 当探活成功时：
    + `ws` 取 `webSocketDebuggerUrl`
    + `http` 固定写成 `http://localhost:<port>`
    + `pid` 优先根据端口重新用 `netstat.exe -ano` 反查；若失败则沿用库中旧值
    + 若记录中的 `user_data_dir` 为空，则会尝试从 `/proc/<pid>/cmdline`，再回退到 `powershell.exe + Win32_Process.CommandLine` 中补齐 `--user-data-dir`
+ 复用成功后，刷新数据库记录并返回成功 JSON

#### 2. 已有记录失活时，优先尝试“原地重启”
+ 当已有记录探活失败时，当前实现不会先删除数据库记录，也不会先删除旧 profile
+ 若旧记录满足以下条件，则尝试复用原端口和原 profile 重启：
    + `port > 0`
    + `user_data_dir` 非空
    + `user_data_dir` 可通过 `wslpath -u` 转成有效的 Unix 路径
    + 对应 profile 目录仍然存在
+ 重启前会清理 profile 内的锁相关文件/目录，包括但不限于：
    + `LOCK`
    + `LOCKFILE`
    + `SINGLETONLOCK`
    + `SINGLETONCOOKIE`
    + `SINGLETONSOCKET`
    + `DEVTOOLSACTIVEPORT`
    + 以 `.lock` 结尾的条目
    + 以 `-journal` 结尾的条目
+ 然后最多等待 5 秒，确认旧 `port` 已被释放
+ 端口释放后，用相同 `port`、相同 `user-data-dir` 重启 Chrome
+ 若重启成功并探活通过，则 upsert 新状态并返回成功 JSON

#### 3. 无可复用实例时，新建实例
+ 当不存在记录，或已有记录无法直接复用且无法成功原地重启时，创建全新 profile 目录并启动新实例
+ profile 根目录固定为：
    + Windows 路径：`C:\tmp`
    + WSL 路径：`/mnt/c/tmp`
+ profile 目录命名规则：
    + `chrome_<4位随机字符>`
    + 随机字符集为 `a-z0-9`
    + 若目录已存在则重新生成，最多尝试 256 次
+ 目录会先在 `/mnt/c/tmp` 真实创建成功，再将对应 Windows 路径传给 Chrome

### Chrome 启动参数
+ Chrome 启动路径规则：
    + 如果指定 `--chrome` 且去除首尾空白后非空，则使用该路径
    + 否则使用默认路径：`/mnt/c/Program Files/Google/Chrome/Application/chrome.exe`
+ 启动参数如下：
    + `--remote-debugging-address=0.0.0.0`
    + `--user-data-dir=<Windows路径>`
    + `--no-first-run`
+ 新建实例时：
    + 使用 `--remote-debugging-port=0`，由 Chrome 自行分配端口
+ 原地重启旧实例时：
    + 使用旧记录中的固定端口，即 `--remote-debugging-port=<旧port>`
+ `--headless=true` 时追加：
    + `--headless=new`
+ `--headless=false` 时不追加任何 headless 参数
+ Chrome 以异步方式启动，标准输入、标准输出、标准错误均重定向到 `/dev/null`

### 就绪判定与超时
+ 单次实例创建/重启的整体等待上限为 30 秒
+ 轮询间隔固定为 5 秒
+ 新建实例时，先从 `<profileUnix>/DevToolsActivePort` 读取端口号
+ 读取到端口后，再通过：
    + `curl -s http://localhost:<port>/json/version`
  判断 CDP 是否可用
+ 每次 `curl` 探活的命令超时为 5 秒
+ 一旦探活成功，返回：
    + `pid`
    + `port`
    + `ws`
    + `http`
    + `user-data-dir`
+ 若 30 秒内始终未就绪，则返回最后一次错误；如果没有更具体错误，则返回：
    + `{"status":1,"message":"cdp not ready after 30s"}`

### 输出约定
+ 成功时输出：
```json
{"status":0,"pid":1234,"port":9222,"ws":"ws://localhost:9222/devtools/browser/...","http":"http://localhost:9222","user-data-dir":"C:\\temp\\chrome_ab12"}
```
+ 失败时输出：
```json
{"status":1,"message":"错误原因"}
```
+ 成功输出中的 `http` 字段来自本地拼装，格式固定为 `http://localhost:<port>`，不是直接抄写 `/json/version` 响应体中的其他字段

### 失败清理语义
+ 仅当“新建实例”路径中为新 profile 启动失败，或等待就绪失败时，程序会尝试删除刚刚新建的 `user-data-dir`
+ 对于旧记录对应的 profile：
    + 当前实现只会尝试清理锁文件并原地重启
    + 不会在探活失败时直接删除旧 profile
    + 也不会先删除数据库旧记录；后续成功时由 upsert 覆盖旧值

### 非本次实现范围
+ 当前实现未覆盖以下内容，因此不应继续作为本迭代的硬性要求：
    + `help` 子命令与完整插件使用手册输出
    + `USER_GUIDE.md` 编写
    + “探活失败后先删除旧记录并删除旧 user-data-dir 再继续”的强制流程
+ 若后续需要补齐这些能力，应以新增迭代需求的形式单独描述
