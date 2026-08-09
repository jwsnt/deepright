### 第一性原则
+ 仅可以新增/更新/删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../build.sh` 与 `../../../config/app/API.md`、`../../../config/app/CANVAS.md`、`../../../config/app/DESIGN.md`。

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Integration介绍：../../REQUIREMENT.md
+ Integration手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ `config/app/API.md`、`config/app/CANVAS.md`、`config/app/DESIGN.md` 是随 Integration 发布包分发的文档。源码中的所有 `http://127.0.0.1:8080` 和 `http://localhost:8080` 都必须统一写为唯一占位符 `http://localhost:#port`；不得保留 `127.0.0.1`、写死的 `8080`，也不得修改其它协议、主机或端口的 URL。
+ `#port` 只允许存在于源码文档，表示“构建时从主应用配置取得的端口”，不是运行时 URL。`config/app/API.md` 必须说明该占位符的含义，以及发布文档使用配置端口、运行时仍可通过 `--port` 覆盖监听端口的区别。
+ 执行 `cli/module/build.sh` 时，每一个目标发布目录在复制 `config/` 后，都必须从源码 `config/config.json` 读取顶层 `port` 属性，并仅在该发布目录的 `config/app/API.md`、`config/app/CANVAS.md`、`config/app/DESIGN.md` 中将全部 `http://localhost:#port` 替换为 `http://localhost:<port>`。
+ `port` 必须能解析为纯数字。`config/config.json` 缺失、缺少 `port` 或 `port` 不是数字时，构建必须以清晰错误失败；不得静默改回 `8080`、留存占位符或产出端口不确定的发布文档。
+ 构建替换不得写回或修改源码 `config/app/` 文档，也不得修改 `config/config.json`。除上述三个发布文档和目标发布目录外，不得替换其它文件内容。
+ macOS 与 Linux/Windows WSL2 发布流程必须使用同一份替换规则：macOS 最终应用包中的 `Contents/Resources/config/app/` 文档应为实际端口；WSL2 安装载荷根目录中的 `config/app/` 文档也应为实际端口。平台包装、签名、沙盒和安装流程不应改变端口替换语义。

### 编写代码
+ 最小范围更新，不新增外部依赖。复用 POSIX shell 与现有构建依赖完成 JSON 端口提取和文本替换。
+ 替换逻辑必须位于共享的目标发布构建流程，保证 macOS x86/arm 与 Linux x86/arm（含其 WSL2 安装载荷）全部覆盖；不得为各平台维护重复、可能漂移的替换实现。
+ 为构建脚本补充或执行验证：shell 语法有效；源码目标文档只含占位符而不含上述两种 `:8080` 本机地址；使用测试配置复制到临时发布目录后，三个发布文档均不含占位符，并全部使用该配置端口。

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`，说明源码占位符、配置端口、重新构建后的发布文档地址、`--port` 运行时覆盖与已打包文档之间的关系，以及 macOS 与 WSL2 发布位置。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求，不记录实现过程。
