# WSL CLI_SANDBOX 使用手册

## 本次更新

- 新增 WSL 版 `CLI_SANDBOX`，使用 `bubblewrap` 实现 3 种模式：
  - `filepick`
  - `net`
  - `filepick_net`
- `--allowed-dir`、`CLI_SANDBOX_ALLOWED_DIR`、`CLI_SANDBOX_FORCE_PICK` 都保留可用
- WSL 下优先尝试通过 `powershell.exe` 拉起 Windows 原生目录选择器；不可用时回退到 `zenity`
- `net` / `filepick_net` 使用 `bubblewrap` 的 network namespace 做禁网

## 构建

```bash
cd /path/to/deepright/cli/module/cli-get/sandbox/wsl
./build.sh
```

产物目录：

```text
../release/wsl/x86/filepick/CLI_SANDBOX
../release/wsl/x86/net/CLI_SANDBOX
../release/wsl/x86/filepick_net/CLI_SANDBOX
../release/wsl/arm/filepick/CLI_SANDBOX
../release/wsl/arm/net/CLI_SANDBOX
../release/wsl/arm/filepick_net/CLI_SANDBOX
```

主应用 `/Users/shenjiawei/Documents/code/deepright/cli/module/build.sh` 现在也会把这些二进制复制进 linux 发布物的 `helpers/<mode>/CLI_SANDBOX`。

## 运行

```bash
/path/to/CLI_SANDBOX --cmd "pwd"
```

预写入授权目录：

```bash
/path/to/CLI_SANDBOX --allowed-dir /absolute/path/to/workspace
```

说明：

- 每个 mode 对应一个独立二进制，mode 已在构建时固化
- 成功时输出写到 `stdout`，退出码为 `0`
- 失败时输出写到 `stderr`，退出码为 `1`
- 未安装 `bubblewrap` 时会返回 `CLI_SANDBOX当前系统未安装bubblewrap`
- 没有显式目录且无法拉起目录选择器时，会提示显式传入 `--allowed-dir`

## 目录授权

- 不再复用 helper 本地“最后一次目录”缓存
- 环境变量 `CLI_SANDBOX_ALLOWED_DIR` 优先级最高
- 环境变量 `CLI_SANDBOX_FORCE_PICK=1` 会触发一次新的目录选择
- `--allowed-dir` 支持直接传 WSL 路径，也支持 Windows 路径，helper 会通过 `wslpath` 自动转换

## 隔离语义

- `filepick`：仅绑定授权目录和默认运行目录，保留网络
- `net`：不绑定业务目录，并通过 `bubblewrap` 隔离网络
- `filepick_net`：同时启用目录白名单和禁网
- shell 内部会使用独立的 `HOME` / `XDG_*` / `ZDOTDIR`，避免读取真实用户目录下的 dotfiles

## 系统工具与特殊路径

WSL 版使用 Bubblewrap 的空根文件系统。未被显式挂入的宿主路径默认不可见；常用系统工具通过只读挂载提供给三种模式，而不是把用户目录或 Windows 磁盘加入白名单。

| 路径类别 | `filepick` | `net` | `filepick_net` |
| --- | --- | --- | --- |
| 系统工具、运行库和基础配置 | 只读 | 只读 | 只读 |
| `/run/current-system/sw`、`/nix/store`（存在时） | 只读 | 只读 | 只读 |
| 用户选择目录及其 realpath | 可读写 | 不挂入 | 可读写 |
| CLI_SANDBOX / DeepRight 精确状态目录 | 可读写 | 可读写 | 可读写 |
| `/tmp`、`/var/tmp` | 沙箱私有可读写 | 沙箱私有可读写 | 沙箱私有可读写 |
| 网络 | 允许 | 拒绝 | 拒绝 |

### 只读工具根

以下路径仅在宿主存在时使用 `--ro-bind` 挂入：

- `/usr`、`/bin`、`/sbin`
- `/lib`、`/lib64`、`/etc`
- `/run/current-system/sw`、`/nix/store`

`/usr` 已覆盖大多数发行版的 `/usr/bin`、`/usr/sbin`、`/usr/lib` 和 `/usr/local`。沙箱内 `PATH` 以系统路径为基础，并在其前加入经验证的用户 Python 环境 `bin` 目录：

```text
/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
```

为兼容图片主体提取等 Python 工具，WSL 沙盒会只读继承用户已安装的 Python 运行时：`~/.local` 的脚本与用户 site-packages、pyenv、conda/miniforge、virtualenv、Poetry、asdf、mise，以及 Integration 启动时已在宿主 `PATH` 中发现的自定义 Python。每个环境仅挂入对应运行时根目录和 `bin`，不会挂入整个用户 Home；激活的 `VIRTUAL_ENV` 和 `CONDA_PREFIX` 优先。

### 工作区、临时目录与禁止路径

- `filepick` 与 `filepick_net` 仅以可读写方式挂入本次 `--allowed-dir` 及其 `filepath.EvalSymlinks()` 解析路径。
- `net` 不挂入业务工作目录；即使内部调用误传目录，也不会生成该目录的 bind mount。
- `filepick` / `filepick_net` 不接受与 `/usr`、`/bin`、`/sbin`、`/lib`、`/lib64`、`/etc` 重叠的选择目录，防止可写工作区覆盖只读系统工具根。
- `/tmp` 通过 `--tmpfs /tmp` 创建，`TMPDIR` 固定为 `/tmp`。
- `/var/tmp` 是沙箱内创建的私有目录，不再 bind 宿主 `/var/tmp`。
- 不会默认挂入 `/home`、`/mnt`、`/mnt/c`、`/opt`、`/mnt/c/Program Files` 或 Windows 系统目录。
- 若用户选择的是 `/mnt/c/...`，仅精确挂入被选择的子目录及其 realpath，不会暴露整块 Windows 磁盘。

位于用户 Home 的 Linuxbrew、nvm、Cargo 等目录不会默认豁免。Python 运行时是例外：仅当目录属于受支持的 Python 环境，或已由宿主 `PATH` 发现为可执行 Python 时，才以只读方式精确挂入。其它工具仍应通过受信任的后续配置挂入精确工具根；不要通过环境变量放大路径权限。
