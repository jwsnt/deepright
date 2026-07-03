### 第一性原则
+ 仅可以新增/更新/删除browser（ ../../..）目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../../../DESIGN.md
+ 本模块设计文档：../../../../DESIGN.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 需求介绍
+ 修改命令init, 如果系统为Windows WSL/WSL2则：
    + 修改--user-data-dir为c://ProgramData/deepright/chrome_$随机字符（原来是c://temp/chrome_$随机字符，不需要兼容和回退，实现最新代码）
    + 如果c://ProgramData/deepright/chrome_$随机字符不存在则从c://ProgramData/deepright/chrome_def复制
        + 如果c://ProgramData/deepright/chrome_def不存在或任意文件复制失败，则跳过并记录日志
        + 如果c://ProgramData/deepright/chrome_$随机字符目录已经存在则跳过不复制
    + 其他逻辑不变
+ 修改命令create, 如果系统为Windows WSL/WSL2则：
    + 修改--user-data-dir为c://ProgramData/deepright/chrome_$随机字符（原来是c://temp/chrome_$随机字符，不需要兼容和回退，实现最新代码）
    + 如果c://ProgramData/deepright/chrome_$随机字符不存在则从c://ProgramData/deepright/chrome_def复制
        + 如果c://ProgramData/deepright/chrome_def不存在或任意文件复制失败，则跳过并记录日志
        + 如果c://ProgramData/deepright/chrome_$随机字符目录已经存在则跳过不复制
    + 其他逻辑不变

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
