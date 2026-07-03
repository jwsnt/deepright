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
+ 区分系统（MAC或是Windows/WSL）调用不同沙盒方案（包括目录选择）
+ 严格隔离MAC系统的实现路径，完全保持原样
+ WSL沙盒需求：../../../cli-get/sandbox/wsl/REQUIREMENT.md

### 需求介绍
+ 修改/api/skills?agentId=xxx：
    + 如果开启了browser插件（需要监测是否开启状态）则增加：__internal_browser
    + 如果开启了remote插件（需要监测是否开启状态）则增加：__internal_remote
    + 同时从主应用的config.json的将skills数据（string array）追加到结果
    + 原本__internal_cron的会改为从config.json读取
```
{
    ...
    "skills": [
        "__internal_cron",
        ...
    ]
}
```
+ 原需求：../20260610-3/REQUIREMENT.md
+ 不需要兼容，改为最新

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



