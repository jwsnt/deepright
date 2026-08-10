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
+ 在定时任务中检查最近24小时已完成且类型不为cron任务明细
    + 定时任务需求：../20260429-1/REQUIREMENT.md
+ 通过任务明细的META_ID从Connect模块的add-request原始数据中找到对应的原始报文
    + 三方消息需求：../../../connect/REQUIREMENT.md
    + 三方消息转任务明细需求：../20260503-15/REQUIREMENT.md
+ 如果该三方消息状态为已启动，则使用三方消息对应的插件（plugin）来回复（send）任务明细响应
    + 插件的推送二进制程序需要通过：meta-list命令来获取--callback参数的应用程序绝对路径，并执行send命令，必须使用参数message带上原消息报文
    + 先通过检查--callback指定应用程序的--help命令，查看是否支持send，如果不支持或不存在则表示不需要回复，为插件记录日志（x.log），不需要回复
            + meta-list需求：../../../connect/iteration/20260504-2/REQUIREMENT.md
            ``` 插件a
            plugin/a send --message {} --content {}
            ```
    + 例如飞书需求：../../../connect/feishu/iteration/20260505-1/REQUIREMENT.md
+ 发送成功后将三方消息状态设置为已回复，将该三方消息时间之前（更早）且状态为已启动的消息设置为已完成
+ 任务开始或完成回推必须按META_ID精确定位原始add-request，并以该原始request作为唯一父消息，禁止通过状态近似匹配其他消息

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



