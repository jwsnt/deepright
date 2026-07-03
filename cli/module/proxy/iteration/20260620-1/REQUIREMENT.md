### 第一性原则
+ 仅可以新增/更新/删除proxy（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Proxy介绍：../../REQUIREMENT.md
+ Proxy手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 需求介绍
+ 修改/api/config和/api/edit接口，增加media属性，是一个Agent维度的JSON Object
```
{
    ... Agent相关config配置中其他属性
    "media": {
        ... 多组属性
    }
}
```
+ 如果media不会空，转发/v1/chat/completions时在agent数组中对应Agent属性需要带上media = {...}
```
{
    "agents": [
        {...},
        {
            配置了media的Agent配置
            "media": {
                多组属性
            }
        }
    ]
}
```
+ 如果media不会空，请求/cli/get时在agent数组中对应Agent属性需要带上media = {...}，属性位置同转发/v1/chat/completions
+ 每次都读取最新，不要缓存

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



