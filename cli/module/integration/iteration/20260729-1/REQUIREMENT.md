### 第一性原则

+ 仅可以新增/更新/删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../build.sh`、`../../../build/install.ps1`、`../../../build/USER_GUIDE.md`、`../../../build/USER_GUIDE.txt` 与 `../../../config/app/API.md`、`../../../config/app/CANVAS.md`、`../../../config/app/DESIGN.md`。

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
+ 不新增外部依赖，不破坏既有 API 的字段兼容性。

### 需求介绍

提供由 `config/config.json` 驱动的本机应用安装检查接口，供 Site 在当前运行环境缺少所需应用时提示用户向当前会话发起安装请求。

- 配置项为 `install_app`。其中 `interval` 是正整数扫描间隔，单位为分钟；`content` 是发往会话的请求模板；`linux`、`wsl`、`mac` 分别是普通 Linux、Windows／WSL、macOS 需要检查的应用名称列表。项目默认配置应保留用户给定的 `interval`、`content` 和三组平台列表。
- 当前平台只能采用相应的一组应用：macOS 使用 `mac`；Windows 与运行在 WSL 内的 Linux 使用 `wsl`；其余 Linux 使用 `linux`。名称需去空白、去重；应用清单唯一来源为该配置对象，不支持 `--install_app` CLI 附加项；既有自动检测项仍可并入检查结果。
- 服务端只返回当前平台实际缺失的应用名称，不执行安装命令，也不在用户工作区创建文件。应用已安装的判断必须覆盖该平台的常见可执行文件发现方式；在 WSL 中仅当命令可由 WSL 进程直接执行时才视为已安装，不得因 Windows 宿主机的 `.exe` 或 `/mnt/c` 软件目录而误判。
- `GET /install_app` 必须继续返回既有的 JSON 字符串数组，并保持其既有缓存兼容性。`GET /install_app?details=1` 返回 JSON 对象，包含 `apps`（缺失应用数组）、`interval`（分钟）和 `content`（请求模板）；详情响应不得缓存。
- `details=1` 是前端的周期扫描请求。它必须在本次检查前使应用可用性判断重新生效，以便用户刚完成安装后，下一扫描周期即可反映最新状态，不能被服务端较长的检测缓存延迟。
- 配置缺失、字段为空或 `interval` 非正时，接口仍应稳定可用：扫描间隔回退到 `60` 分钟，请求模板回退到 `请安装 $namelist`。`content` 中的 `$namelist` 由前端替换，后端不得执行或解释该模板。

### 编写代码
+ 最小范围更新，不新增外部依赖。
+ 保持 Integration 与 Proxy 暴露的 `/install_app` 路径、旧数组响应和错误处理语义一致；启动、重启和运行时配置写入均不得写入或覆盖 `install_app` 对象；详情响应只扩展配置数据，不引入安装执行能力。
+ 覆盖配置读取、平台选择、去重、已安装过滤、详情响应字段、缓存控制及刚安装后下一次详情扫描可见的自动化测试。

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`

### 其他要求
+ `REQUIREMENT.md` 仅描述需求，不记录实现过程。
