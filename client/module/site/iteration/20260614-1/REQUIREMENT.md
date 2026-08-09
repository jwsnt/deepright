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
+ 在设置，新增Agent小图标后增加导入和导出小图标
    + 导出：点击后执行/api/agent/export?agent_id=xxx，导出当前Agent配置，并自动使用浏览器下载
        + 需要提示用户已下载
    + 导出：点击后打开上传文件或目录的浏览器窗口，选择后执行/api/agent/import，导入当前Agent配置
        + 需要检查同名Agent不能被覆盖
+ Integration需求：../../../integration/iteration/20260614-1/REQUIREMENT.md

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写