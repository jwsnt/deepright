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
+ 如果cli/get有待处理任务，则在待处理任务完成后，提交cli/pub前从message_insert（插入消息）数据库中获取cli/get对应`ChatId`的状态为`待上传`，并与cli/pub一起上传，最多一次获取并上报5条：
    + cli/get待处理任务：../../REQUIREMENT.md
    ```
    "chat": {
        "type": string,
        "description": "ChatID，会话ID"
    }
    ```
+ 上报属性为:
```
{
    ... 其他属性
    "insert": [
        {"mid":xxx, "message":yyy}, .. 1到多条
    ]
}
```
+ 上报成功后，message_insert状态改为已上传

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
    + 使用文件名为data的sqlite存储，并使用连接池
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写
+ 最小范围更新

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写




