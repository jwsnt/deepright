---
name: __internal_browser
description: 通过脚本或录制，模拟人工在网页上的点击、输入及导航操作，实现高效的浏览器数据爬取、功能测试或流程自动化
---

### 可执行文件
#dir/plugins/browser
+ 当前agentId: #agentId
+ 当前chatId: #chat

### 执行顺序
+ 查看帮助
```
browser --help
```
+ 注册CDP服务
```
browser instance create --agentId=#agentId --chatId=#chat
```
+ 操作浏览器, 每次都要填写session（由agentId@chatId组合）
```
browser --session #agentId@#chat goto https://www.ctrip.com
browser --session #agentId@#chat eval 'document.body ? document.body.innerText.slice(0, 1000) : ""'
```

### 等待加载
+ 读取内容时需要等待至少5秒，尤其是SPA页面要提供足够加载时间

### 超时处理
+ 超时时前等待重试，而不是立即切换命令或页面
+ 可以通过timeout或browser-timeout调整
```
#dir/plugins/browser
    --session "#agentId@#chat" \
    --navigation-timeout 120000 \
    --browser-timeout 30s \
    --timeout 15000 \
    eval 'new Promise(function(resolve){var check=function(){var app=document.querySelector("#app");if(app&&app.children.length>0){resolve({found:true,children:app.children.length,html:app.innerHTML.slice(0,1500)})}else{setTimeout(check,500)}};setTimeout(function(){resolve({found:false})},10000);check()})'
```
+ 区别：
    + --navigation-timeout 120000 导航超时，单位是毫秒
    + --browser-timeout 30s, 作用于整条CLI命令的总超时
    + --timeout 15000, 作用于单次动作，单位是毫秒

### 执行代码
+ 优先browser命令包装的Playwright
+ 如果browser命令多次尝试无法完成任务，使用原生Playwright，并自行安装驱动

### 任务完成
+ 需要使用shutdown销毁CDP实例，释放资源
```
browser instance shutdown --agentId=#agentId --chatId=#chat
```