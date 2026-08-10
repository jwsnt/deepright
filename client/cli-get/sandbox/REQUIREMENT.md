### 第一性原则
+ 仅可以新增/更新/删除当前需求文档（REQUIREMENT.md）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../DESIGN.md

### CLI-GET
+ CLI-GET元数据介绍：../REQUIREMENT.md
+ CLI-GET元数据手册：../USER_GUIDE.md
+ CLI-GET元数据迭代：../iteration/日期/REQUIREMENT.md

### 需求介绍
+ 新建一个CLI_SANDBOX沙盒，允许使用沙盒执行系统命令，并通过控制台Output流返回结果
+ 案例：执行"cat hello.txt | wc -l"
``` 沙盒APP为/home/SANDBOX.app
 /home/SANDBOX.app -cmd "cat hello.txt | wc -l"
```
+ 响应结果需要与直接调用"cat hello.txt | wc -l"完全相同
+ 在当前目录增加sandbox模块自己的build.sh，为CLI_SANDBOX构建MAC的沙盒.app
    + .app分mac/x86和mac/arm放在当前目录的release/mac/arm和release/mac/x86
    + build.sh放在当前目录
+ 为主build.sh中联动构建MAC的Sandbox的build.sh，区分arm和x86将沙箱打包到最终.app

### 沙箱配置
+ MAC沙箱：
    + MAC沙箱介绍：mac/REQUIREMENT.md
    + MAC沙箱手册：mac/USER_GUIDE.md
    + MAC沙箱迭代：mac/iteration/日期/REQUIREMENT.md
+ WSL沙箱：
    + MAC沙箱介绍：wsl/REQUIREMENT.md
    + MAC沙箱手册：wsl/USER_GUIDE.md
    + MAC沙箱迭代：wsl/iteration/日期/REQUIREMENT.md

### 编写代码
+ 以Golang编写以上代码，要求：
    + 能用开源包的就用开源包
    + 代码简洁，包体积越小越好
+ 作为其他模块可以调用的子模块和可独立运行的CLI命令来编写

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 相关迭代：iteration/日期/REQUIREMENT.md
+ 同步代码：../../integration/REQUIREMENT.md
> 合并截止：./iteration/20260513-1/REQUIREMENT.md，下次合并从此之后的新迭代开始
