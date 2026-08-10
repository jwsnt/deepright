### 第一性原则
+ 仅可以新增/更新/删除email（../..）同目录及其子目录下`的文件和文件夹

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
+ 严格遵守测试必过集：../../TEST_CASE.md

### 需求介绍
+ 邮件模块需要具备独立的CLI启动（start），终止（stop）命令，执行错误需要抛出异常
+ 运行时主键固定为"email_smtp"，展示名为"邮件"，需要提供的参数为
    + email、email_pop3、、email_smtp、email_password、email_whitelist
        + email_pop3和email_smtp分别为POP3和SMTP地址
        + email_whitelist为以,分隔的白名单邮件地址, 如果没有填写或为空就表示不过滤
    + 参考飞书参数和名称需求：../../../feishu/iteration/20260504-2/REQUIREMENT.md
+ 启动和关闭参考Connect模块规则和飞书模块实现，启动后自动使用代码登录邮箱，每60秒扫描一次未读邮件，并记录日志在email.log
    + 参考Connect模块：../../../REQUIREMENT.md
    + 参考飞书模块启动和终止需求：../../../feishu/iteration/20260504-1/REQUIREMENT.md
+ 在收到邮件后需要提取邮件消息并推送至add-request：
    + 准入名单：邮件发件人必须是email_whitelist中定义的白名单用户或email_address自己，否则跳过并记录在在email.log
    + 时间过滤：当前应用（mail）启动时间前30分钟为起始时间线，例如20260506 15:00:00启动，则起始时间线为20260506 14:30:00
        + 更新：记录每次处理过的邮件的最后时间（包括非准入邮件）为最新时间线，下次仅处理该时间线后的邮件（防止一直全量读取）
    + 去重邮件：记录准入后的邮件Message-ID，已经处理过的邮件不再处理，即使再时间线之后
    + 持久配置：时间过滤和去重邮件都需要记录，保证在多次重启后依旧可以恢复
    + 报文映射：
        + 创建时间（create_time）：使用邮件消息头中的Date字段 (RFC 5322)
        + 邮件报文头、标题、内容组成JSON作为原始请求（用于回溯数据）
            + {"headers":[{}],"content":""}
        + 邮件标题、内容、下载附件拼接组成请求内容（用于执行备忘录明细）
        + 将文字编码统一解析为UTF-8，避免乱码（包括输出日志email.log前）
    + 下载：
        + 图文消息下载对应图片到应用启动目录下的email_images目录下，不存在则新建
        + 下载完图片后在artifacts属性上追加图片在本地文件系统的绝对路径
        + 对于下载资源的报文进行消息内容的归一化：
            + 图片：[image]图片绝对链接
            + 文件：[file]文件绝对路径
        + 图片和文件资源需要考虑多个协议解析方式：
            + 嵌入邮件本身的图片或附件
            + 附件邮件

### 链路整理
+ 邮件模块，通过命令行启动
+ 邮件模块，从Integration代理的Connect能力获取name=email_smtp的启动配置
+ 邮件模块，管理连接，并等待邮件
    + 邮件接收文字邮件
    + 邮件接收图片邮件
+ 邮件模块，收到消息并向Integration代理的Connect能力推送

### 同步代码
+ ../../../email/REQUIREMENT.md
+ 所以设计/编译都需要遵守email的二进制和CLI收口原则

### 编写代码
+ 以Golang编写以上代码，要求：
    + 邮件内部如果需要调用Connect语义，必须统一通过integration代理执行，并遵循 connect <subcommand> [options...] 的命令格式，子命令必须固定放在第一个位置，禁止将通用参数排在子命令之前
    + 邮件模块只负责从integration代理的Connect能力获取配置来维护长连接、获取和推送消息，不能连接db和指定agent目录
    + 用户发送的图片或文件，必须同时支持从附件解析和从邮件嵌入内容解析
    + 图片下载后，必须落到email_artifacts，并用image_key命名
    + 文件下载后，必须落到email_artifacts，并用file_key命名
    + 用标准文件名email启动时，直接执行二进制，不再先探测
    + 下载失败时只能记日志，不能因为空响应把email进程打崩
    + 编译应用名：email
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写
+ 编译后的二进制文件需要放在connect模块同目录的plugins目录

### 验证测试
+ 使用mock数据，验证流程是否生效
    + 模拟邮件模块启动，正确通过CLI从Integration代理的Connect能力获取配置，并建立独立的扫描任务
    + 模拟邮件模块收到消息，并正确通过CLI向Integration代理的Connect能力推送消息
``` 启动 integration
./integration --agent-dir ../agent/test-case --site ../site
{
  "status": "started"
}
``` 通过integration注册邮件
./integration connect meta-create \
  --name email_smtp \
  --meta '{"email":使用环境变量EMAIL,"email_address":使用环境变量EMAIL_ADDRESS,"email_password":使用环境变量EMAIL_PASSWORD,"mode":"email_smtp"}' \
  --stream true \
  --callback ./email \
  --agent a \
  --model deepseek
```
``` 启动邮件（如果已经启动则自动执行stop后再start，相当于restart；最终用户主流程应优先通过integration触发插件启动）
./integration plugins start --name email_smtp
```
``` 如需独立验证插件CLI，允许直接执行插件，但connect-bin必须指向integration
./email start --connect-bin ../integration/integration
```
``` 关闭邮件
./email stop --pid-file ./email.pid
```
``` 通过integration注销邮件
./integration meta-delete \
  --name email_smtp \
```
``` 关闭integration
停止当前integration进程
```

### 撰写手册
+ 编写USER_GUIDE.md

### 关联需求
+ 邮件插件：email/iteration/日期/REQUIREMENT.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 复制至Plugin：../../../../plugins/

