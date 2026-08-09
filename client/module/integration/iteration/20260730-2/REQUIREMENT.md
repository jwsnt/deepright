### 第一性原则

+ 仅可以新增/更新/删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../build.sh`、`../../../build/install.ps1`、`../../../build/USER_GUIDE.md`、`../../../build/USER_GUIDE.txt` 与 `../../../config/app/API.md`、`../../../config/app/CANVAS.md`、`../../../config/app/DESIGN.md`。

### 技术规范

+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
+ 不新增外部依赖，不破坏既有 API 的字段兼容性。

### 需求介绍

macOS `DeepRight.app` 的签名包必须与可变状态隔离。服务启动、重启或更新本机服务地址后，均不得修改 `.app` 内的受保护资源，避免下一次启动被 macOS 以 `CODESIGNING / Taskgated Invalid Signature` 终止。

#### 配置定义与位置

- **静态主应用配置**：macOS `.app` 使用 `DeepRight.app/Contents/Resources/config/config.json`；WSL、Linux 和目录形式的 macOS 命令行发布使用可执行文件同级的 `config/config.json`。它是发布物提供的默认启动参数和静态功能配置来源。
- **服务地址覆盖项**：只有用户主动修改的 `host` 可以持久化，保存在共享 SQLite 的 `integration_persistent_settings` 表中；macOS `.app` 的数据库路径为 `~/Library/Containers/cn.deepright.integration/Data/Library/Application Support/deepright/data`。
- Agent 工作目录中的 `config.json` 与主应用配置无关；不得作为主应用配置或服务地址的回退来源。

#### 读取规则

- 所有读取主应用 `config/config.json` 的功能必须使用统一的静态配置解析规则；macOS `.app` 始终读取 `Contents/Resources/config/config.json`，不得改读运行目录中的同名文件。
- `GET /api/runtime_config`、`skills_git_install`、`miniapp`、`install_app`、技能清单和其它静态配置消费者必须读取当前发布包（或目录发布）的静态配置，确保升级后能立即加载新增或修改的字段。
- 服务地址生效优先级固定为：显式 `--host` > SQLite 中用户保存的 `host` > 静态 `config.json.host` > 内置默认值 `https://www.deepright.cn`。
- 启动期间派生的 `app`、`app-dir`、`resources-dir`、`db`、PID、日志路径、端口、超时、缓存、重试和设备等值必须每次重新计算或由静态配置/显式参数提供；不得写回主应用 `config.json`。

#### 写入规则

- 不得创建、读取或依赖运行目录中的 `config/config.json`。旧版本遗留的同名文件不得覆盖当前发布包的静态配置。
- `POST` / `PUT /api/host` 与 `integration host set` 必须先校验服务地址，再原子地写入共享 SQLite；数据库写入失败时不得切换当前运行时地址。
- `DELETE /api/host`、`POST /api/host?reset=true` 与 `integration host reset` 必须删除 SQLite 覆盖项，并立即恢复静态 `config.json.host`；静态值缺失时恢复内置默认值。
- 共享 SQLite、日志、PID、插件状态、浏览器复用标记和其它可变状态必须继续位于用户级运行目录或既有运行位置，不得落在 `.app` 包内。

#### 非 `.app` 兼容性

- WSL、Linux 和目录形式的 macOS 命令行发布仍从可执行文件同级 `config/config.json` 读取静态默认值；启动期派生字段不再写回该文件。
- 这些发布形态的用户服务地址同样保存到其共享 SQLite 中，不再写入 `config.json`。
- 本需求不改变 Agent 模板复制、Agent `config.json`、发布文档端口替换、接口字段兼容性或现有命令行参数名称。

#### 验收标准

- 对使用 Developer ID 签名的 macOS `.app`，首次启动和至少一次服务重启后，`codesign --verify --deep --strict` 仍成功；包内 `Contents/Resources/config/config.json` 与签名时字节内容一致，且包内不产生 `Contents/MacOS/data`、`data-shm`、`data-wal` 或运行时 `config.json`。
- 在运行目录预先存在旧版 `config/config.json` 时，新版仍读取包内静态配置；升级包中新增的配置字段能被 `GET /api/runtime_config` 及相关功能读取。
- 保存服务地址后，静态 `config.json` 保持不变，SQLite 中保存规范化后的 `host`，下一次启动读取该地址；重置后删除该覆盖项并恢复静态配置中的地址。
- 显式 `--host` 优先于 SQLite 覆盖项；WSL/目录发布的静态配置读取路径保持兼容，并且服务启动不再修改它。

### 编写代码

+ 最小范围更新，不新增外部依赖。
+ 为静态配置路径、SQLite 服务地址覆盖、旧运行时 `config.json` 忽略行为以及包内配置不变提供清晰边界和测试覆盖。

### 撰写手册

+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`，说明 `miniapp` 的配置格式、`/api/runtime_config` 的受控透传语义、静态配置路径、服务地址 SQLite 存储、优先级、重置行为、WSL/目录发布兼容性以及 `.app` 内配置不可修改的原因。

### 其他要求

+ `REQUIREMENT.md` 仅描述需求，不记录实现过程。
