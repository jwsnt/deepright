# 迭代说明

本次迭代为 macOS `filepick` 与 `filepick_net` 增加了显式的系统工具路径规则。用户选择工作目录后，命令仍可使用系统 shell、Homebrew 和 Xcode 工具，但这些工具目录不会获得写权限。

## 模式行为

| 模式 | 文件系统行为 | 网络 | 子进程环境 |
| --- | --- | --- | --- |
| `filepick` | 选择目录、沙箱状态目录和 `/tmp` 可读写；特殊工具路径只读 | 允许 | `ZDOTDIR=<选择目录>`、`TMPDIR=/tmp` |
| `net` | 使用基础默认文件策略，不应用本次文件选择白名单 | 拒绝 | 不覆盖 |
| `filepick_net` | 与 `filepick` 相同 | 拒绝 | `ZDOTDIR=<选择目录>`、`TMPDIR=/tmp` |

## 特殊路径

只读路径：

- `/bin`、`/sbin`、`/usr/bin`、`/usr/sbin`
- `/usr/lib`、`/usr/libexec`、`/System/Library`、`/Library/Apple/System/Library`
- `/usr/local/{bin,sbin,lib}`、`/opt/homebrew/{bin,sbin,lib}`
- `/Library/Developer/CommandLineTools/usr/bin`
- `/Applications/Xcode.app/Contents/Developer/usr/bin`
- `/private/etc`、`/private/dev`、`/private/var/{select,run,db}`

可读写路径：

- 用户本次选择目录及其解析后的真实路径
- `~/Library/Containers/cn.deepright.integration/Data/Library/Application Support/deepright`
- `os.UserConfigDir()/CLI_SANDBOX`
- `/private/tmp`、`/tmp`

`/private/var/folders` 不会因临时文件需求而放开。文件选择模式会将 `TMPDIR` 固定为 `/tmp`。

## 使用提示

- `filepick` 与 `filepick_net` 必须先提供或选择有效的 `--allowed-dir`；未授权时拒绝执行。
- 特殊路径的放行不修改 `PATH`。调用 Homebrew 工具前，宿主环境应包含 `/opt/homebrew/bin` 或 `/usr/local/bin`。
- 访问未选择的 `/Users`、`/Volumes`、`/private` 子树仍会被拒绝；需求中列出的运行时例外除外。

## 验证

已执行：

```bash
cd /path/to/deepright/cli/module/cli-get/sandbox
go test ./...
```
