### 第一性原则
+ 仅可以新增/更新/删除browser（ ../../..）目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../../../DESIGN.md
+ 本模块设计文档：../../../../DESIGN.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 需求介绍
+ 修改命令param，固定返回2个结果：
```
["headless","chrome"]
```

+ 修改命令start, 如果系统为Windows WSL2则在原逻辑结束前启动一个指定端口和目录的CDP进程
    + Chrome应用程序路径通过`meta-get --key browser`的chrome参数获得
        ``` integration connect meta-get --key browser
        {
          "key": "browser",
          "name": "浏览器",
          "meta": {
            "chrome": 绝对路径
          },
          其他属性
        }
    + 启动案例
    ```
    "$CHROME的绝对路径" --remote-debugging-port=29876 --remote-debugging-address=0.0.0.0 --remote-debugging-port=29876 --user-data-dir="$(wslpath -w $DIR的绝对路径)" --no-first-run &
    ```
    + 如果meta-get指定了chrome参数则使用该路径，否则使用"/mnt/c/Program Files/Google/Chrome/Application/chrome.exe"为默认路径
    + 固定端口29876，同步启动CDP进程，直到CDP进程自动终止（用户关闭、异常等）后才正常返回
    + 无法启动CDP需要阻断命令，抛出异常并记录在browser.log日志，包括完整的启动chrome.exe命令

+ 新增命令shutdown, 销毁指定的Agent和Chat的CDP服务
``` 假定应用程序为browser_instance
./browser_instance instance shutdown --agentId xxx --chatId yyy
```
    + 通过instance get --agentId xxx --chatId yyy获取CDP，并通过WebSocket发送Browser.close指令（优雅关闭）
    + 如果不指定Agent和Chat则默认为销毁29876端口的CDP（由命令start创建，如果已销毁则默认成功）
    ``` 通过WebSocket发送Browser.close指令（优雅关闭）
    curl -s http://localhost:29876/json/version
    ```
    + 无法销毁CDP需要阻断命令，抛出异常并记录在browser.log日志
    + 需要保证WSL宿主机Chrome被同时也不清理

+ 修改命令create，如果系统为Windows WSL2则在原逻辑开始前尝试复制chrome_base目录作为当前user-data-dir的拷贝
    + --remote-debugging-port：计算逻辑相同，得出一个端口号
    + --user-data-dir：固定路径为/mnt/c/ProgramData/deepright/agent/chrome_$端口号，例如agentId和chatId计算后的端口为10086，则路径为/mnt/c/ProgramData/deepright/agent/chrome_10086
        +  启动CDP前需要使用cp -a完整复制chrome_base为chrome_10086（去掉同步锁文件），复制"/mnt/c/ProgramData/deepright/chrome_base"
            + chrome_base路径同命令start，如果目录为空或不存在则不需要复制
            + 复制过程需要有详细的日志记录在browser.log
    + --headless：通过`meta-get --key browser`的headless参数获得，可能为空
        + false/FALSE/False：归一小写后如果为真则使用有头
        + true/TRUE/True：归一小写后如果为真则使用无头
        + 解析错误则默认为无头
    ``` integration connect meta-get --key browser
    {
      "key": "browser",
      "name": "浏览器",
      "meta": {
        "headless": false/true/TRUE/FALSE
      },
      其他属性
    }
    ```
    ``` 案例
    "/mnt/c/Program Files/Google/Chrome/Application/chrome.exe" \
      --headless new \
      --remote-debugging-port=$端口号 \
      --remote-debugging-address=127.0.0.1 \
      --user-data-dir="$(wslpath -w /mnt/c/ProgramData/deepright/agent/chrome_$端口号)" \
      --no-first-run &
    ```
        + WSL中remote-debugging-address需要使用127.0.0.1
        + 通过CURL检查是否启动成功
        ```
            curl -s http://localhost:29876/json/version
        ```
    + 同原逻辑，已经创建不要重复创建，创建失败的阻断命令

+ 修改命令stop，安全关闭所有已启动CDP
    + 使用instance list获取所有CDP，并逐个调用shutdown命令优雅关闭
    + 如果系统为Windows WSL2则在原逻辑结束后删除/mnt/c/ProgramData/deepright/agent下所有目录
    + 任何失败日志记录在browser.log，不要阻断关闭

+ 新增命令init，初始化指定AgentId+ChatId的CDP
    + 先使用instance get，如果CDP已启动则使用destroy注销
    + 然后使用同步进程，创建新的有头CDP
    ```
    "$CHROME的绝对路径" --remote-debugging-port=$端口号 --remote-debugging-address=0.0.0.0 --remote-debugging-port=29876 --user-data-dir="$(wslpath -w /mnt/c/ProgramData/deepright/agent/chrome_$端口号)" --no-first-run
    ```
        + $CHROME的绝对路径、$端口号、user-data-dir获取方式同create命令完全一致
            + user-data-dir如果已存在则不需要复制或重复创建，如果不存在则进行复制，复制逻辑同命令start（分Mac和WLS）
        + init命令强制使用有头模式，同步创建CDP，并等待CDP创建成功或失败后返回

### 关闭实例
+ 修改命令shutdown, 关闭chrome的CDP服务
``` 假定应用程序为browser_instance
./browser_instance shutdown --agentId xxx --chatId yyy
```
+ 从instance get --agentId xxx --chatId yyy查找对应pid并关闭
``` 返回响应
OK或失败异常
```

### 自动释放
+ 保持原逻辑不变：../20260508-1/REQUIREMENT.md
+ 释放的进程改为chrome cdp，包括MAC和Windows WSL

#### 关于WSL2
+ WSL2和宿主机一定是"mirrored mode"，Windows宿主机和WSL2可以彼此用[localhost]作为目标地址通信

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
