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
