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
+ integration在服务启动后的浏览器打开应自动打开的地址应为http://localhost:<port>/site/#app的页面，其中port为--port指定端口，或默认值
+ 打开浏览器的动作应在服务启动后延迟约200ms触发异步执行，不阻塞integration主服务启动流程
+ 自动打开浏览器时，应优先以“最大化窗口”方式启动，如Chrome的--start-maximized
  ``` Windows WSL Chrome
  /mnt/c/Program\ Files/Google/Chrome/Application/chrome.exe --start-maximized "http://localhost:8080/site/#app"
  ```
+ 在MacOS上，应优先按以下顺序查找并启动浏览器：Google Chrome、Google Chrome for Testing、Microsoft Edge、Brave Browser、Chromium
+ 在Linux上，应优先按以下顺序查找并启动浏览器：google-chrome、google-chrome-stable、chromium-browser、chromium、microsoft-edge、microsoft-edge-stable、brave-browser
+ 在Windows上（含WSL），应优先从常见安装目录查找并启动以下浏览器：Chrome、Edge、Chromium；若未命中，再从 PATH 中查找 chrome、msedge、chromium、brave
+ 当命中上述优先浏览器时，应直接启动该浏览器并附带最大化参数打开目标URL
+ 当未命中优先浏览器时，应回退到系统默认浏览器：MacOS使用open，Linux使用xdg-open，Windows使用 cmd /c start /max
+ 若当前操作系统不在支持范围内，应返回“unsupported OS”错误
+ 若自动打开浏览器失败，不应影响integration服务继续运行，系统只记录失败日志。
+ 自动打开成功后，应记录成功日志，包含实际打开的 URL

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 验证测试
+ /cli/get:
    + 用户同步一个蜂群Agent，模型deepseek，关闭了蜂群，关闭思考模式
        + test/cli-get-test-case1.json
    + 用户同步一个蜂群Agent，模型bigmodel，开启了蜂群，开启思考模式
        + test/cli-get-test-case2.json
+ /v1/chat/completions
    + 用户同步发送1+1，模型deepseek，关闭了蜂群，关闭了html，关闭思考模式，同时同步一个蜂群Agent，模型bigmodel，开启了蜂群，开启思考模式
        + test/chat-test-case1.json
    + 用户同步发送1+1，模型bigmodel，开启了蜂群，开启了html，开启思考模式，同时同步一个蜂群Agent，模型deepseek，开启了蜂群，开启思考模式
        + test/cli-get-test-case2.json

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写