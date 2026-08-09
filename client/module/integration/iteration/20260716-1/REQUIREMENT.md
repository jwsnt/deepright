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
+ Integration 服务的 `device` 需要按统一优先级解析，并在服务运行期间支持从 `config/config.json` 热更新：
    + 非空命令行参数 `--device` 优先级最高
    + 未传有效 `--device` 时，读取主应用 `config/config.json` 的 `device`
    + 上述两项均未配置时，使用系统自动生成的 device
+ `config/config.json` 示例：
```
{
  "host": "https://www.deepright.cn",
  "version": "1.0",
  "device": "1234",
  "skills": ["__internal_cron", "__internal_token"],
  "http": {"heartbeat": 3, "debug": false}
}
```
+ 服务启动后需要全局缓存当前有效 device；每 `60秒` 检查一次 `config/config.json` 是否更新。
+ 配置刷新成功或失败均需要记录日志；配置文件未变化时不读取、不更新、不记录日志。
+ 本需求只调整 Integration 服务进程；`proxy`、`cli-get`、`agent-scanner` 的既有 device 解析行为不在本次范围内。

### 技术实现
+ 配置来源与优先级：
    + `--device` 仅在 trim 后非空时视为有效；有效值始终覆盖 `config/config.json.device`，包括配置文件热更新后的值。
    + `config/config.json.device` 只有 JSON 值为字符串且 trim 后非空时视为有效配置。
    + 键缺失、`null`、非字符串、空字符串和仅空白字符串均视为“未配置 device”，继续回退到系统自动生成值。
    + 系统自动生成逻辑复用 `agentcore.GenerateDeviceID()`；仅在服务启动时调用一次，并将结果作为本进程固定的 fallback device。运行中不得因配置刷新重复生成。
    + 有效值计算固定为：`有效 --device > 有效 config device > 启动时生成的 fallback device`。
+ 启动与配置写回：
    + 沿用现有启动顺序：先加载 `config/config.json` 作为 flag 默认值，再解析命令行参数，因此 `--device` 可覆盖配置值。
    + 沿用现有 `writeRuntimeConfig` 行为：若指定有效 `--device`，该值会写回同一份 `config/config.json`；这是预期的覆盖和持久化行为。
    + 服务启动完成前必须完成首次配置读取和 fallback device 生成，随后立即发布首个全局 device 快照。
+ 全局缓存与并发模型：
    + 在 `Config` 中新增 Integration 专用的 device 运行时状态，不直接在运行中写入现有 `cfg.Device` 字符串。
    + 运行时状态至少保存：启动参数 device、最近一次成功读取的 config device、启动时 fallback device、当前有效 device、最近一次已处理的配置文件版本。
    + 当前有效 device 使用标准库 `sync/atomic` 的 `atomic.Value` 保存不可变快照；请求路径只执行一次原子读取，不使用互斥锁，不要求跨请求强一致。
    + 心跳、`/v1/chat/completions`、`/api/deviceId`、Agent/技能/文件接口、cron 执行和 Agent 校验等所有构建 Agent metadata 的路径，必须通过同一 `effectiveDeviceID()` 读取快照后再调用 `getAgentOutput` / `getAgentOutputForChat`。
    + 单个请求在开始构建 metadata 时取得的 device 快照可继续使用到该请求结束；60 秒刷新期间已在执行的请求允许继续使用旧值，之后的新请求使用新快照。
    + `agentcore` 的 metadata 缓存键包含传入的 device；调用方必须传入上述有效 device，而不是空字符串，确保 device 变更后自然使用新的 metadata 缓存项。
+ 60 秒配置刷新：
    + 服务启动成功后启动一个后台定时任务，首次检查在启动完成后 `60秒` 执行，之后每 `60秒` 执行一次；服务关闭时随主 context 退出。
    + 使用配置文件版本标识（至少包含存在状态、修改时间、文件大小；可附加内容摘要）判断文件是否发生变化。版本未变化时直接返回且不记日志。
    + 检测到文件变化后再读取和解析 JSON：
        + JSON 解析成功时，将 `device` 按上述有效性规则更新为新的 config device，重新计算并原子发布当前有效 device；记录一次成功刷新日志，包含配置 device、有效 device 与其来源（`flag` / `config` / `system`）。
        + JSON 解析失败、读取失败或配置路径不可用时，不修改“最近一次成功读取的 config device”和当前有效 device；记录一次失败刷新日志，包含错误及保留的有效 device 来源。
        + 对同一个失败的文件版本只记录一次失败；文件版本再次变化后才重新尝试并记录结果，避免每 60 秒重复刷错误日志。
        + JSON 有效但 `device` 缺失、`null`、非字符串或 trim 后为空，不属于刷新失败；将 config device 更新为空，按优先级切换到 `--device` 或启动时 fallback device，并记录成功刷新日志。
    + 即使存在有效 `--device`，配置文件仍按上述规则检查和记录刷新结果；日志需明确本次配置变化未改变有效 device，原因是命令行优先级更高。
+ 接口与行为：
    + `GET /api/deviceId` 返回当前原子快照中的有效 device。
    + 对上游的 heartbeat、聊天转发和 cron metadata 中的 `deviceId` 与同一时刻的接口读取遵循最终一致性；不承诺所有并发请求在刷新瞬间同时切换。
    + 不新增 CLI 参数、不改变已有 `--device`、`config/config.json` 或 metadata 的字段名与 JSON 协议。
+ 日志要求：
    + 配置文件版本发生变化且刷新成功时记录 info 日志；包含 `device refresh succeeded`、配置来源、当前有效值来源和是否发生有效值切换。
    + 配置文件版本发生变化且刷新失败时记录 warning 日志；包含 `device refresh failed`、错误信息、保留的有效值来源。
    + 文件版本未变化不记录刷新日志；日志中不得输出完整 metadata、token 或其他无关敏感信息。
+ 测试要求：
    + 覆盖 `--device > config device > fallback device` 三层优先级，以及有效 `--device` 写回配置文件后的重启行为。
    + 覆盖 config device 的缺失、`null`、数字/对象等非字符串、空字符串和仅空白字符串均回退到 fallback device。
    + 通过可注入的系统 device 生成函数验证 fallback 仅在启动时生成一次，配置从有值切换为空后复用同一 fallback 值。
    + 覆盖配置文件变化后成功更新、JSON 损坏/读取失败后保留最近成功值、修复后恢复更新，以及同一文件版本不重复记录日志。
    + 覆盖有有效 `--device` 时配置变更只更新配置缓存和日志、不改变有效 device。
    + 覆盖 `/api/deviceId`、heartbeat 和至少一个 chat/cron metadata 路径读取当前有效 device；使用 `go test -race` 验证刷新与并发请求不存在数据竞争。

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写
+ 最小范围更新

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
