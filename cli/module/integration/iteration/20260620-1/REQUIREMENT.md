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
+ 技术收口：
    + 修改/api/config和/api/edit接口，增加media属性，是一个Agent维度的JSON Object
    ```
    {
    ... Agent相关config配置中其他属性
    "media": {
        ... 多组属性
    }
    }
    ```
    + 如果media不会空，转发/v1/chat/completions时在agent数组中对应Agent属性需要带上media = {...}
    ```
    {
    "agents": [
        {...},
        {
            配置了media的Agent配置
            "media": {
                多组属性
            }
        }
    ]
    }
    ```
    + 如果media不会空，请求/cli/get时在agent数组中对应Agent属性需要带上media = {...}，属性位置同转发/v1/chat/completions
    + 每次都读取最新，不要缓存
+ Proxy需求：../../../proxy/iteration/20260620-1/REQUIREMENT.md

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
