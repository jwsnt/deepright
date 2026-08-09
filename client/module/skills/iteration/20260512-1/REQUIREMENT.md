### 第一性原则
+ 仅可以新增/更新/删除skills（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ skills介绍：../../REQUIREMENT.md
+ skills手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 每分钟扫描skills目录所有子目录及其子孙目录，检查`SKILL.md`文件，是否可以正确解析
+ 如果不能解析则记录SKILLS解析错误原因附加时间保存在数据库中，如果周期检查可以正确解析了则删除SKILLS解析错误提醒
    + Skills需求：../../REQUIREMENT.md
+ SKILLS解析提醒属性：
    + 错误SKILL.md的路径
    + 错误原因
    + 时间

### 编写代码
    + 使用文件名为data的sqlite存储，并使用连接池，避免每次都新建连接
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



