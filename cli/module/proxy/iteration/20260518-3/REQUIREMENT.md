### 第一性原则
+ 仅可以新增/更新/删除proxy（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Proxy介绍：../../REQUIREMENT.md
+ Proxy手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 通过META_ID找到原始add-request，使用对应插件send命令回复任务响应前，对响应报文进行JSON标准化处理：
    + 回复send命令：../20260518-3/REQUIREMENT.md
+ 检查SSE响应是不是以```json开头或```开头并且以```结尾的Markdown格式，如果是需要去掉Markdown格式
    + 案例
    ```json
    {
        "hello": "world"
    }
    ```
        + 标准化为：{"hello": "world"}的仅包含json内容的数据
    + 案例
    ```
    {
        "today": "sunday"
    }
    ```
        + 标准化为：{"today": "sunday"}的仅包含json内容的数据
    + 案例
        ```json
        [
            {
                "hello": "world"
            }
        ]
        ```
            + 标准化为：[{"hello": "world"}]的仅包含json内容的数据

    + 案例
    ```
    [
        {
            "today": "sunday"
        }
    ]
    ```
        + 标准化为：[{"today": "sunday"}]的仅包含json内容的数据
+ 如果SSE响应不为Json Object（{}）或Json Array（[]）或标准化失败，则使用原始SSE响应

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



