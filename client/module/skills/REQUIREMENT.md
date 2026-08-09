### 第一性原则
+ 仅可以新增/更新/删除当前需求文档（REQUIREMENT.md）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../DESIGN.md
+ 本模块设计文档：DESIGN.md

### 需求介绍
+ 遍历指定目录及其子孙目录，查找名称为`SKILL.md`文件，提取文件内容作为技能元数据
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则
> 新增自 ./iteration/20260511-1/REQUIREMENT.md
+ 每次执行时实时遍历指定目录及其子孙目录，不缓存技能元数据

### 元数据
+ 元数据位于SKILL.md文件头部，以---包裹（-数量可能多于3个），内容为Yaml格式，以下为案例：
+ 案例1：
```
---
name: __internal_system
description: 帮助用户使用系统命令完成工作（比如检索、截屏、视频录制、播放语音等）
metadata:
    os: darwin
---
其他内容
```
+ 案例2：
```
---
name: pdf
description: Use this skill whenever the user wants to do anything with PDF files. This includes reading or extracting text/tables from PDFs, combining or merging multiple PDFs into one, splitting PDFs apart, rotating pages, adding watermarks, creating new PDFs, filling PDF forms, encrypting/decrypting PDFs, extracting images, and OCR on scanned PDFs to make them searchable. If the user mentions a .pdf file or asks to produce one, use this skill.
license: Proprietary. LICENSE.txt has complete terms
---

# PDF Processing Guide
```
+ 案例3：
```
---
name: pdf-processing
description: Extract PDF text, fill forms, merge files. Use when handling PDFs.
license: Apache-2.0
metadata:
  author: example-org
  version: "1.0"
---
```
+ 提取Yaml内容，包括如如下属性：
| 字段 (Field) | 必填 (Required) | 约束 (Constraints) |
| :--- | :---: | :--- |
| **name** | 是 | 最多 64 个字符。仅限小写字母、数字和连字符（-），不能以连字符开头或结尾 |
| **description** | 是 | 最多 1024 个字符。不能为空。描述技能的功能及使用场景。 |
| **license** | 否 | 许可证名称或指向随附许可证文件的引用。|
| **compatibility** | 否 | 最多 500 个字符。说明环境要求（目标产品、系统包、网络访问等）。|
| **metadata** | 否 | 用于存储额外元数据的自定义键值映射（Key-Value Mapping） |
| **allowed-tools** | 否 | 以空格分隔的字符串，包含该技能获准使用的工具列表。（实验性功能） |
+ 补充属性location：当前技能目录下SKILL.md的绝对路径
    + 如：当前技能目录为/a/b则location=/a/b/SKILL.md

### 格式整理
+ 将每个skill声明的属性，整理为如下json格式，未声明的属性跳过
```JSON
{
    "name": string,
    "location": string,
    "description": string,
    "license": string,
    "compatibility": string,
    "metadata": {},
    "allowed-tools": string
}
```
+ 将所有skill整理为一个json array，如果存在名称相同的技能，则后覆盖前
```
[
    {
        "name": string,
        "location": string,
        "description": string,
        "license": string,
        "compatibility": string,
        "metadata": {},
        "allowed-tools": string
    },
    {
        "name": string,
        "location": string,
        "description": string,
        "license": string,
        "compatibility": string,
        "metadata": {},
        "allowed-tools": string
    }
]
```


### 定期SKILL.md解析检查
> 新增自 ./iteration/20260512-1/REQUIREMENT.md
+ 每分钟扫描skills目录所有子目录及其子孙目录，检查`SKILL.md`文件是否可以正确解析
+ 如果不能解析则记录解析错误原因附加时间保存在数据库中（文件名为data的sqlite）
+ 如果周期检查可以正确解析了则删除对应的解析错误提醒
+ 解析提醒属性：
    + 错误SKILL.md的路径
    + 错误原因
    + 时间

### Compatibility数组兼容
> 新增自 ./iteration/20260515-1/REQUIREMENT.md
+ compatibility字段支持字符串与字符串列表两种YAML声明格式，并统一整理为标准字符串输出
+ 字符串格式：按原值输出
+ 字符串列表格式：将各项去除首尾空白、忽略空字符串、使用`;`拼接为单个字符串
+ 最终输出结果中的compatibility字段类型固定为字符串
+ 其他字段解析行为保持不变

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
    + 每次生成时实时扫描目录，不缓存技能元数据
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 验证测试
+ 以`test-case`作为指定目录，验证代码生产内容是否符合：
    + 技能__internal_A，绝对路径是否符合
        + description=技能A
        + compatibility=测试内容
    + 技能__internal_c，绝对路径是否符合
        + description:=技能C
        + metadata={os: darwin}
    + 技能__internal_F，绝对路径是否符合
        + description=技能F
        + license=F授权
        + metadata={os: win}

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 相关迭代：iteration/日期/REQUIREMENT.md
+ 同步代码：../integration/REQUIREMENT.md
> 合并截止：./iteration/20260515-1/REQUIREMENT.md，下次合并从此之后的新迭代开始
