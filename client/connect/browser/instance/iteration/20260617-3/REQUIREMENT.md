### 第一性原则
+ 仅可以新增/更新/删除browser（ ../../..）目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../../../DESIGN.md
+ 本模块设计文档：../../../../DESIGN.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 需求介绍
+ 修改命令start, 如果系统为Windows WSL/WSL2则：
    + 关闭当前所有Chrome进程（记住是所有），包括integration自身开启动Chrome
    + 复制Windows系统默认Chrome的User Data目录到c://ProgramData/deepright/chrome_def
        + 如果c://ProgramData/deepright/chrome_def已存在则删除重新复制
        + 复制时参考MAC版本的复制逻辑，仅保留必要的目录和文件
        + 复制后删除文件锁
    + 如果复制失败，不要终止命令start，而是需要在日志中记录原因
    + 复制成功后重新调用主应用integration start，打开新Chrome

#### 插件帮助
+ 必须实现help来提供完成的插件使用手册
``` 假设插件可执行程序为a
./a help
```

### 撰写手册
+ 编写USER_GUIDE.md

### 同步代码
+ ../../../../browser/REQUIREMENT.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 编写代码
+ 以Golang编写以上代码，要求：
    + 代码简洁，包体积越小越好
    + 能用开源包的就用开源包
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写
+ 严格遵守指纹需求：../../../CHECK.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 同步代码：../../../../../integration/REQUIREMENT.md（每次都要同步更新代码）
