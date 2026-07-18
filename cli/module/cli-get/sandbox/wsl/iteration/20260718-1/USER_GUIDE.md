# 迭代说明

本次迭代为 WSL `CLI_SANDBOX` 明确了三种模式共用的系统工具路径与临时目录策略。Bubblewrap 从空根文件系统启动，只挂入必需的只读系统根和精确的可写业务路径。

## 模式行为

| 模式 | 工具与运行库 | 业务目录 | 临时目录 | 网络 |
| --- | --- | --- | --- | --- |
| `filepick` | 只读挂入 | 仅选择目录及其 realpath 可读写 | 私有 `/tmp`、`/var/tmp` | 允许 |
| `net` | 只读挂入 | 不挂入 | 私有 `/tmp`、`/var/tmp` | 拒绝 |
| `filepick_net` | 只读挂入 | 仅选择目录及其 realpath 可读写 | 私有 `/tmp`、`/var/tmp` | 拒绝 |

三种模式都会只读挂入：`/usr`、`/bin`、`/sbin`、`/lib`、`/lib64`、`/etc`，以及存在时的 `/run/current-system/sw`、`/nix/store`。

## 特殊路径

- CLI_SANDBOX 状态目录和已存在的 `~/deepright` 目录维持精确可读写挂载。
- `/tmp` 使用 `--tmpfs /tmp`；`TMPDIR=/tmp`。
- `/var/tmp` 在沙箱内以 `--dir /var/tmp` 创建，宿主 `/var/tmp` 不会被挂入。
- 不会默认挂入 `/home`、`/mnt`、`/mnt/c`、`/opt`、`/mnt/c/Program Files` 或 Windows 系统目录。
- 对 `/mnt/c/...` 工作区，仅挂入用户选择的子目录及其解析后的真实路径。

## 使用提示

- `filepick` 与 `filepick_net` 必须提供或选择有效的 `--allowed-dir`；未授权时拒绝执行。
- 选择目录不得与 `/usr`、`/bin`、`/sbin`、`/lib`、`/lib64`、`/etc` 重叠，否则会被拒绝，避免覆盖只读工具根。
- `net` 不会挂入业务目录，即使内部调用传入目录参数。
- 沙箱内 `PATH` 为 `/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin`。
- Linuxbrew、pyenv、nvm、Cargo、Conda 等用户Home工具目录不默认可见；后续如需支持，必须由可信配置精确加入只读工具根。

## 验证

已执行：

```bash
cd /path/to/deepright/cli/module/cli-get/sandbox/wsl
go test ./...
```
