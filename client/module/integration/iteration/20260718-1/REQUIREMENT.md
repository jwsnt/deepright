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
+ Integration 启动时必须确定唯一的最终生效端口：显式 `--port` 优先；未显式传入时读取 `config/config.json` 的 `port`；两者均未配置时沿用现有内置默认端口。
+ 所有 Integration 发往上游的请求都必须在顶层 `metadata` 写入整数 `port`，值为该次进程启动后最终生效的端口，不得从外部请求体透传或信任该字段。
+ 覆盖范围：
    + 外部请求经 Integration 转发到上游的 `POST /v1/chat/completions`。
    + Integration 发往上游的 `POST /cli/get` 心跳请求。
    + Integration 内部执行的备忘录/定时任务，以及邮件、飞书等 Connect 任务最终发往上游的 `POST /v1/chat/completions` 请求。
+ `metadata.port` 与现有 Agent、会话、插件等 metadata 字段同级；不写入 `metadata.agent`、`metadata.agents[]` 或任务持久化数据。
+ 不新增 HTTP 接口、CLI 参数或配置项；只补齐现有请求转发的 metadata。

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
