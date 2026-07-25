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
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+

### 编写代码
+ 最小范围更新，不新增外部依赖。

### 撰写手册
+ 更新 `../../USER_GUIDE.md`、`../../../site/USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`，说明仅纯文本文件创建成功后自动打开预览并进入编辑态，以及目录、失败和其它文件类型均不自动打开的行为。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求，不记录实现过程。
