### 第一性原则
+ 仅可以新增/更新/删除当前需求文档（REQUIREMENT.md）同目录及其子目录下`的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../DESIGN.md
+ 本模块设计文档：../DESIGN.md

### 需求介绍
+ 二进制收口, 最终交付给用户的主程序必须是`feishu`一个二进制文件，日志必须是与feishu同目录下的feishu.log
+ 飞书模块需要具备独立的CLI启动（start），终止（stop）命令，执行错误需要抛出异常
+ 参考feishu/main.go的实现为飞书实现Connect模块接收消息，运行时主键固定为"feishu"，展示名为"飞书"
    + Connect模块：../REQUIREMENT.md
+ 每60秒检查一次心跳，如果心跳不存在或无法接收则自动断开连接重新建立
+ 每次收到消息，在调用Integration代理的Connect能力推送消息前都追加到同目录下的feishu.log
    + ws报文内容需要全部输出到日志，单日志最大10M，最多保存5个
+ 每次启动时，在feishu.log记录调用Integration代理的Connect能力时使用的启动参数
```
./feishu start
```
```
./feishu stop
```
+ 飞书文档：
    + 接收消息（事件体示例）：https://open.feishu.cn/document/server-docs/im-v1/message/events/receive
    + 获取图片: https://open.feishu.cn/document/server-docs/im-v1/image/get?appId=cli_a93407f1de789cb1
    + 获取文件: https://open.feishu.cn/document/server-docs/im-v1/file/get
+ 提取飞书消息
``` 案例：文本
{"schema":"2.0","header":{"event_id":"acd2004c65232fabdbb537e99f44b12d","token":"","create_time":"1777890990994","event_type":"im.message.receive_v1","tenant_key":"1786b076276ed740","app_id":"cli_a93407f1de789cb1"},"event":{"message":{"chat_id":"oc_016646866ff0a26222ed03834a154eca","chat_type":"p2p","content":"{\"text\":\"你好\"}","create_time":"1777890990639","message_id":"om_x100b50b3242598a0c39cb0feb59d799","message_type":"text","update_time":"1777890990639"},"sender":{"sender_id":{"open_id":"ou_00f5da27adc5fca2337493c54d2bc44d","union_id":"on_52bdae542d0b4b0bb84a2bd9ee973414","user_id":null},"sender_type":"user","tenant_key":"1786b076276ed740"}}
```
``` 案例：图片（需要使用/im-v1/image/get方法下载）
{"schema":"2.0","header":{"event_id":"99e381a25382adec8e605ed3ac543b0e","token":"","create_time":"1777899530913","event_type":"im.message.receive_v1","tenant_key":"1786b076276ed740","app_id":"cli_a93407f1de789cb1"},"event":{"message":{"chat_id":"oc_016646866ff0a26222ed03834a154eca","chat_type":"p2p","content":"{\"image_key\":\"img_v3_0211c_2b3c4ef3-8169-4620-a7e4-bb478e20175g\"}","create_time":"1777899530590","message_id":"om_x100b50bdce6ba0d4c14a8073247757f","message_type":"image","update_time":"1777899530590"},"sender":{"sender_id":{"open_id":"ou_00f5da27adc5fca2337493c54d2bc44d","union_id":"on_52bdae542d0b4b0bb84a2bd9ee973414","user_id":null},"sender_type":"user","tenant_key":"1786b076276ed740"}}
```
+ 报文映射：
    + 创建时间（create_time）+ 报文内容（content）的MD5构成唯一键
    + 整个JSON作为原始报文
        + 存储的JSON SCHEMA（严格遵守）：REQUEST_SCHEMA.json
+ 下载：
    + 图文消息下载对应图片到应用启动目录下的feishu_images目录下，不存在则新建
    + 下载完图片后在artifacts属性上追加图片在本地文件系统的绝对路径
    + 对于下载资源的报文进行消息内容的归一化：
        + 图片：[image]图片绝对链接
        + 文件：[file]文件绝对路径

### 链路整理
+ 飞书模块，通过命令行启动
+ 飞书模块，从Integration代理的Connect能力获取name=feishu的启动配置
+ 飞书模块，管理连接，并等待消息（存储的JSON SCHEMA（严格遵守）：REQUEST_SCHEMA.json）
    + 飞书接收文字消息
    + 飞书接收图片消息
+ 飞书模块，收到消息并向Integration代理的Connect能力推送

### CLI命令
###### 启动与停止
```
./feishu start
```
```
./feishu stop
```
+ 启动时从Integration代理的Connect能力获取name=feishu的配置
+ 停止通过PID文件管理进程

###### param命令
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 返回启动飞书时需要提供的参数SCHEMA，即使未启动长连接也返回固定值
```
./feishu param
```
``` 固定返回
["appId","appSecret"]
```

###### name命令
> 新增自 iteration/20260504-2/REQUIREMENT.md
+ 返回飞书插件的系统主键和展示名，即使未启动长连接也返回固定值
```
./feishu name
```
``` 固定返回
{"key":"feishu","name":"飞书"}
```
+ 插件标识统一原则：展示名（name）可以是中文，系统主键（key）必须稳定唯一，所有运行时链路只能用主键，不能混用展示名

###### send命令
> 新增自 iteration/20260505-1/REQUIREMENT.md
+ 向飞书推送消息（回复已有消息）
```
./feishu send --message 原消息报文（json string） --content 消息文本内容 --image 以逗号分隔的图片附件 --file 以逗号分隔的文件附件
```
    + message为必填，为add-request的原始报文（存储的JSON SCHEMA（严格遵守）：REQUEST_SCHEMA.json）
    + image、file可为空
    + 文本、图片、文件至少要提供一种，否则报错
+ 启动参数（如密钥）获取方式同./feishu start
+ 每次推送时在feishu.log记录调用
+ 文本消息以飞书interactive卡片消息发送，正文使用Markdown渲染
+ 图片消息为先上传图片，再发送图片消息
+ 文件消息为先上传文件，再发送文件消息
+ 如果同时带文本和附件，拆成多次飞书API调用，发送顺序为图片→文件→文本
+ 飞书消息的原messageId从参数message的原始报文中获取

### 参考文档
+ 发送消息：https://open.feishu.cn/document/server-docs/im-v1/message/create?appId=cli_a93407f1de789cb1
    + 文本内容使用富文本（Markdown / post）：https://open.feishu.cn/document/feishu-cards/card-components/content-components/rich-text
    + 图片消息需要先上传图片，再发送图片消息；如果是图文复合消息则上传图片后发送图片消息，再发送文本内容
        + 上传图片：https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/im-v1/image/create
    + 文件消息需要先上传文件，再发送文件消息；如果是附件复合消息则上传文件后发送文件消息，再发送文本内容

### 同步代码
+ 所以设计/编译都需要遵守feishu的二进制和CLI收口原则

### init命令
> 新增自 ./iteration/20260507-1/REQUIREMENT.md
+ 新增CLI命令 `init`，向飞书推送任务初始化消息，参数与send相同（代理方法模式）
+ 日志需要提示调用了init
```
./feishu init --message 原消息报文（json string） --content 消息文本内容 --image 以逗号分隔的图片附件 --file 以逗号分隔的文件附件
```

### 数据存储改用add-request
> 新增自 ./iteration/20260507-2/REQUIREMENT.md
+ 飞书保存三方请求数据改为使用add-request命令
+ --key需要与name命令返回的key相同（feishu）
+ 保持自身模块独立，不自行操作数据库或调用非模块内代码

### command命令
> 新增自 ./iteration/20260507-3/REQUIREMENT.md
+ 新增CLI命令 `command`，返回飞书插件的功能列表
```
./feishu command
```


### 消息批处理
> 新增自 ./iteration/20260521-1/REQUIREMENT.md
+ 飞书插件每30秒扫描一次10分钟内的待处理（还未推送add-request命令的）消息，如果仅存在图片或附件消息则等待，直到出现文本消息后推送
+ 案例1：当前为一条文本消息 则 立即推送
+ 案例2：当前为一条文本消息 加 另一条文本消息 则 立即推送
+ 案例3：当前为一条文本消息 加 一条图片或文件消息 则下载资源并进行消息内容的归一化，将归一化后的内容附带在文本消息后推送
    + 图片：[image]图片绝对链接
    + 文件：[file]文件绝对路径
    + 最终消息为：文本消息 [image]图片绝对链接
+ 案例4：当前为一条图片或文件消息 加 一条文本消息 则处理方式同上
+ 案例5：当前为一条图片或文件消息 则 等待下个扫描周期，直到出现文本消息
+ 案例6：当前为一条图片或文件消息 加 另一条图片或文件消息 则 等待下个扫描周期，直到出现文本消息
+ 案例7：当前为一条图片或文件消息 加 另一条图片或文件消息 加 一条文本消息 则 附带在文本消息后推送
    + 最终消息为：[image]图片1绝对链接 [image]图片2绝对链接 文本消息
+ 如果消息产生超过10分钟仍然无法推送（系统故障或仅存在图片或文件消息）则标记过期不推送

### 发送日志
> 新增自 ./iteration/20260522-1/REQUIREMENT.md
+ 飞书插件发送消息时需要记录详细内容，包括请求报文、解析结果、发送结果、失败原因
+ 发送消息包括 init 命令和 send 命令

### 插件参数标准化
> 新增自 ./iteration/20260610-1/REQUIREMENT.md
+ 修改param命令，固定返回：
```
[{"appId":"飞书开放平台（https://open.feishu.cn/app）中应用凭证的App ID ","appSecret":"App Secret"}]
```

### 会话复用
> 新增自 ./iteration/20260612-1/REQUIREMENT.md
+ 在插件页面点击复用当前会话，启动email时使用当前ChatID作为后续备忘录任务明细的ChatID
+ 如果未点击复用，则会话（ChatID）固定为feishu

### JSON提取兼容
> 新增自 ./iteration/20260612-2/REQUIREMENT.md
+ 如果响应报文无法解析成JSON，尝试使用正则从文本中提取JSON并应用到schema中
+ 典型案例：前缀干扰文本+JSON，需提取纯净JSON部分
+ 如果依旧提取失败则使用原逻辑，全部发送

### 编写代码
+ 以Golang编写以上代码，要求：
    + 飞书内部如果需要调用Connect语义，必须统一通过integration代理执行，并遵循 connect <subcommand> [options...] 的命令格式，子命令必须固定放在第一个位置，禁止将通用参数排在子命令之前
    + 飞书模块只负责从integration代理的Connect能力获取配置来维护长连接、获取和推送消息，不能连接db和指定agent目录
    + 用户发送的图片，必须走message_resource.get，不能误用普通image.get
    + 图片下载后，必须落到feishu_artifacts，并用image_key命名
    + 文件下载后，必须落到feishu_artifacts，并用file_key命名
    + 用标准文件名feishu启动时，直接执行二进制，不再先探测
    + 下载失败时只能记日志，不能因为空响应把feishu进程打崩
    + 编译应用名：feishu
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写
+ 编译后的二进制文件需要放在connect模块同目录的plugins目录

### 验证测试
+ 测试必过集：TEST_CASE.md
+ 使用mock数据，验证流程是否生效
    + 模拟飞书模块启动，正确通过CLI从Integration代理的Connect能力获取配置，并建立长连接
    + 模拟飞书模块收到消息，并正确通过CLI向Integration代理的Connect能力推送消息
``` 启动 integration
./integration --agent-dir ../agent/test-case --site ../site
{
  "status": "started"
}
```
``` 通过integration注册飞书
./integration connect meta-create \
  --name feishu \
  --meta '{"appId":使用环境变量FEISHU_APP_ID,"appSecret":使用环境变量FEISHU_APP_SECRET,"mode":"feishu"}' \
  --stream true \
  --callback ./feishu \
  --agent a \
  --model deepseek
```
``` 启动飞书（如果已经启动则自动执行stop后再start，相当于restart；最终用户主流程应优先通过integration触发插件启动）
./integration plugins start --name feishu
```
``` 如需独立验证插件CLI，允许直接执行插件，但connect-bin必须指向integration
./feishu start --connect-bin ../integration/integration
```
``` 关闭飞书
./feishu stop --pid-file ./feishu.pid
```
``` 通过integration注销飞书
./integration meta-delete \
  --name feishu \
```
``` 关闭integration
停止当前integration进程
```
+ ./feishu param，固定返回["appId","appSecret"]
+ ./feishu name，固定返回{"key":"feishu","name":"飞书"}
> 新增自 iteration/20260518-1/REQUIREMENT.md
+ 新增feishu schema命令，返回Response Json Schema
+ 为feishu的command命令添加schema命令
> 新增自 iteration/20260518-2/REQUIREMENT.md
+ feishu调用add-request时通过自身schema命令获取response_schema，通过--schema传递
> 新增自 iteration/20260518-3/REQUIREMENT.md
+ send/init命令处理发送消息时检查--content是否为符合schema的JSON
+ 符合schema则归一化：content转Markdown（图片下载/上传飞书image_key）、artifacts作为附件发送
+ 不符合或异常则降级为整体发送
+ init/send共用schema归一化与Markdown图片替换逻辑
+ 发送飞书超时180秒
> 新增自 iteration/20260518-1/REQUIREMENT.md
+ 新增feishu schema命令，返回Response Json Schema
+ 为feishu的command命令添加schema命令
> 新增自 iteration/20260518-2/REQUIREMENT.md
+ feishu调用add-request时通过自身schema命令获取response_schema，通过--schema传递
> 新增自 iteration/20260518-3/REQUIREMENT.md
+ send/init命令处理发送消息时检查--content是否为符合schema的JSON
+ 符合schema则归一化处理：content转Markdown（含图片下载/上传到飞书image_key）、artifacts作为附件发送
+ 不符合或异常则降级，将整个--content作为整体发送
+ init命令与send命令共用相同的schema归一化与Markdown图片替换逻辑
+ 发送飞书超时180秒
> 新增自 iteration/20260518-1/REQUIREMENT.md
+ 新增feishu schema命令，返回Response Json Schema
+ 为feishu的command命令添加schema命令
> 新增自 iteration/20260518-2/REQUIREMENT.md
+ feishu调用add-request时通过自身schema命令获取response_schema，通过--schema传递
> 新增自 iteration/20260518-3/REQUIREMENT.md
+ send/init命令处理发送消息时检查--content是否为符合schema的JSON
+ 符合schema则归一化处理：content转Markdown（含图片下载/上传到飞书image_key）、artifacts作为附件发送
+ 不符合或异常则降级，将整个--content作为整体发送
+ init命令与send命令共用相同的schema归一化与Markdown图片替换逻辑
+ 发送飞书超时180秒
> 新增自 iteration/20260518-1/REQUIREMENT.md
+ 新增feishu schema命令，返回Response Json Schema
+ 为feishu的command命令添加schema命令
> 新增自 iteration/20260518-2/REQUIREMENT.md
+ feishu调用add-request时通过自身schema命令获取response_schema，通过--schema传递
> 新增自 iteration/20260518-3/REQUIREMENT.md
+ send/init命令处理发送消息时检查--content是否为符合schema的JSON
+ 符合schema则归一化处理：content转Markdown（含图片下载/上传到飞书image_key）、artifacts作为附件发送
+ 不符合或异常则降级，将整个--content作为整体发送
+ init命令与send命令共用相同的schema归一化与Markdown图片替换逻辑
+ 发送飞书超时180秒

### 撰写手册
+ 编写USER_GUIDE.md

### 关联需求
+ 飞书插件：feishu/iteration/日期/REQUIREMENT.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 相关迭代：iteration/日期/REQUIREMENT.md
+ 同步代码：../../integration/REQUIREMENT.md（每次都要同步更新代码）
+ 复制至Plugin：../../../../plugins/
> 合并截止：./iteration/20260612-2/REQUIREMENT.md，下次合并从此之后的新迭代开始
