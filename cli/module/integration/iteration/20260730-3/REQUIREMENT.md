### 第一性原则
+ 仅可以新增/更新/删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../build.sh`、`../../../build/install.ps1`、`../../../build/USER_GUIDE.md`、`../../../build/USER_GUIDE.txt` 与 `../../../config/app/API.md`、`../../../config/app/CANVAS.md`、`../../../config/app/DESIGN.md`。

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
+ 不新增外部依赖，不破坏既有 API 的字段兼容性。

### 需求介绍
+ 在主应用静态 `config/config.json` 的 `miniapp` 对象中新增必填正整数 `recover`，单位为分钟；发布默认值为 `30`。
+ Integration 启动后立即执行一次迷你应用参考文档巡检，随后每隔 `miniapp.recover` 分钟巡检一次。配置缺失、不是对象、`recover` 缺失、非正整数或超出 `time.Duration` 可表示范围时，禁用该后台巡检并记录明确错误；不得自行回退到其他间隔。
+ 巡检范围为 `agent-dir` 下每个合法 Agent 的 `app/API.md`、`app/CANVAS.md`、`app/DESIGN.md`。源文件必须使用当前运行 `default-dir/app/` 下对应文件，即新建 Agent 时复制的同一份发布文档；不得使用 Agent 工作目录中的副本、浏览器请求或源码树文件作为恢复源。
+ 缺失、非普通文件、内容不同或权限不同的目标文件视为已改动，仅恢复该一个文件；未变化的文件不得重写。恢复时应保留源文件内容和权限，并拒绝通过 Agent 或 `app` 目录的符号链接写出 Agent 工作目录。
+ 单个文件恢复失败不得阻止同一 Agent 的其它受保护文件或其它 Agent 继续巡检；恢复及错误均写入 Integration 日志，包含 Agent、文档名和路径，不暴露文档内容。

### 编写代码
+ 最小范围更新，不新增外部依赖。

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`，说明 `miniapp` 的配置格式、`miniapp.recover` 的单位和生效方式、三份受保护文档的范围与单文件恢复语义、`/api/runtime_config` 的受控透传语义、静态配置路径、服务地址 SQLite 存储、优先级、重置行为、WSL/目录发布兼容性以及 `.app` 内配置不可修改的原因。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求，不记录实现过程。
