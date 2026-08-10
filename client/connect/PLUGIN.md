### Plugin规范
#### 生命周期
+ 启动插件 -> 监听自己感兴趣的事件 -> 将事件记录在三方请求 -> 推送事件为一次性备忘录任务 -> 开始备忘录任务前通知 -> 结束备忘录任务后通知 -> 持续监听事件 -> 关闭插件

#### 插件信息
+ 必须实现name和param命令来提供插件信息
    + name命令：获取插件的唯一键和展示名称，格式为json
    ``` 假设插件可执行程序为a
    ./a name
    ```
    ``` 响应报文
    {"key":"a","name":"App A"}
    ```
    + param命令：获取插件运行时需要使用的参数列表，格式为json string array
    ``` 假设插件可执行程序为a
    ./a param
    ```
    ``` 响应报文，生命周期内需要使用的配置Key-Value集合都需要在此返回
    [{"appId":""},{"appSecret":""}]
    ```
+ 必须实现scope命令来提供插件配置范围
    ``` 假设插件可执行程序为a
    ./a scope
    ```
    ``` 响应报文
    ["reuse","agent","provider","thinking"]
    ```
    + 不同于param命令返回插件自定义配置，scope命令返回插件支持容器配置
        + reuse: 是否允许配置"复用当前会话"
        + agent: 是否允许配置"Agent"
        + provider: 是否允许配置"模型"
        + thinking: 是否允许配置思考模式
        + swarm：是否开启蜂群模式
    + 如果插件返回["reuse"]，则表示仅支持"复用当前会话"配置
    + 如果插件返回[]空列表，则表示完全不支持容器配置
        + 如果插件不支持容器通用配置项，必须稳定返回[]，不得触发healthz、/command、自动拉起daemon等运行时副作用
    + scope 必须稳定返回合法 JSON string array；容器不会再对缺失命令、执行失败、错误文本或非法输出做任何回退
+ 可选实现schema命令来提供LLM Response Json Schema
    ``` 假设插件可执行程序为a
    ./a schema
    ```
    ``` 响应报文（Json Schema）
    {
      "type": "object",
      "properties": {
        "id": {
          "description": "用户唯一标识符",
          "type": "integer"
        },
        "name": {
          "description": "用户姓名",
          "type": "string"
        },
        "email": {
          "description": "电子邮箱地址",
          "type": "string",
          "format": "email"
        }
      },
      "required": ["id", "name"]
    }
    ```
+ 所有插件内部识别、回调映射和通知发送必须统一使用 key，name 仅用于界面和日志展示
+ 可执行程序名称需要与 name 命令响应报文里的 key 完全一致

#### 目录结构
+ 插件目录plugins位于主应用程序integration平级目录, 插件可执行程序需要位于plugins
+ 案例：插件a
```
 - integration
 - plugins
    - a
    - a.log
```

#### 注册配置
+ 插件在生命周期通过由Integration代理Connect模块的meta-create/meta-update命令创建或更新插件配置
    + 创建指定插件的配置
    ``` 假设插件key为a
    ./integration connect meta-create --key a --meta {} --callback ...
    ```
    ``` 响应成功，或失败抛出错误码和异常
    OK
    ```
    + 更新指定插件的配置
    ``` 假设插件key为a
    ./integration connect meta-update --key a --meta {} --callback ...
    ```
    ``` 响应成功，或失败抛出错误码和异常
    OK
    ```
+ 容器会根据插件 key 解析出真实插件二进制并写入 callback；插件无需依赖展示名、旧别名或额外特判

#### 获取配置
+ 插件在生命周期通过由Integration代理Connect模块的meta-get命令获取自身配置
``` 假设插件key为a, 响应报文为数组，元素key=a的meta即为插件的运行时配置, 所有value均为string
../integration connect meta-get --key a
```
``` 返回配置
{
  "key": "a",
  "meta": {"appId":"appId Value","appSecret":"appSecret Value","stream":true...}
}
```
+ 获取配置的应用名称、相对路径和命令是固定, 插件本身无法也无需修改, 遵从规范即可

#### 启动与关闭插件
+ 必须实现start和stop命令来启动或关闭插件，start和stop命令仅由插件容器使用，不要出现在--help中
``` 假设插件可执行程序为a, 启动后台进程, 需要在应用启动目录下记录进程.pid文件
./a start --connect-bin 主程序路径
```
``` 关闭后台进程, 删除在应用启动目录下记录进程.pid文件
./a stop
```
+ 插件可以是常驻HTTP服务, 也可以是无状态的命令行工具. 如果是常驻HTTP服务, 则通过start启动并通过stop关闭并释放资源
    + 常驻HTTP服务型插件启动后, 执行其他插件命令可以通过curl来与本地服务交互
    ``` 假设插件可执行程序为a, 需要执行init命令
    ./a init ...参数
    ```
        + 内部可以采用curl与http://localhost交互, 为了防止端口冲突可以在param命令中显式声明端口

#### 记录插件事件
+ 插件在生命周期通过由Integration代理Connect模块的add-request命令记录感兴趣的事件为三方请求
    + 插件在收到事件后组装参数，并调用add-request命令，而不能直接写数据库
``` 案例
./integration connect add-request --key 三方Key --externalId 外部ID --content 请求内容 --artifacts 以,分隔的字符串附件路径 --original 原始请求 --status 状态 --created 创建时间的时间戳
```
    + key：name命令返回的key
    + externalId：对应的外部事件的溯源ID
    + content：用于执行备忘录一次性任务的实际请求
    + artifacts：转存为本地文件系统路径以,分割的附件路径（图片、文件）等, 会附带在content之后
    + original：原始请求报文，用于溯源报文
    + status：默认为待处理
    + created：默认为创建时间
+ 获取配置的应用名称、相对路径和命令是固定, 插件本身无法也无需修改, 遵从规范即可

#### 开始备忘录任务前通知
+ 如插件需要接收备忘录任务执行前通知，则必须实现 init 命令，并在 command 命令里显式声明 init
``` 假设插件可执行程序为a
./a init --message 原消息报文（json string） --content 消息文本内容 --image 以逗号分隔的图片附件 --file 以逗号分隔的文件附件
```

#### 结束备忘录任务后通知
+ 如插件需要接收备忘录任务结束后通知，则必须实现 send 命令，并在 command 命令里显式声明 send
``` 假设插件可执行程序为a
./a send --message 原消息报文（json string） --content 消息文本内容 --image 以逗号分隔的图片附件 --file 以逗号分隔的文件附件
```

#### 插件能力
+ 必须实现command来提供插件能力列表
``` 假设插件可执行程序为a
./a command
```
``` 响应报文，支持的命令都要在此返回
["name","param","init","send",...]

#### 插件使用
+ 插件容器与插件交互只允许通过CLI命令：start、stop、init、send、name、param、command等
+ 不允许直接调用插件代码

#### 插件帮助
+ 必须实现help来提供完成的插件使用手册
``` 假设插件可执行程序为a
./a help
```
+ start和stop命令仅由插件容器使用，不要出现在--help中

#### 插件日志
+ 必须在插件同目录下提供以log作为后缀名的同名日志文件
    + 假设插件可执行程序为a, 那么日志就是a.log
