### 第一性原则
+ 仅可以新增/更新/删除connect（../..）同目录的文件和文件夹
+ 如非授权，禁止修改其他插件目录文件和文件夹
### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
    + browser、email、feishu等

### 迭代要求
+ Connect介绍：../../REQUIREMENT.md
+ Connect手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 关联需求
+ 飞书插件：../../feishu/iteration/日期/REQUIREMENT.md

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 重要提示
+ 设计前需要仔细阅读Connect的设计，需要兼容方案
    + Connect介绍：../../REQUIREMENT.md

### 需求介绍
+ 为connect模块增加新命令list-meta，获取当前所有已配置的插件meta
    + ./connect meta-create需求：../../REQUIREMENT.md
+ 案例
```
./integration connect meta-list 返回 [{"key":"feishu","name":"飞书","meta":{...},"stream":true...},{"key":"mail","name":"邮件","meta":{...},"stream":true...}]
```
    + 插件标识统一原则：展示名（name）可以是中文，系统主键（key）必须稳定唯一，所有运行时链路只能用主键，不能混用展示名
        + 参考飞书展示：../../feishu/iteration/20260504-2/REQUIREMENT.md
    + 最终用户验收与文档应优先写integration顶层入口；connect list-meta仅作为内部实现或兼容说明

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



