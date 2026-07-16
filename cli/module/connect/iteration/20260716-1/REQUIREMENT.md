### 第一性原则
+ 仅可以新增/更新/删除connect（../..）同目录的文件和文件夹
+ 如非授权，禁止修改其他插件目录文件和文件夹
### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
    + browser、email、feishu等

### 迭代要求
+ Connect介绍：../../REQUIREMENT.md
+ Connect手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 重要提示
+ 设计前需要仔细阅读Connect的设计，需要兼容方案
    + Connect介绍：../../REQUIREMENT.md

### 需求介绍
+ 右上角插件扇形开关：
    + 不展示 `remote` 插件
    + 不允许通过该入口打开 `remote` 的插件浮层
    + 不影响 `remote` 插件二进制、远程技能及命令行能力
+ Integration关联插件生命周期：
    + 应用启动（`integration`）时，从 `config/config.json` 读取 `associated` 数组
    + `associated` 中的插件作为常驻进程，随 Integration 异步单独启动，不能阻塞 Integration 的监听、就绪或界面启动流程
    + 调用关联插件 `start` 时，必须传递实际 Integration 二进制路径：`--connect-bin <integration-bin>`
    + 关联插件启动失败只记录日志，不中止 Integration 启动
    + Integration 收到退出信号、通过关闭接口退出，或执行 `integration stop` 时，必须停止同一批关联插件，调用 `stop` 时也必须传递 `--connect-bin <integration-bin>`
    + 关联插件停止失败只记录告警，不能阻止 Integration 自身退出
    + 转发请求和 `cli/get` 的 `metadata.plugins` 必须包含固定的 `remote`，并合并当前实时运行的 Connect Meta 插件；任意插件启动或停止均不能移除 `remote`
+ 关联配置示例：
```json
{
  "associated": [
    "remote"
  ]
}
```
+ Remote超时配置：
    + `remote` 插件只从 `config/config.json` 的 `remote` 节读取超时配置，不再从插件 Meta 配置读取
    + `remote.exec_timeout` 控制 SSH `exec` 超时；`remote.scp_timeout` 控制 `scp` 超时
    + 两个字段的单位均为秒
    + 字段缺失、空值、非正整数或配置文件不可用时，分别默认 `300` 秒
    + 配置示例：
```json
{
  "remote": {
    "exec_timeout": 300,
    "scp_timeout": 300
  }
}
```

### 编写代码
+ 以Golang编写以上代码，要求：
    + 所有Connect模块复用启动时初始化的全局数据库连接，禁止每次请求单独打开和关闭数据库文件
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
