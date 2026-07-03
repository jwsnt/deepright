### 第一性原则
+ 仅可以新增/更新/删除当前需求文档（REQUIREMENT.md）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../DESIGN.md
+ 本模块设计文档：DESIGN.md

### 需求介绍
+ 遍历指定目录，提取目录和文件内容作为Agent元数据

### Agent
+ 指定目录为主目录，主目录下包含一至多个子目录（不包含嵌套子孙目录），每个子目录都代表一个Agent，该子目录既为Agent工作目录
    + 工作目录/workspace：该目录的绝对路径
    + Agent Name/agentId：该目录的名称
+ 每个Agent工作目录下的`skills`目录既为其技能目录，skills即为技能元数据列表
    + 技能元数据介绍：skills/REQUIREMENT.md
    + 技能元数据获取：skills/USER_GUIDE.md
    > 新增自 ./iteration/20260511-1/REQUIREMENT.md
    + skills每次都需要实时遍历指定目录及其子孙目录后提取文件内容，不要缓存
+ 每个Agent工作目录下的`SOUL.md`和`USER.md`文件内容分别对应`soul`和`user`，如果文件不存在则使用""
+ 每个Agent工作目录下的`config.json`对应`router_disable`、`thinking`和`description`，属性router_disable默认为true，thinking默认为false，description默认为""，如果json文件不存在则使用默认值
> 新增自 ./iteration/20260516-1/REQUIREMENT.md
> 新增自 ./iteration/20260610-1/REQUIREMENT.md
+ config.json新增`version`和`sandbox`属性，version仅在启动时获取一次并缓存，sandbox从存储沙盒的数据表实时获取Agent+ChatId维度（不缓存），不存在则使用""
```
{
    "description": string,
    "thinking": boolean,
    "swarm": boolean,
    "provider": string,
    "version": string,
    "sandbox": string,
    "router_disable": boolean
}
```
    + 沙盒模式需求：../cli-get/iteration/20260609-1/REQUIREMENT.md
    + Proxy需求：../proxy/iteration/20260608-1/REQUIREMENT.md
+ 以下全局共享属性，为所有Agent共享：
    + 设备编号/deviceId：设备ID，可以从命令行参数--device获取，或自动生成与系统硬件信息相关（硬件UUID），每次启动不变的唯一码
    + 终端类型/terminal：终端执行环境（例如zsh）
    + 网关编号/gateway：MAC网关地址（例如：arp -n $(route -n get default | grep gateway | awk '{print $2}') | awk '{print $4}'）
    + 系统类型/sys：操作系统（例如Darwin 23.4.0)
    + 时区信息/timezone：当前设备所在的时区，使用IANA时区标识（如Asia/Shanghai），不使用缩写（如CST）
    + APP路径/app：当前APP启动的绝对路径
    + Git路径/git：本机安装的git可执行文件绝对路径，获取不到则为空字符串；需区分Windows/Mac/Linux
    > 引用自 ./iteration/20260501_1/REQUIREMENT.md（新增 git 全局属性）
    + Agent 元数据需加上 plugins 信息
    > 新增自 iteration/20260509_1/REQUIREMENT.md
    + 获取应用知识库的绝对路径和最后更新时间，并将knowledge作为key放入json对象中，如果没有则不添加
        + Knowledge需求：../knowledge/REQUIREMENT.md
    > 新增自 ./iteration/20260510-1/REQUIREMENT.md

### 格式整理
+ 将每个agent声明的属性，整理为如下json格式，其中agents是array：
```JSON
{
    "timezone": string,
    "deviceId": string,
    "terminal": string,
    "git": string,
    "gateway": string,
    "sys": string,
    "app": string,
    "plugins": [string],
    "knowledge": {
        "lastUpdate": long,
        "path": string
    },
    "agents": [
        {
            "description": string,
            "workspace": string,
            "thinking": boolean,
            "provider": string,
            "agentId": string,
            "router_disable": boolean,
            "swarm": boolean,
            "version": string,
            "sandbox": string,
            "soul": string,
            "user": string,
            "skills": [
                技能元数据列表
            ]
        }
    ]
}
```

### Skills名称查询
> 引用自 ./iteration/20260419_2/REQUIREMENT.md
+ 从Agents Array获取指定AgentId的所有Skills名称列表
+ CLI：`--skills <agentId> <目录>`，输出 JSON 数组
+ API：`GetSkillNames()` 返回 `[]string`
+ agentId不存在时返回错误

### AgentId查询
> 引用自 ./iteration/20260419_1/REQUIREMENT.md
+ 从Agents Array获取所有AgentId列表
+ 从Agents Array获取指定AgentId的元数据
+ CLI：`--list <目录>` 列出所有AgentId，`--get <agentId> <目录>` 获取指定Agent元数据
+ API：`GetAgentIDs()` 返回 `[]string`，`GetAgentByID()` 返回 `*Agent` 或 nil
+ agentId不存在时 CLI 返回错误退出，API 返回 nil
+ 共享原有Agent元数据缓存，原有功能完全不变

### Git路径实时查询
> 新增自 ./iteration/20260512-1/REQUIREMENT.md
+ Git路径每次实时查询本机安装路径，不要缓存
+ 原需求：./REQUIREMENT.md 中 Git路径/git 属性

### Skills兼容数组compatibility
> 新增自 ./iteration/20260515-1/REQUIREMENT.md
+ skill属性需要兼容数组形式的compatibility属性
+ Skills需求：../skills/iteration/20260515-1/REQUIREMENT.md
+ compatibility支持字符串和字符串列表两种格式，统一整理为标准字符串输出

### 同步代码
+ ../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
    + 每次生成后缓存，缓存时间以命令行参数--agent-cache指定的毫秒数，默认10秒
    + `skills`字段每次都需要实时遍历目录生成，不受`--agent-cache`缓存影响
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 验证测试
+ 以`test-case`作为指定目录，验证代码生产内容是否符合：
    + deviceId、terminal、gateway、sys不为空，且取值正确
    + 2个Agent：a和b，名称分别为name=a, name=b
    + a的workspace和b的workspace均为绝对路径
    + a的技能：__internal_E和__internal_F
    + b的技能：__internal_A和__internal_F
    + a的soul=HELLO SOUL, user=HELLO USER
    + b的soul=GOOD SOUL, user=
    + AgentId查询验证 > 引用自 ./iteration/20260419_1/REQUIREMENT.md：
        + AgentId列表为：[a,b]
        + AgentId=a的元数据：workspace为绝对路径，soul=HELLO SOUL，user=HELLO USER，skills包含__internal_F
        + AgentId=b的元数据：workspace为绝对路径，soul=""，user=""，skills包含__internal_A和__internal_F
    + Skills名称查询验证 > 引用自 ./iteration/20260419_2/REQUIREMENT.md：
        + AgentId=a的Skills名称：[__internal_F]
        + AgentId=b的Skills名称：[__internal_A，__internal_F]

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 相关迭代：iteration/日期/REQUIREMENT.md
+ 同步代码：../integration/REQUIREMENT.md
> 合并截止：./iteration/20260610-1/REQUIREMENT.md，下次合并从此之后的新迭代开始
