---
name: __internal_remote
description: 通过复用的网络通道连接指定SSH远程主机，执行所需命令，并完成文件上传或下载操作
---

### 可执行文件
#dir/plugins/remote
+ 当前agentId: #agentId
+ 当前chatId: #chat

### 执行顺序
+ 查看帮助
```
remote --help
```
+ 连接SSH服务
```
./remote create --agentId=#agentId --chatId=#chat --remote ubuntu@1.2.3.4 --password xxx --port 10086
```
+ 执行远程系统命令，每次都要填写remote和session（由agentId@chatId组合）
```
./remote exec --remote ubuntu@1.2.3.4 --session #agentId@#chat "ls -l /a"
```
+ 从本地拷贝到远程服务器
```
./remote scp /local/path/file.txt ubuntu@1.2.3.4:/remote/path/ --session #agentId@#chat
```
+ 从远程服务器下载到本地
```
./remote scp ubuntu@1.2.3.4:/remote/path/file.txt . --session #agentId@#chat
```

### 超时处理
+ 可以通过timeout调整
```
#dir/plugins/remote exec
    --remote ubuntu@1.2.3.4
    --session "#agentId@#chat" \
    --timeout 15000 \
    "ls /"
```

### 执行代码
+ 优先使用remote管理连接的会话（session）
+ 仅在remote多次尝试失败后才回退到原生ssh