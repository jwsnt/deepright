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
+ 新增 `integration backup-clean` CLI 子命令，用于定期清理每个 Agent 工作目录下 `User.md` / `Soul.md` 的备份文件
+ 备份文件通常文件名中带 `bak` 或时间戳；如果仍停留在 Agent 工作目录根部，且最后更新时间距离当前时间已超过 `24h`，则自动 `mv` 到该工作目录下的 `bak/`
+ 如果工作目录下不存在 `bak/`，则在归档时自动创建
+ 清理 `bak/` 目录中的历史备份；当文件最后更新时间距离当前时间已超过 `72h` 时，自动删除
+ 仅处理当前 Agent 自己 workspace 下的文件，不影响其他目录和其他模块

### 技术实现
+ CLI 收口：
    + 通过 `integration` 主二进制新增 `backup-clean` 子命令，保持 integration 的二进制与 CLI 收口原则
    + 命令支持 `--agent-dir`、`--archive-after`、`--delete-after`
    + 默认阈值分别为 `24h` 和 `72h`
+ Agent 根目录解析：
    + 复用 integration 现有的 Agent 根目录解析逻辑
    + `--agent-dir` 未传时，继续按主应用 `config/config.json`、环境变量 `AGENT_DIR`、默认目录的既有优先级回退
+ 扫描范围：
    + 仅扫描 Agent 根目录下每个合法 Agent 子目录
    + 仅检查每个 workspace 根目录下的直接文件，以及该 workspace 下的 `bak/` 目录
    + 不把当前生效中的 `USER.md`、`SOUL.md` 当作备份文件处理
+ 备份文件识别：
    + 文件名需与 `user/soul` 语义相关
    + 同时文件名中必须带 `bak`，或带可判定为时间戳的数字串
    + 例如 `USER.md.bak`、`Soul-20260701-120000.md`、`SOUL_20260628_120000.md` 需要命中
    + 如 `USER_GUIDE.md` 这类普通文档不能误判为备份
+ 归档规则：
    + 命中备份文件且最后更新时间超过 `archive-after` 时，移动到 `workspace/bak/`
    + 若 `bak/` 中已存在同名文件，需要自动避让，按 `name.1.ext`、`name.2.ext` 递增重命名，避免覆盖历史备份
+ 删除规则：
    + 对 `bak/` 目录中的文件按最后更新时间检查
    + 超过 `delete-after` 的文件直接删除
    + `bak/` 不存在时跳过，不报错
+ 输出与验证：
    + CLI 输出 JSON，总结 `agentDir`、`agentCount`、`archived`、`archivedCnt`、`deleted`、`deletedCnt`
    + 需要补充测试，覆盖备份识别、归档、重名避让、72 小时删除以及帮助文案
+ 启动行为：
    + `backup-clean` 属于轻量本地命令，应避免依赖插件运行时初始化
    + `integration --help`、`integration file-last-update`、`integration backup-clean` 这类轻命令保持可直接运行，不因运行时插件目录缺失而失败

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
