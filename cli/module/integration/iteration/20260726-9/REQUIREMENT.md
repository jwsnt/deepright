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
+ 图片编辑保存继续收口到既有 `POST /api/edit`，不新增图片专用 HTTP 接口。
+ 当请求为 `agentId=<agentId>`、`path=images/<filename>.png` 且 `saveAsNew=true` 时，服务端必须仅在该 Agent 的工作区内创建 `images/` 目录并写入 PNG 二进制内容；`images/` 不存在时自动创建。
+ 服务端必须在扩展名前追加高精度时间戳生成最终文件名，返回 `savedAs` 系统绝对路径；不得覆盖原图片或同名历史导出。
+ 服务端继续拒绝绝对路径、`~`、`..` 路径逃逸、目录目标、跨 Agent 路径与工作区外符号链接；图片保存失败时不得创建工作区外文件。

### 技术约束
+ PNG 必须沿用既有二进制扩展名处理：请求 JSON 中的 `content` 为 Base64，服务端解码后以二进制写入；不得按 UTF-8 文本写入或转换图片内容。
+ `saveAsNew=true` 仍由服务端生成最终时间戳，前端传入的 `path` 只作为工作区内的目标目录与基础文件名；响应中的 `savedAs` 为复制到系统剪贴板的唯一权威路径。
+ 需要覆盖自动创建 `images/`、服务端时间戳命名、二进制内容不变、源文件不变及路径/符号链接逃逸拒绝的回归测试。

### 编写代码
+ 最小范围更新，不新增外部依赖

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`，说明源码占位符、配置端口、重新构建后的发布文档地址、`--port` 运行时覆盖与已打包文档之间的关系，以及 macOS 与 WSL2 发布位置。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求，不记录实现过程。
