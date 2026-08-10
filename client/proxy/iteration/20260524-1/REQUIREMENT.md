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
+ 虚拟文件系统在任意当前目录下点击“新建文件或目录”时，输入合法名称（如 data）必须创建到当前浏览目录，不得因为当前目录拼接后的相对路径中包含/而误报name contains invalid characters
+ 虚拟文件系统“新建文件或目录”弹窗中的输入框仍然只允许输入单个文件名或目录名，不允许用户直接输入 /、\、空白路径段、.、..
+ 当前浏览目录不是workspace根目录时，前端需要将“当前相对目录 + 用户输入名称”拼接成相对路径，再调用/api/agent/create
+ 创建成功后关闭弹窗、清空错误提示、刷新当前虚拟文件系统目录内容
+ 创建失败时在原错误提示位置展示后端返回内容，不要吞错或改成无关提示

### 接口要求
+ HTTP GET /api/agent/create?agentId=xxx&name=yyy&type=zzz
    + name 对外语义升级为“workspace 内相对路径”，不再只表示单个名字
    + name 允许形如 docs/data、tmp/a/b 这样的相对路径，以 / 作为目录分隔
    + name 的每个路径段都必须满足：
        + 非空
        + 不能是 . 或 ..
        + 不能包含空格和 \:*?"<>|
        + name 必须限制在当前 Agent 的 workspace 内，禁止绝对路径、~、../ 或其他越界写入
        + type=0 时创建目录，若父目录不存在则按相对路径自动补齐
        + type=1 时创建文件，若父目录不存在则按相对路径自动补齐
+ 现有“已存在”“Agent不存在”“参数缺失”等错误语义保持不变

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 验收测试
+ 当前位于 workspace 根目录时，新建目录 data 成功，实际创建为 <workspace>/data
+ 当前位于 docs 目录时，新建目录 data 成功，实际创建为 <workspace>/docs/data
+ 当前位于 docs 目录时，新建文件 `note.md` 成功，实际创建为 `<workspace>/docs/note.md`
+ 调用 /api/agent/create 传入 name=docs/data 时成功，不得返回 name contains invalid characters
+ 调用 `/api/agent/create` 传入 `name=../escape`、`name=~/escape`、`name=/tmp/a` 时必须失败，且不能在 workspace 外产生任何文件或目录

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



