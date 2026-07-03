### 第一性原则
+ 仅可以新增/更新/删除browser（ ../../..）目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../../../DESIGN.md
+ 本模块设计文档：../../../../DESIGN.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 需求介绍
+ 修改命令start, 如果系统为Windows WSL/WSL2去掉启动一个指定端口和目录的CDP进程的逻辑
    + WSL系统中Chrome应用程序路径保持通过`meta-get --key browser`的chrome参数获得
        + 如果meta-get没有指定chrome参数则使用默认路径"/mnt/c/Program Files/Google/Chrome/Application/chrome.exe"
    ``` integration connect meta-get --key browser
    {
      "key": "browser",
      "name": "浏览器",
      "meta": {
        "chrome": 绝对路径
      },
      其他属性
    }
    ```
+ 修改命令init，如果系统为Windows WSL/WSL2则：
    + 依旧需要有效的老逻辑：
        + 先get现有实例，如果实例存在，就先走一次shutdown优雅关闭旧实例，如果不存在则继续
        + 使用有头模式，端口逻辑同原计算逻辑（与MAC版本逻辑相同），Chrome应用程序路径保持获取同命令start
        + 等端口和CDP真正可用后，把实例写进状态文件，之后命令继续阻塞，直到Chrome进程退出或被关闭，再把状态删掉
    + 使用../20260617-1/browser_instance_wsl.go来启动CDP，并解析响应报文
        + 脚本手册在../20260617-1/REQUIREMENT.md
        + 如果脚本无JSON响应则默认为失败
        + 使用有头模式
        + 响应报文需要与原逻辑一致
+ 修改命令create，如果系统为Windows WSL/WSL2则：
    + 去掉复制系统Chrome的user-data-dir的逻辑
    + 使用../20260617-1/browser_instance_wsl.go来启动或复用CDP，并解析响应报文
        + 脚本手册在../20260617-1/REQUIREMENT.md
        + 如果脚本无JSON响应则默认为失败
        + 是否使用无头还是有头与MAC版本判断逻辑相同，读取meta-get的headless
        ``` integration connect meta-get --key browser
            {
              "key": "browser",
              "name": "浏览器",
              "meta": {
                "headless": false/true/TRUE/FALSE
              },
              其他属性
            }```
        + 响应报文需要与原逻辑一致
+ 修改命令stop，如果系统为Windows WSL/WSL2则不要删除chrom_*
+ 修改命令shutdown，如果系统为Windows WSL/WSL2则不要删除chrom_*
+ 编译都需要遵守browser的二进制原则

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
