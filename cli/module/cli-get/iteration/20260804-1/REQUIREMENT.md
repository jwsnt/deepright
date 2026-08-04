### 第一性原则
+ 仅可以新增/更新/删除`../../目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Cli-Get介绍：../../REQUIREMENT.md
+ Cli-Get手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 降低独立 `cli-get` 进程在待机状态下向上游 `POST /cli/get` 的访问频率。
+ `config/config.json` 必须支持如下配置：
    ```json
    {
      "get": {
        "sleep": 15000,
        "await": 30000,
        "check": 10
      }
    }
    ```
    + 单位均为毫秒。
    + `get.sleep` 用作心跳失败、Agent 扫描失败及本地任务队列满时的等待基数。
    + `get.await` 用作每次成功 `/cli/get` 响应后的下一次上报间隔。
    + `get.check` 用作收到需执行 `cmd` 后、重新进入 `await` 前允许连续快速检查的无 cmd 成功响应次数。
    + `sleep`、`await` 必须是非负整数，`check` 必须是正整数。配置缺失时保留兼容默认值，`await` 默认 `30000`、`check` 默认 `10`。
+ `--sleep` 的优先级：
    + 显式传入 `--sleep` 时使用命令行值。
    + 未显式传入 `--sleep` 时使用 `config.json.get.sleep`。
    + 不新增 `--await` 参数，`await` 仅从 `config.json.get.await` 获取。
+ 独立 `cli-get` 是独立进程，无法读取 Integration 进程内的 SSE 原子计数；因此其待机策略固定为：
    + `/cli/get` 成功返回任务时，任务先进入现有本地 `taskQueue`，不等待执行或 `/cli/pub`；下一次请求是否立即发起由下列 cmd 检查规则决定。
    + 进程启动后必须先立即请求一次 `/cli/get`；如果该响应没有需执行的 `cmd`，等待 `get.await`。
    + 收到非空 `cmd` 时，必须原子重置连续无 cmd 计数，并立即请求下一次 `/cli/get`。
    + 收到过 `cmd` 后，每次成功响应无 `cmd` 都必须原子递增连续无 cmd 计数；计数小于 `get.check` 时立即请求下一次，达到 `get.check` 时等待 `get.await`。
    + 快速检查期间任一次响应再次有非空 `cmd`，必须原子重置计数并重新开始快速检查周期。
    + 不得为独立 cli-get 新增进程间数据库、文件锁、轮询接口或其他共享状态来读取 SSE 活跃数。
+ 下列既有异常和队列语义必须保持：
    + `/cli/get` 网络错误、超时、HTTP 非 200 或响应解析失败时，按 `--sleep` / `get.sleep` 的既有指数退避重试，最大上限保持不变。
    + 本地 `taskQueue` 满时，不发起 `/cli/get`，按 `--sleep` / `get.sleep` 等待后再次检查。
    + 执行 worker、发布队列、`/cli/pub` 重试、`ddl` 校验、沙盒、`subOps.exempted`、活跃命令和日志协议保持不变。
+ Integration 同步边界：
    + Integration 内置 cli-get 必须读取相同的 `get.sleep` 与 `get.await` 配置。
    + Integration 会额外按其进程内 SSE 原子计数决定是否跳过 `get.await`；该能力不要求也不允许复制到独立 cli-get 进程。
+ 验收要求：
    + 覆盖 `get.sleep`、`get.await` 读取和非法值回退/拒绝行为。
    + 覆盖 `--sleep` 显式传入时优先于 `get.sleep`。
    + 覆盖启动后的首个无 cmd 响应使用 `get.await`、收到 cmd 后的连续空响应快速检查、达到 `get.check` 后使用 `get.await`，以及新 cmd 原子重置计数。
    + 覆盖失败退避、队列满和 `/cli/pub` 重试策略不变。

### 编写代码
+ 以Golang编写以上代码，要求：
    + 复用既有配置读取、心跳循环和队列实现，不新增外部 Go 依赖
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写
+ 最小范围更新
+ 需要同步更新 integration 中对应实现与测试

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 并编写本迭代目录 `USER_GUIDE.md`，说明 `get.sleep`、`get.await`、`--sleep` 优先级以及独立进程的待机策略。

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
