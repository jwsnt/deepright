### 第一性原则
+ 仅可以新增/更新/删除当前需求文档（REQUIREMENT.md）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md

### CLI-GET
+ CLI-GET元数据介绍：../../../../REQUIREMENT.md
+ CLI-GET元数据手册：../../../../USER_GUIDE.md
+ CLI-GET元数据迭代：../../../../iteration/日期/REQUIREMENT.md

### CLI-CLI_SANDBOX
+ CLI-SANDBOX介绍：../../REQUIREMENT.md
+ CLI-SANDBOX手册：../../USER_GUIDE.md
+ CLI-SANDBOX迭代：../../iteration/日期/REQUIREMENT.md

### Mac沙箱技术规范
+ 严格遵守技术规范：../../DESIGN.md

### 需求介绍
+ 为`filepick`和`filepick_net`沙盒补充系统工具、动态运行库、包管理器工具和临时目录的显式路径规则。
+ 用户选择工作目录后，系统命令必须仍可执行；系统工具目录不得因此获得写入权限。
+ 保持工作目录隔离边界：不得因为工具运行需要而放开整个`/Users`、`/Volumes`、`/private`，也不得放开用户临时目录的父路径。

### 路径授权原则
+ 文件选择模式的基础限制为：拒绝访问`/Users`、`/Volumes`和`/private`。
+ 仅对本需求列出的特殊路径重放开访问权限；未列出的用户目录、挂载卷目录和`/private`子目录继续保持拒绝。
+ 系统 shell、系统工具、动态运行库、Homebrew和开发工具路径仅允许读取和加载执行依赖，不允许写入。
+ 用户选择目录、其`filepath.EvalSymlinks()`解析后的真实目录、CLI_SANDBOX运行状态目录和临时目录允许读取和写入。
+ 路径放行不等同于Shell命令发现；调用Homebrew等工具时，宿主环境仍需要在`PATH`中包含对应的`bin`目录。

### 特殊处理路径
+ 系统 shell、系统工具和运行库路径（只读）
    + `/bin`
    + `/sbin`
    + `/usr/bin`
    + `/usr/sbin`
    + `/usr/lib`
    + `/usr/libexec`
    + `/System/Library`
    + `/Library/Apple/System/Library`
+ 包管理器和开发工具路径（只读）
    + Intel Homebrew：`/usr/local/bin`、`/usr/local/sbin`、`/usr/local/lib`
    + Apple Silicon Homebrew：`/opt/homebrew/bin`、`/opt/homebrew/sbin`、`/opt/homebrew/lib`
    + Command Line Tools：`/Library/Developer/CommandLineTools/usr/bin`
    + Xcode：`/Applications/Xcode.app/Contents/Developer/usr/bin`
+ `/private`下的系统运行时路径（只读）
    + `/private/etc`
    + `/private/dev`
    + `/private/var/select`
    + `/private/var/run`
    + `/private/var/db`
    + 以上路径位于基础拒绝范围`/private`内，必须通过显式只读规则重放开。
+ 临时目录（可读写）
    + `/private/tmp`
    + `/tmp`
    + 文件选择模式下必须将子进程`TMPDIR`覆盖为`/tmp`，不得放开macOS默认的`/private/var/folders`及其父目录，避免访问无关用户临时文件。
+ 会话和运行状态目录（可读写）
    + 用户本次选择目录。
    + 用户选择目录经`filepath.EvalSymlinks()`解析后的真实路径。
    + `~/Library/Containers/cn.deepright.integration/Data/Library/Application Support/deepright`。
    + `os.UserConfigDir()/CLI_SANDBOX`。
    + 以上用户目录位于基础拒绝范围`/Users`内，必须仅按上述精确子路径重放开；不得放开其父目录或祖先目录元数据。

### 三种沙盒模式下的路径行为
+ `filepick`
    + 必须选择有效目录；未选择目录时拒绝执行。
    + 当前工作目录必须设置为用户选择目录。
    + 特殊系统工具、运行库、包管理器和开发工具路径按“只读”规则访问。
    + `/private`运行时路径按“只读”规则访问。
    + 选择目录、其真实路径、CLI_SANDBOX状态目录和`/tmp`按“可读写”规则访问。
    + 覆盖子进程环境变量`ZDOTDIR=<选择目录>`和`TMPDIR=/tmp`。
    + 不限制网络访问。
+ `net`
    + 不要求选择目录，不设置工作目录，不注入文件选择模式的特殊路径白名单和环境变量覆盖。
    + 使用基础`(allow default)`文件策略；本需求列出的特殊路径不作为额外白名单生效。
    + 必须拒绝全部网络访问：`(deny network*)`。
+ `filepick_net`
    + 文件系统行为必须与`filepick`完全一致：系统和工具路径只读，选择目录、状态目录和`/tmp`可读写，未选择目录时拒绝执行。
    + 必须覆盖`ZDOTDIR=<选择目录>`和`TMPDIR=/tmp`。
    + 同时必须拒绝全部网络访问：`(deny network*)`。

### 验收要求
+ 在`filepick`和`filepick_net`模式中，`/bin/sh`、`/usr/bin`内系统命令及已安装Homebrew工具能够启动并加载依赖，且无法写入其工具目录。
+ 在`filepick`和`filepick_net`模式中，命令能够在选择目录内创建、修改和删除文件；除本需求明确重放开的运行时子路径外，访问未选择的`/Users`、`/Volumes`或`/private`目录必须被拒绝。
+ 在`filepick`和`filepick_net`模式中，临时文件必须落在`/tmp`；不得因临时文件需求放开`/private/var/folders`。
+ 在`net`和`filepick_net`模式中，任意网络连接必须被拒绝；`filepick`模式不施加网络限制。

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
+ 同步代码：../../../../../integration/REQUIREMENT.md
