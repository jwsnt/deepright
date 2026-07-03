### 第一性原则
+ 仅可以新增/更新/删除skills（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ skills介绍：../../REQUIREMENT.md
+ skills手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 兼容compatibility的字符串与字符串列表两种声明格式，并统一整理为标准字符串输出，避免因字段写法差异导致技能解析失败。
+ 功能要求：compatibility支持以下两种YAML格式：字符串、字符串列表
    + 若compatibility为字符串，则按原值输出
    + 若compatibility 为字符串列表，则将各项按声明顺序整理为单个字符串
    + 列表项在整理时需要：去除每项首尾空白、忽略空字符串项、使用;作为拼接分隔符
    + 最终输出结果中的compatibility字段类型固定为字符串
+ 其他字段的解析行为保持不变
+ 案例
``` 输入
compatibility:
  - macOS (Darwin)
  - zsh shell
```
``` 输出
{
  "compatibility": "macOS (Darwin); zsh shell"
}
```

### 编写代码
    + 使用文件名为data的sqlite存储，并使用连接池，避免每次都新建连接
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好

### 验收测试
+ compatibility: macOS (Darwin) 可被正常解析
+ compatibility: [macOS (Darwin), zsh shell] 可被正常解析并输出标准字符串
+ 多行列表形式的compatibility可被正常解析并输出标准字符串
+ 使用列表写法的SKILL.md 不再进入解析告警
+ 现有字符串写法的技能文件解析结果不发生回归

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写



