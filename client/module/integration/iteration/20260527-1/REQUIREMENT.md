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
+ 技术规范的指导下收口：
    + 新增命令token，记录Token消费明细，需要保存至数据库，同时新增/api/consume?agentId=xxx&&starTime=yyy&&closeTime=zzz&&limit=aaa, 获取指定Agent在指定时间范围内的Token消费数据
        + Proxy需求：../../../proxy/iteration/20260527-1/REQUIREMENT.md
    ```
    integration token --thinking 1 --input 2 --total 3 --cache 4 --model deepseek --agentId A --function 完成任务
    ```
    ``` 执行返回（案例）
        {
          "record" : {
            "thinking" : 778,
            "input" : 9512,
            "total" : 10377,
            "cache" : 0,
            "model" : "deepseek",
            "agentId" : "A",
            "function" : "main",
            "timestamp" : 1779854014340
          },
          "status" : 0
        }
    ```
### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写