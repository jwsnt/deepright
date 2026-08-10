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
+ 转发下游（--host指定）服务的HTTP超时调整如下
    + --http_connect_timeout：默认15000（15秒）
    + --http_socket_timeout：默认45000（45秒）
    + --http_timeout：默认45000（45秒）
    + 所有HTTP请求强制使用HTTP/1.1
+ 先从主应用的config.json中的http读取以上配置，如果不存在则使用默认值：
```
{
    ...
    "http": {
        "http_connect_timeout":
        "http_socket_timeout":
        "http_timeout":
    }
}
```
+ 如果主应用的config.json中http.debug=true，则记录cli/get和cli/pub的详细日志，默认为false
```
{
    ...
    "http": {
        ...
        "debug": true
    }
}
```
    + 明细日志包括：
        + cli/get的发起时间
        + cli/get的响应：
            + 如果超时，记录超时的时间和原因
            + 如果有待执行renew，记录原始报文和时间
        + cli/pub：
            + 开始执行的时间
            + 完成执行的状态、结果、时间

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
