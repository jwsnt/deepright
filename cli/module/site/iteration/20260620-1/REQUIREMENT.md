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
+ 在设置中的每个模型的多模态输出输入框的左上，展示一个闪烁的荧光蓝色小球的小图标
    + 如果模型不支持多模态输出，则不展示
+ 点击蓝色小球则收起配置浮层，展开一个多模态输出配置浮层（需要有动画过度）
+ 多模态输出配置浮层由多个Key-Val输入框组成，默认包含:
    + aspectRatio:16:9
    + imageSize:2K
    + 其他由随意添加，Key/Val都只能英文
+ 点击取消（绑定ESC）则关闭浮层，点击保存（绑定回车）则使用/api/config和/api/edit接口，增加该Agent和该模型（两个维度的）media属性的存储
```
{
    ... Agent相关config配置中其他属性
    "media": {
        "模型服务商名称": {
            ... 多组属性
        }
    }
}
```
    + 模型服务商名称与选择模型时的名称保持一致
+ 该配置是Agent维度的，切换Agent时需要重新读取最新配置
+ Integration需求：../../../integration/iteration/20260620-1/REQUIREMENT.md
+ Proxy需求：../../../proxy/iteration/20260620-1/REQUIREMENT.md

### 编写代码
+ 以现有site页面技术栈编写以上代码，要求：
    + 在../../index.html中按现有HTML/CSS/JavaScript组织方式实现
    + 不引入新的构建流程和额外运行时依赖
    + 代码简洁，尽量复用现有SSE解析、历史恢复、消息渲染和动画逻辑

### 撰写手册
+ 如有必要同步更新USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
