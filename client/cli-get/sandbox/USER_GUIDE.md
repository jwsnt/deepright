# CLI_SANDBOX

`CLI_SANDBOX` 是 `cli-get` 的沙盒执行子模块。当前版本只保留命令行执行链路：宿主进程通过 `CLI_SANDBOX --cmd "<命令>"` 调起沙盒，沙盒按与 `cli-get` 一致的 `shell -c` 方式执行命令，并通过标准输出 / 标准错误把结果返回给宿主。

- macOS 发布物为 `.app`
- WSL/Linux 发布物为独立 `CLI_SANDBOX` 二进制，内部使用 `bubblewrap`

## 构建 MAC 沙盒 App

在当前目录直接执行：

```bash
cd /path/to/deepright/cli/module/cli-get/sandbox
./build.sh
```

产物会输出到：

```text
./release/mac/arm/filepick/CLI_SANDBOX.app
./release/mac/arm/net/CLI_SANDBOX.app
./release/mac/arm/filepick_net/CLI_SANDBOX.app
./release/mac/x86/filepick/CLI_SANDBOX.app
./release/mac/x86/net/CLI_SANDBOX.app
./release/mac/x86/filepick_net/CLI_SANDBOX.app
```

`/path/to/deepright/cli/module/build.sh` 也会联动调用这里的 `build.sh`，并把三种模式的 bundle 打包进最终的 `integration.app/Contents/Helpers/<mode>/CLI_SANDBOX.app`。

## 构建 WSL 沙盒二进制

在当前目录执行：

```bash
cd /path/to/deepright/cli/module/cli-get/sandbox/wsl
./build.sh
```

产物会输出到：

```text
../release/wsl/x86/filepick/CLI_SANDBOX
../release/wsl/x86/net/CLI_SANDBOX
../release/wsl/x86/filepick_net/CLI_SANDBOX
../release/wsl/arm/filepick/CLI_SANDBOX
../release/wsl/arm/net/CLI_SANDBOX
../release/wsl/arm/filepick_net/CLI_SANDBOX
```

## 能力边界

- 执行单条命令并返回原始输出
- 支持 3 种模式：`filepick`、`net`、`filepick_net`
- 命中沙盒权限拒绝时尽快终止子进程
- `filepick` 模式只使用当前执行显式提供的目录或当次选择结果
- 日志写入 `sandbox.log`

## 运行

```bash
cd /path/to/deepright/cli/module/cli-get/sandbox
go run . --cmd "cat hello.txt | wc -l"
```

显式指定模式：

```bash
go run . --mode net --cmd "curl http://127.0.0.1:9999"
```

为 `filepick` 模式显式提供授权目录：

```bash
go run . --mode filepick --allowed-dir /absolute/path/to/workspace
```

也可以通过 `.app` 内主程序执行：

```bash
/path/to/deepright/cli/module/cli-get/sandbox/release/mac/arm/filepick_net/CLI_SANDBOX.app/Contents/MacOS/CLI_SANDBOX \
  --cmd "cat hello.txt | wc -l"
```

说明：

- `CLI_SANDBOX` 必须带 `--cmd` 或单独带 `--allowed-dir`
- 成功时把原始结果写到 stdout，退出码为 `0`
- 失败时把错误结果写到 stderr，退出码为 `1`
- `--mode filepick` 与 `--mode filepick_net` 会弹出目录选择器；没有选择就返回 `CLI_SANDBOX权限拒绝`
- `--mode net` 与 `--mode filepick_net` 会通过 `sandbox-exec` 关闭网络
- 可以配合 `--shell` 指定执行 shell
- 可以配合 `--timeout` 指定超时毫秒数；传 `0` 时使用默认超时
- 可以配合 `--log-file` 指定日志文件；默认写入 `sandbox.log`
- 可以配合 `--allowed-dir` 显式提供已授权目录；如果同时带 `--cmd`，会直接使用该目录执行命令

## 启动参数

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--mode` | 空 | 沙盒模式：`filepick` / `net` / `filepick_net` |
| `--shell` | 当前 `SHELL` 或 `/bin/sh` | 命令执行 Shell |
| `--log-file` | `sandbox.log` | 沙盒日志文件 |
| `--allowed-dir` | 空 | 为当前执行显式提供授权目录 |
| `--cmd` | 空 | 要执行的单条命令 |
| `--timeout` | `0` | 命令超时，毫秒；`0` 表示走默认超时 |

## 目录授权来源

- `filepick` / `filepick_net` 不再复用本地“最后一次目录”缓存
- 无界面环境可以通过 `--allowed-dir` 或 `CLI_SANDBOX_ALLOWED_DIR=/absolute/path` 直接提供授权目录
- 图形界面路径由当前这一次目录选择器返回，不会影响其他会话

## 日志

- 沙盒开始执行、执行完成都会记录到 `sandbox.log`
- 超时返回 `命令执行超时`
- 被取消返回 `命令被终止`
- 命中沙盒权限拒绝时会尽快终止当前子进程，不等待原命令自行结束
