### 第一性原则
+ 仅可以新增/更新/删除feishu（../..）同目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../../DESIGN.md
+ 本模块设计文档：../../../DESIGN.md

### 迭代要求
+ Connect介绍：../../../REQUIREMENT.md
+ Connect手册：../../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 重要提示
+ 设计前需要仔细阅读Connect的设计，需要兼容方案
    + Connect介绍：../../../REQUIREMENT.md
+ 严格遵守原始报文JSON SCHEMA：../../REQUEST_SCHEMA.json）
+ 严格遵守测试必过集：../../TEST_CASE.md

### 需求介绍
+ 新增飞书CLI命令send，向飞书推送消息
```
./feishu send --message 原消息报文（json string） --content 消息文本内容 --image 以逗号分隔的图片附件 --file 以逗号分隔的文件附件
```
    + message为必填，为add-request的原始报文
        + add-request需求： ../../../REQUIREMENT.md
    + image、file 可为空
+ 启动参数（如密钥）等获得方式同./feishu start
    + 获取参数需求：../20260504-2/REQUIREMENT.md
+ 每次推送时，在feishu.log记录调用
+ 文本消息为发送飞书富文本
+ 图片消息为先上传图片，再发送图片消息
+ 文件消息为先上传文件，再发送文件消息

### 参考文档
+ 发送消息：https://open.feishu.cn/document/server-docs/im-v1/message/create?appId=cli_a93407f1de789cb1
    + 文本内容使用富文本（Markdown / post）：https://open.feishu.cn/document/feishu-cards/card-components/content-components/rich-text
    + 图片消息需要先上传图片，再发送图片消息；如果是图文复合消息（文+图片）则上传图片后发送图片消息，在发送文本内容
        + 上传图片：https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/im-v1/image/create
    + 文件消息需要先上传文件，再发送文件消息；如果是附件复合消息（文+文件）则上传文件后发送文件消息，在发送文本内容
        + 上传文件：https://open.feishu.cn/search?from=header&page=1&pageSize=10&q=%E4%B8%8A%E4%BC%A0%E6%96%87%E4%BB%B6&topicFilter=
+ 飞书消息的原messageId从参数message的原始报文中获取

### 同步代码
+ ../../../feishu/REQUIREMENT.md
+ 所以设计/编译都需要遵守feishu的二进制和CLI收口原则

### 编写代码
+ 以Golang编写以上代码，要求：
    + 编译应用名：feishu
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 复制至Plugin：../../../../plugins/

