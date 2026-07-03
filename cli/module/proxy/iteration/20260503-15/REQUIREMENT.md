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
+ 在定时任务中将处于待处理状态三方的add-request产生的消息转为一次性任务，同时标记消息为已启动或已完成
    + 定时任务需求：../20260429-1/REQUIREMENT.md
    + 三方消息需求：../../../connect/REQUIREMENT.md
+ 属性映射：
    + 任务内容：当前指定name的插件所有待处理消息的拼接字符串
    ``` 以下拼接为一条消息内容：你好 [image]/a/b/c.png [file]/d/e/f.csv 天气怎么样
    消息1：你好
    消息2：[image]/a/b/c.png
    消息3：[file]/d/e/f.csv
    消息4：天气怎么样
    ```
    + 模型选择：来自于三方启动时向meta-create注册的模型
    + AgentID：来自于三方启动时向meta-create注册的Agent
    + 思考模式：来自于三方启动时向meta-create注册的思考模式
    + 会话ID：来自于三方启动时向meta-create注册的思考模式
    + 执行时间：生成后的时间（约等于立即执行）
    + 任务类型：插件的Key（非展示名称）
    + META_ID：被拼接的最后一条消息add-request消息在自身数据库的ID
    + 状态：正常转换（合并）的状态为已启动
+ 异常状态：
    + 如果当前add-request拼接字符串不含文本内容（仅含图片或文件）则不处理，也不要修改状态（待后续累计）
    + 将超过当前5分钟还处于待处理的add-request消息（通常都为过期消息或仅有图片或文件的消息）合并为一条备忘录明细，同时标记（多个）add-request消息为已过期，备忘录明细消息状态是无需启动（该备忘录仅用于查看）
+ 检查备创建后待启动的忘录明细对应的Agent是否存在且没删除，模型是否存在且填写了密钥，如果被删除则不运行任务明细
+ 立即执行：正常转换（合并）状态为已启动的任务明细需要立即执行，同时修改任务明细为已启动，而不是等待下一个周期
+ 消息通知
    + 如果每分钟执行的定时任务在本次为三方插件生成了至少一条状态为待执行的备忘录明细则使用该插件的send命令回复（推送）由参数--reply指定的内容，默认为"<开始执行>可通过新消息更新任务"（不含"）
        + 执行提醒消息内容：
            + 启动时如果配置了--reply，那么执行开始的推送消息就用--reply配置的文案
            + 如果没有配置，那么执行开始的推送消息就用 <开始执行>可通过新消息更新任务
        + 因为任务内容可能会有多条原始消息拼接，这里使用META_ID关联的原始消息
        + 插件的推送二进制程序需要通过：meta-list命令来获取--callback参数的应用程序绝对路径，并执行send命令，必须使用参数message带上原消息报文
            + meta-list需求：../../../connect/iteration/20260504-2/REQUIREMENT.md
            + 例如飞书callback为feishu：../../../connect/feishu/iteration/20260505-1/REQUIREMENT.md
                ```
                plugins/feishu send \
                  --message '{"id":1,"name":"feishu","request":"你好","rawRequest":"{\"schema\":\"2.0\",\"header\":{\"event_id\":\"evt_demo\",\"create_time\":\"1777890990994\"},\"event\":{\"message\":{\"chat_id\":\"oc_xxx\",\"content\":\"{\\\"text\\\":\\\"你好\\\"}\",\"message_id\":\"om_x100b50af4b4d98b4c4d2ae7728edd20\",\"message_type\":\"text\",\"create_time\":\"1777890990639\"},\"sender\":{\"sender_id\":{\"open_id\":\"ou_xxx\"},\"sender_type\":\"user\"}}}"}' \
                  --content "收到，这是一条回复"
                ```
        + 任务开始或完成回推必须按META_ID精确定位原始add-request，并以该原始request作为唯一父消息，禁止通过状态近似匹配其他消息
        + 即使一次生成了多条，每次轮询单插件也仅发送一次，保证每个任务明细仅推送一次开始通知

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



