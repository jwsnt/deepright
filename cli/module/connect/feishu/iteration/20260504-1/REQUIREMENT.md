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

### 同步代码
+ ../../../feishu/REQUIREMENT.md
+ 所以设计/编译都需要遵守feishu的二进制和CLI收口原则

### 需求介绍
+ 飞书模块需要具备独立的CLI启动（start），终止（stop）命令，执行错误需要抛出异常
+ 参考feishu/main.go的实现为飞书实现Connect模块接收消息，运行时主键固定为"feishu"，展示名为"飞书"
    + Connect模块：../../../REQUIREMENT.md
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
+ 下载：
    + 图文消息下载对应图片到应用启动目录下的feishu_images目录下，不存在则新建
    + 下载完图片后在artifacts属性上追加图片在本地文件系统的绝对路径
    + 对于下载资源的报文进行消息内容的归一化：
        + 图片：[image]图片绝对链接
        + 文件：[file]文件绝对路径

### 链路整理
+ 飞书模块，通过命令行启动
+ 飞书模块，从Integration代理的Connect能力获取name=feishu的启动配置
+ 飞书模块，管理连接，并等待消息
    + 飞书接收文字消息
    + 飞书接收图片消息
+ 飞书模块，收到消息并向Integration代理的Connect能力推送

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
+ 使用mock数据，验证流程是否生效
    + 模拟飞书模块启动，正确通过CLI从Integration代理的Connect能力获取配置，并建立长连接
    + 模拟飞书模块收到消息，并正确通过CLI向Integration代理的Connect能力推送消息
``` 启动 integration
./integration --agent-dir ../agent/test-case --site ../site
{
  "status": "started"
}
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

### 撰写手册
+ 编写USER_GUIDE.md

### 关联需求
+ 飞书插件：feishu/iteration/日期/REQUIREMENT.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 复制至Plugin：../../../../plugins/


