### 第一性原则
+ 仅可以新增/更新/删除proxy（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Proxy介绍：../../REQUIREMENT.md
+ Proxy手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 新建HTTP GET服务，路径为/api/plugins/meta，获取插件信息，并缓存120秒
    + 调用../../../connect list-plugins，获取可用插件信息
        + Connect需求：../../../connect/iteration/20260504-1/REQUIREMENT.md
    + 调用../../../connect meta-list，获取已填插件信息
        + Connect需求：../../../connect/iteration/20260504-2/REQUIREMENT.md
    + 以插件名称name为唯一键进行合并，返回插件名称、可填参数、已填参数
    + 返回结构案例
    ``` json
    {
        "status": 0,
        "data": [
            {
                "name": "飞书",
                "param": ["appId", "appSecret"],
                "meta": {
                    "appId": "cli-app",
                    "appSecret": "cli-secret"
                }
            },
            {
                "name": "邮件",
                "param": ["token"],
                "meta": {}
            }
        ]
    }
    ```
    + 合并规则
        + list-plugins中的name和param为基准，返回所有可用插件
        + meta-list中的name和meta用于补充已填参数
        + 如果某个插件在list-plugins中存在，但在meta-list中不存在，则meta返回空对象{}
        + 如果某个插件在meta-list中存在，但在list-plugins中不存在，则不返回该项
        + 如果meta-list中的meta为空、null、或未填写，也按空对象{}返回
    + 请求和响应要求
        + 只支持HTTP GET
        + 成功时返回application/json
        + 成功时HTTP状态码为200，body中status=0
        + 调用connect相关能力失败时，HTTP状态码为500，body中status=1，并返回错误原因
        + 非GET请求返回405

### 链路整理
+ 接收/api/plugins/meta请求
+ 调用connect list-plugins获取插件名称和可填参数
+ 调用connect meta-list获取已填参数
+ 以name合并结果后输出JSON

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 验证测试
+ 使用mock数据验证以下场景
###### 正常返回
``` json
../../../connect list-plugins 返回
[
    {"name":"飞书","param":["appId","appSecret"]},
    {"name":"邮件","param":["token"]}
]
```
``` json
../../../connect meta-list 返回
[
    {"name":"飞书","meta":{"appId":"cli-app","appSecret":"cli-secret"},"stream":true}
]
```
+ /api/plugins/meta返回
``` json
{
    "status": 0,
    "data": [
        {
            "name": "飞书",
            "param": ["appId", "appSecret"],
            "meta": {
                "appId": "cli-app",
                "appSecret": "cli-secret"
            }
        },
        {
            "name": "邮件",
            "param": ["token"],
            "meta": {}
        }
    ]
}
```
+ 返回项数量与list-plugins一致
+ 飞书的meta来自meta-list
+ 邮件未配置时meta为空对象{}
###### 异常返回
+ connect list-plugins执行失败时，接口返回500
+ connect meta-list执行失败时，接口返回500
+ 非GET请求返回405

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



