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
+ 设计前需要仔细阅读Connect的设计
    + Connect介绍：../../../REQUIREMENT.md
+ 严格遵守原始报文JSON SCHEMA：../../REQUEST_SCHEMA.json）
+ 严格遵守测试必过集：../../TEST_CASE.md

### 需求介绍
+ 如果响应报文无法解析成JSON，则尝试使用正则从文本中提取JSON并应用到schema中：
+ 典型案例如下（前缀文本加JSON，"先看看桌面文件情况，同时尝试截图。截图已生成。现在截图下载目录："为干扰内容）
先看看桌面文件情况，同时尝试截图。截图已生成。现在截图下载目录：{
"content": "已截取桌面和下载目录两张截图，均保存在桌面：\n\n桌面目录截图：desktop_screenshot_20260612_172306.png\n下载目录截图：downloads_screenshot_20260612_172344.png",
"artifacts": [
{"path": "/Users/Desktop/desktop_screenshot_20260612_172306.png", "desc": "桌面目录截图"},
{"path": "/Users/Desktop/downloads_screenshot_20260612_172344.png", "desc": "下载目录截图"}
],
"why_do_this": "用户飞书要求截取桌面和下载目录的截图，通过ls查看两个目录结构确认文件情况后，使用screencapture分别截取，最终两张截图均存放于桌面。"
+ 需要提取为如下格式正确的JSON
{
    "content": "已截取桌面和下载目录两张截图，均保存在桌面：\n\n桌面目录截图：desktop_screenshot_20260612_172306.png\n下载目录截图：downloads_screenshot_20260612_172344.png",
    "artifacts": [
        {"path": "/Users/Desktop/desktop_screenshot_20260612_172306.png", "desc": "桌面目录截图"},
        {"path": "/Users/Desktop/downloads_screenshot_20260612_172344.png", "desc": "下载目录截图"}
    ],
    "why_do_this": "用户飞书要求截取桌面和下载目录的截图，通过ls查看两个目录结构确认文件情况后，使用screencapture分别截取，最终两张截图均存放于桌面。"
}
+ 如果依旧提取失败则使用原逻辑，全部发送

### 同步代码
+ ../../../feishu/REQUIREMENT.md
+ 所以设计/编译都需要遵守feishu的二进制和CLI收口原则

### 编写代码
+ 以Golang编写以上代码，要求：
    + 编译应用名：feishu
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 复制至Plugin：../../../../plugins/
