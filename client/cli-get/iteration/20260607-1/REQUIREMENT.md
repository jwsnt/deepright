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
+ 为每个Agent+Chat保存一个sandbox_exe的boolean型变量，默认为false
    + 如果sandbox_exe=false，cli/get和cli/pub保持原逻辑（不要破坏任何原逻辑）：cli/get获取待执行命令->执行命令->cli/pub提交
    + 如果sandbox_exe=true，cli/get如果有待执行命令，通过CLI_SANDBOX执行命令->cli/pub提交
+ AgentId和ChatId从cli/get的响应报文获取
    ```
    "agentId": {
        "type": string,
        "description": "AgentID"
    },
    "chat": {
        "type": string,
        "description": "ChatID，会话ID"
    }
    ```
    + cli/get响应报文：../../REQUIREMENT.md
+ 沙盒应用程序相对与主应用程序路径由--sandbox_app指定（或从config/config.json读取）
```
$sandbox_app -cmd "cat hello.txt | wc -l"
```
+ Sandbox 接口需求：../../sandbox/REQUIREMENT.md
+ Sandbox MAC需求：../../sandbox/mac/REQUIREMENT.md
    + MAC Sandbox的执行逻辑
    ``` 如果$sandbox_app为/Users/deepright/CLI_SANDBOX.app
    /Users/deepright/CLI_SANDBOX.app/Contents/MacOS/CLI_SANDBOX --cmd "cat ~/Desktop/test.txt"
    ```
    ```
    /Users/deepright/CLI_SANDBOX.app/Contents/MacOS/CLI_SANDBOX --cmd "ls ~/Documents"
    ```
    + 一定要执行：.app/Contents/MacOS/CLI_SANDBOX，不要执行源码版、`go run`、普通二进制，也不要直接执行`./CLI_SANDBOX.app`


### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
    + 使用文件名为data的sqlite存储，并使用连接池
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




