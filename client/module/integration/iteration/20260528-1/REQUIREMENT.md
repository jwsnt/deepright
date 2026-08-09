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
   + 统一变量名
       + Site居中输入框、备忘录元数据、插件配置的（3个位置）的蜂群（Swarm）开关、HTML开关、思考模式开关、模型选择
       + Site模型配置的的蜂群（Swarm）开关、思考模式开关、模型选择
           + 统一变量名
               + 蜂群（Swarm）开关：对应参数名router_disable，开关打开时router_disable=false，开关关闭时router_disable=true
               + 思考模式：对应参数名thinking
               + HTML开关：对应参数名html
               + 模型选择：对应参数名model
   + 转发--host的/v1/chat/completions时统一报文
       ```
       {
           ... messages和stream保持原逻辑
           "messages": [
               {
                   "role": "user",
                   "content": 消息内容
               }
           ],
           "stream": true,
           "metadata": {
               ...
               "router_disable": 对应居中对话框的蜂群（Swarm）开关、由备忘录元数据创建的任务（含一次性任务）或插件通过add-request传递的插件蜂群（Swarm）开关。3个来源
               "thinking": 对应居中对话框的思考模式开关、由备忘录元数据创建的任务（含一次性任务）或插件通过add-request传递的插件思考模式开关。3个来源
               "html"：对应居中对话框的蜂群HTML开关
               ...,
               "agents": [
                   "router_disable":
                   "thinking":
                   "provider":
               ]
           },
           "model": 模型选择
       }
       ```
       + 其中metadata的router_disable: 对应居中对话框的蜂群（Swarm）开关、由备忘录元数据创建的任务（含一次性任务）或插件通过add-request传递的插件蜂群（Swarm）开关（注意开关打开时router_disable=false，开关关闭时router_disable=true）。3个来源
       + 其中metadata的thinking：对应居中对话框的思考模式开关、由备忘录元数据创建的任务（含一次性任务）或插件通过add-request传递的插件思考模式开关。3个来源
       + 其中metadata的html：对应居中对话框的HTML开关
       + 其中metadata的agents数据元素的router_disable、thinking、provider的来源是`设置`中蜂群开关（注意开关打开时router_disable=false，开关关闭时router_disable=true）、思考模式和模型选择（模型在agents里的参数名叫provider）
       + 其中model的来源居中对话框的模型选择、由备忘录元数据创建的任务（含一次性任务）或插件通过add-request传递的插件配置，3个来源
   + 转发--host的/cli/get时统一报文
       ```
       {
           ... messages保持原逻辑（不含stream）
           "messages": [
               {
                   "role": "user",
                   "content": 空字符串
               }
           ],
           "metadata": {
               ...,
               "agents": [
                   "router_disable":
                   "thinking":
                   "provider":
               ]
           }
       }
       ```
       + 其中metadata的agents数据元素的router_disable、thinking、provider的来源是`设置`中蜂群开关（注意开关打开时router_disable=false，开关关闭时router_disable=true）、思考模式和模型选择（模型在agents里的参数名叫provider）
       + 原始CLI/GET和CLI/PUB需求：../../../cli-get/REQUIREMENT.md
   + Site或定时任务触发proxy/integration后，需要按统一报文的格式转发至--host指定的服务
       + 只有一套协议，不需要老逻辑兼容
+ 唯一端口
    + proxy/integration所有功能均使用由--port指定的端口，默认为8080
+ 与Proxy需求对齐：../../../proxy/iteration/20260528-1/REQUIREMENT.md

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