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
+ 增加Post接口/api/message_insert/add，保存指定ChatId的插入消息，mid为插入消息id，由上游传递（通常为时间戳）
    + 需要保存到数据库，数据库中增加状态（status=0,待上传、1已上传，2取消）
    + AgentId仅做记录用由哪个Agent发起
```
{
    "agentId": xxx,
    "chatId": yyy,
    "mid": mmm,
    "message": zzz,
}
```
+ 增加Post接口/api/message_insert/del，删除指定ChatId的插入消息，mid为插入消息id，由上游传递（通常为时间戳）
```
{
    "chatId": yyy,
    "mid": mmm
}
```
+ 增加Post接口/api/message_insert/status，获取指定ChatId的消息状态，mid为插入消息id（可以多条），由上游传递（通常为时间戳）
```
{
    "chatId": yyy,
    "mid": [aaa,bbb,ccc]
}
```

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写
+ 最小范围更新

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
