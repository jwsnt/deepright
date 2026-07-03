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
+ 修改Agent+Chat的sandbox_exe变量为枚举值，默认为空（无值）
    + key：filepick
        + 用户选择目录（没有选择均认为没有权限）
    + key：net
        + 关闭网络
    + key：filepick_net
        + 两者都限
+ 如果Agent+Chat的sandbox_exe存在枚举值且为以上3个则调用指定key的沙盒（区分系统）
    + cli/get如果有待执行命令，通过指定key的CLI_SANDBOX执行命令->cli/pub提交
    + cli/get只在有待处理任务时才需要执行沙盒
+ 如果Agent+Chat的sandbox_exe不存在枚举值或不为以上3个则：
    + cli/get和cli/pub保持原逻辑（不要破坏任何原逻辑）：cli/get获取待执行命令->执行命令->cli/pub提交
+ 原需求逻辑：../20260607-1/REQUIREMENT.md
    + 修改为最新方案，不需要做兼容，数据库删除重建
+ MAC沙盒需求：../../sandbox/mac/iteration/20260609-1/REQUIREMENT.md
+ 如果cli/get响应报文开启了豁免（subOps.exempted=true）则不使用沙盒：cli/get获取待执行命令->执行命令->cli/pub提交
```
"subOps": {
    "exempted": {
        "type": boolean,
        "description": "是否豁免"
    }
}
```

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




