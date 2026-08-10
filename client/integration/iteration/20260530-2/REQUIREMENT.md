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
+ 技术规范的指导下收口：
    + 修改/install_app接口，如果python3没有安装则在返回中添加一个元素"python3"
        + Proxy需求：../../../proxy/iteration/20260530-2/REQUIREMENT.md

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 验证测试
+ /cli/get:
    + 用户同步一个蜂群Agent，模型deepseek，关闭了蜂群，关闭思考模式
        + test/cli-get-test-case1.json
    + 用户同步一个蜂群Agent，模型bigmodel，开启了蜂群，开启思考模式
        + test/cli-get-test-case2.json
+ /v1/chat/completions
    + 用户同步发送1+1，模型deepseek，关闭了蜂群，关闭了html，关闭思考模式，同时同步一个蜂群Agent，模型bigmodel，开启了蜂群，开启思考模式
        + test/chat-test-case1.json
    + 用户同步发送1+1，模型bigmodel，开启了蜂群，开启了html，开启思考模式，同时同步一个蜂群Agent，模型deepseek，开启了蜂群，开启思考模式
        + test/cli-get-test-case2.json

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写