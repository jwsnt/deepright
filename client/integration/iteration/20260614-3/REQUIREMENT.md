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
    + 增加token get命令，获取最近N条，或指定时间段的token数据
        + 读取的数据为token在本地数据库存储的用量数据
        + 数据查询方式需要与接口/api/consume相同
    + 案例
    ``` 使用integration代理proxy，查询最新500条
    integration token --n 500
    ```
    ``` 使用integration代理proxy，查询2026-06-14 12:00:00至2026-06-14 14:00:00最新500条
    integration token --n 500 --start "2026-06-14 12:00:00" --close "2026-06-14 14:00:00"
    ```
    + 不能破坏现有token命令，如下
    ``` 使用integration代理proxy
    integration token
    integration token --provider deepseek
    integration token --agentId demo-agent --model deepseek-chat --function cli/get --thinking 10 --input 20 --total 30 --cache 5
    ```
    + 增加--help
    ```
    integration token get --help
    ```
    + Proxy需求：../../../proxy/iteration/20260614-2/REQUIREMENT.md

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写