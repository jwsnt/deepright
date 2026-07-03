### 第一性原则
+ 仅可以新增/更新/删除browser（../..）同目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
    + MAC设计文档：MAC_CHROME.md
    + WSL设计文档：WSL_CHROME.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 需求介绍
+ 为每个Agent和Chat（会话ID）启动CDP服务，并周期检查是否需要关闭

### 新建实例
+ 新增命令create, 启动obscura的CDP服务
``` 假定应用程序为browser_instance
./browser_instance create --agentId xxx --chatId yyy
```
    + 调用obscura的serve命令，使用--stealth启动CDP服务（需要区分当前系统环境是MAC还是Linux）
    + 必须启动--stealth（命令create的验证，启动后navigator.webdriver需要为undefined）
    ```
    obscura serve --port 端口 --stealth &
    ```
        + 可执行程序obscura需要复制到当前应用程序下的obscura目录(./obscura/obscura), 使用时需要使用相对路径
            + 编译需求: ../obscura/REQUIREMENT.md
        + 端口为Agent + Chat的Hash值, 如果不在端口可用范围内则进行调整
            + 首先归一化--agentId和--chatId参数为小写字母，防止因为大小写引起的匹配错误
            + 先检查计算后的端口，如果端口存在检查是不是CDP进程，如果不是Kill重新创建，是就直接返回不要重复创建
            + 算法需要幂等，端口需要在20000以后，并保证相同Agent和Chat每次生成端口一致
        + GIT地址：https://github.com/h4ckf0r0day/obscura
        + 使用后台线程启动
``` 返回响应
{"agentId":"xxx","chatId":"yyy","port":1024,"pid":9999,"cdp":"ws://127.0.0.1:1024/devtools/browser"}
```
+ 新建完实例需要创建或更新应用程序同目录下的browser_instance.json, 该文件负责记录当前创建所有实例的端口号、pid和CDP地址
``` browser_instance.json
[{"agentId":"xxx","chatId":"yyy","port":1024,"pid":9999,"cdp":"ws://127.0.0.1:1024/devtools/browser"},...]
```
+ 如果Agent和Chat（会话ID）已经存在CDP服务且pid依旧存在则不用重复创建, 否则创建并更新browser_instance.json

### 关闭实例
+ 新增命令shutdown, 关闭obscura的CDP服务
``` 假定应用程序为browser_instance
./browser_instance shutdown --agentId xxx --chatId yyy
```
    + 从browser_instance.json查找对应pid并关闭
``` 返回响应
OK或失败异常
```

### 实例列表
+ 新增命令list, 获取已启动obscura的CDP服务列表
``` 假定应用程序为browser_instance
./browser_instance list
```
``` 返回响应
[{"agentId":"xxx","chatId":"yyy","port":1024,"pid":9999,"cdp":"ws://127.0.0.1:1024/devtools/browser"},...]
```

### 指定实例
+ 新增命令get, 获取指定obscura的CDP服务信息
    + 首先归一化--agentId和--chatId参数为小写字母，防止因为大小写引起的匹配错误
``` 假定应用程序为browser_instance
./browser_instance get --agentId xxx --chatId yyy
```
``` 返回响应
{"agentId":"xxx","chatId":"yyy","port":1024,"pid":9999,"cdp":"ws://127.0.0.1:1024/devtools/browser"}
```

#### 插件帮助
+ 必须实现help来提供完成的插件使用手册
``` 假设插件可执行程序为a
./a help
```

### 同步代码
+ ../../browser/REQUIREMENT.md
+ 所以设计/编译都需要遵守browser的二进制和CLI收口原则

### 编写代码
+ 以Golang编写以上代码，要求：
    + 代码简洁，包体积越小越好
    + 能用开源包的就用开源包
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 相关迭代：iteration/日期/REQUIREMENT.md
+ 同步代码：../../../integration/REQUIREMENT.md（每次都要同步更新代码）
