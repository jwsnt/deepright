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
+ HTTP POST `/api/cron/create?agentId=xxx`：创建备忘录元数据时增加router_disable参数（boolean，默认true关闭）
    + Cron需求：../../../cron/iteration/20260524-1/REQUIREMENT.md
+ HTTP POST `/api/cron/detail/metadata?agentId=xxx`：查询元数据时也需要返回router_disable参数
+ 备忘录任务创建时，右上角SWARM开关与实际转发/v1/chat/completions的metadata.router_disable必须全链路一致
    + 映射规则固定为：
        + 开启SWARM 时，router_disable=false
        + 关闭SWARM 时，router_disable=true
+ 上述规则不仅要求创建接口入库正确，也要求任务实际执行并转发/v1/chat/completions时保持一致，禁止在执行阶段被Agent配置或其他默认值覆盖
+ 周期任务：
    + 创建时必须保存到task_meta.router_disable
    + 创建出的任务明细也必须保存到task_detail.router_disable
    + 后续由task_meta自动拆分出的新明细，必须继承所属元数据的router_disable
+ 一次性任务：
    + 创建时任务明细必须保存task_detail.router_disable
    + 实际执行时必须以该条任务明细的router_disable作为最终转发值
    + 任务执行器在扫描并执行task_detail时，转发/v1/chat/completions必须显式写入metadata.router_disable
+ 执行时metadata.router_disable的取值来源必须优先使用当前执行中的task_detail.router_disable
+ 即使Agent配置中的router_disable 与任务明细不同，实际执行转发时也必须以任务明细为准
+ 禁止在执行链路中丢失该字段，禁止回退为Agent config.json中的router_disable
+ 禁止用旧字段swarm替代最终执行字段

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 验收测试
+ 备忘录创建一次性任务，开启SWARM后，落库的任务明细router_disable=false，实际转发/v1/chat/completions时metadata.router_disable=false
+ 备忘录创建一次性任务，关闭SWARM后，落库的任务明细router_disable=true，实际转发/v1/chat/completions时metadata.router_disable=true
+ 备忘录创建周期任务，元数据和首批任务明细都要正确保存router_disable
+ 周期任务后续自动拆分出的新明细，router_disable必须与所属元数据一致

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



