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

### WSL沙箱技术规范
+ 严格遵守技术规范：../../DESIGN.md

### 需求介绍
+ 为WSL `filepick`、`net`、`filepick_net`三种沙盒统一定义常用系统工具、运行库和受信任工具根的挂载策略。
+ 使用Bubblewrap空根文件系统和显式bind mount实现路径豁免；不得采用整棵`/home`、`/mnt`、`/opt`或Windows磁盘的宽泛挂载。
+ 系统工具和运行库必须可执行并只读；用户选择目录仅在需要目录授权的模式中可读写。
+ 临时文件不得使用宿主`/var/tmp`或`/private/var/folders`等共享临时树；必须使用Bubblewrap私有`/tmp`和私有`/var/tmp`。

### 路径授权原则
+ Bubblewrap使用`--unshare-all`和空根文件系统，未显式挂入的路径默认不可见。
+ 系统工具、动态运行库、系统配置和包管理器运行时根必须使用`--ro-bind`挂入，禁止使用`--bind`。
+ 用户选择目录、其`filepath.EvalSymlinks()`解析后的真实路径、CLI_SANDBOX状态目录和DeepRight运行状态目录按现有语义使用精确`--bind`挂入。
+ `filepick`和`filepick_net`不得因运行常用工具而放开选择目录的父目录、`/home`、整棵`/mnt`或整块Windows磁盘。
+ `filepick`和`filepick_net`必须拒绝与系统只读根重叠的选择目录，防止可写业务bind覆盖`/usr`、`/bin`等工具挂载。
+ Linuxbrew、pyenv、nvm、Cargo、Conda等位于用户Home的工具目录不得默认豁免；后续如需支持，必须通过可信宿主配置指定精确工具根并只读挂载，不能读取不受信任环境变量扩权。
+ Windows系统目录与`/mnt/c/Program Files`不得默认挂入或执行；需要Windows工具时应使用独立、受控的宿主桥接方案。

### 特殊处理路径
+ 系统shell、工具、动态运行库和基础配置（只读`--ro-bind`）
    + `/usr`：覆盖`/usr/bin`、`/usr/sbin`、`/usr/lib`和常见的`/usr/local`工具。
    + `/bin`
    + `/sbin`
    + `/lib`
    + `/lib64`
    + `/etc`
+ 按宿主存在情况挂入的包管理器运行时根（只读`--ro-bind`）
    + `/run/current-system/sw`
    + `/nix/store`
+ 用户与应用运行状态目录（可读写`--bind`）
    + 用户本次选择目录。
    + 用户选择目录经`filepath.EvalSymlinks()`解析后的真实路径。
    + `os.UserConfigDir()/CLI_SANDBOX`。
    + `~/deepright`，仅当该目录存在时挂入。
+ 临时目录（沙箱私有、可读写）
    + `/tmp`必须使用`--tmpfs /tmp`创建，子进程`TMPDIR`固定为`/tmp`。
    + `/var/tmp`必须在沙箱内用`--dir /var/tmp`创建；不得再将宿主`/var/tmp`通过`--bind`带入。
+ 明确不豁免的路径
    + `/home`及任何用户Home根目录。
    + `/mnt`、`/mnt/c`和Windows系统目录；若用户选择`/mnt/c/...`下的工作目录，只允许精确挂入该选择子路径及其真实路径。
    + `/opt`整体目录；不得因为某个可选工具位于`/opt/<tool>`而挂入整棵`/opt`。

### 三种沙盒模式下的特殊路径行为
+ `filepick`
    + 系统shell、运行库、基础配置和已识别包管理器运行时根按只读规则挂入。
    + 用户选择目录和其真实路径、CLI_SANDBOX状态目录、已存在的DeepRight运行状态目录按可读写规则挂入。
    + `/tmp`和`/var/tmp`均为沙箱私有可读写目录；设置`TMPDIR=/tmp`。
    + 追加`--share-net`，保留网络。
+ `net`
    + 系统shell、运行库、基础配置和已识别包管理器运行时根按只读规则挂入，保证命令可执行。
    + 不挂入用户选择目录及其真实路径，默认工作目录为私有`/tmp`。
    + CLI_SANDBOX状态目录和已存在的DeepRight运行状态目录维持现有精确可读写挂载。
    + `/tmp`和`/var/tmp`均为沙箱私有可读写目录；设置`TMPDIR=/tmp`。
    + 不追加`--share-net`，依赖`--unshare-all`的网络namespace禁网。
+ `filepick_net`
    + 文件系统行为必须与`filepick`完全一致：系统和工具根只读，选择目录、状态目录和私有临时目录可读写。
    + 不追加`--share-net`，依赖`--unshare-all`的网络namespace禁网。

### 验收要求
+ 三种模式均能通过`/bin/sh`、`/usr/bin`及已挂入的Nix运行时执行常用命令，并且系统/工具根不出现`--bind`可写挂载。
+ `filepick`和`filepick_net`仅挂入用户选择目录及其真实路径；未选择目录、`/home`、整棵`/mnt`和Windows系统目录不得出现在Bubblewrap参数中。
+ `filepick`和`filepick_net`选择`/usr`、`/bin`、`/sbin`、`/lib`、`/lib64`、`/etc`或其祖先目录时必须拒绝执行，避免覆盖系统只读挂载。
+ `net`不挂入业务工作目录且默认工作目录为`/tmp`；`filepick_net`与`net`均不得出现`--share-net`。
+ 三种模式均不得出现宿主`/var/tmp`的`--bind`参数；必须设置`TMPDIR=/tmp`并在沙箱内创建`/var/tmp`。

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
