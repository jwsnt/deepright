### 第一性原则
+ 仅可以新增/更新/删除connect（../..）同目录的文件和文件夹
+ 如非授权，禁止修改其他插件目录文件和文件夹
### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../../DESIGN.md
    + browser、email、feishu、remote等

### 迭代要求
+ Remote介绍：../../REQUIREMENT.md
+ Remote手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 关联需求
+ SSH插件：../日期/REQUIREMENT.md

### 同步代码
+ ../../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 重要提示
+ 设计前需要仔细阅读Remote的设计，需要兼容方案
    + Remote介绍：../../REQUIREMENT.md

### 需求介绍
+ 修改命令create的缓存策略, 从agentId + chantId改为agentId + chantId + remote
    + 原本create先按agentId + chatId查现有会话，命中就直接返回，这意味着同一个agentId/chatId下即使传了不同远程主机，也会复用已有缓存，需要修改这个逻辑
    + create原始需求： ../../REQUIREMENT.md
``` 假定应用程序为remote，需要缓存的维度为-agentId xxx --chatId yyy --remote ubuntu@1.2.3.4
./remote create --agentId xxx --chatId yyy --remote ubuntu@1.2.3.4 --password xxx --port 10086
```
+ 修改命令shutdown的缓存策略, 从agentId + chantId改为agentId + chantId + remote（可选）
    + shutdown原始需求： ../../REQUIREMENT.md
``` 假定应用程序为remote，关闭ubuntu@1.2.3.4
./remote shutdown --agentId xxx --chatId yyy --remote ubuntu@1.2.3.4
```
``` 假定应用程序为remote，关闭所有连接（不带remote）
./remote shutdown --agentId xxx --chatId yyy
```
+ 修改命令get的缓存策略, 从agentId + chantId改为agentId + chantId + remote
    + 保持归一化--agentId和--chatId参数为小写字母
    + get原始需求： ../../REQUIREMENT.md
``` 假定应用程序为remote
./remote get --agentId xxx --chatId yyy --remote ubuntu@1.2.3.4
```
``` 返回响应
{"agentId":"xxx","chatId":"yyy","port":10086,"pid":9999,"ssh":"ubuntu@1.2.3.4"}
```
+ 修改exec命令的缓存策略, 从agentId + chantId改为agentId + chantId + remote
``` 假定编译后唯一应用程序为remote
./remote exec --session xxx@yyy --remote ubuntu@1.2.3.4 "ls -l /a"
```
+ 修改scp命令的缓存策略，自动提取命令中的--remote并使用agentId + chantId + remote维度获取连接
    + scp原始需求：../20260521-1/REQUIREMENT.md
``` 假设应用名称为remote，从本地拷贝到远程服务器
remote scp /local/path/file.txt ubuntu@43.155.234.33:/remote/path/ --session #agentId@#chat
```
    + --remote：ubuntu@43.155.234.33
``` 假设应用名称为remote，从远程服务器下载到本地
remote scp ubuntu@43.155.234.33:/remote/path/file.txt . --session #agentId@#chat
```
     + --remote：ubuntu@43.155.234.33
+ 本次需求仅调整缓存策略，最小改动

### 编写代码
+ 以Golang编写以上代码，要求：
    + 所有Remote模块复用启动时初始化的全局数据库连接，禁止每次请求单独打开和关闭数据库文件
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



