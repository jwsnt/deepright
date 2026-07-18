# CLI_SANDBOX MAC

`CLI_SANDBOX MAC` 负责把 `CLI_SANDBOX` 打包为可直接命令行调用的 `.app`，并为 `filepick`、`net`、`filepick_net` 三种模式分别生成独立 bundle。

它会：

- 构建 `CLI_SANDBOX` helper 二进制
- 构建一个 `.app` 内部运行包装器
- 生成 `Info.plist`
- 生成用于签名的 entitlements plist 文件
- 把 runner 与 helper 装配成 `.app`
- 在可用时完成 codesign
- 通过 [build.sh](build.sh) 从 `DEEPRIGHT_KEY` 指向的证书目录自动完成导证、签名与输出
- 记录 build / verify 结果到当前目录的 SQLite `data`

## 输出结构

直接使用构建脚本：

```bash
cd /path/to/deepright/cli/module/cli-get/sandbox/mac
export DEEPRIGHT_KEY=/path/to/key
./build.sh
```

`build.sh` 会把产物输出到：

```text
../release/mac/arm/filepick/CLI_SANDBOX.app
../release/mac/arm/net/CLI_SANDBOX.app
../release/mac/arm/filepick_net/CLI_SANDBOX.app
../release/mac/x86/filepick/CLI_SANDBOX.app
../release/mac/x86/net/CLI_SANDBOX.app
../release/mac/x86/filepick_net/CLI_SANDBOX.app
```

手工指定 Go 构建器输出时，默认输出到：

```text
./dist/CLI_SANDBOX.app
```

单个模式 bundle 的内部结构：

```text
CLI_SANDBOX.app
└── Contents
    ├── Helpers
    │   └── CLI_SANDBOX
    ├── MacOS
    │   └── CLI_SANDBOX
    ├── Resources
    │   └── runner-config.json
    └── Info.plist
```

其中：

- `Contents/MacOS/CLI_SANDBOX` 是 `.app` 主可执行文件
- `Contents/Helpers/CLI_SANDBOX` 是真正执行命令的 helper
- `runner-config.json` 会写入当前 bundle 的 `mode` 与 `bundleId`
- 主可执行文件会把 `--cmd` / `--shell` / `--timeout` / `--allowed-dir` 连同 bundle 内置的 `mode` 一起透传给 helper

## 实际限制方式

- `filepick`：运行前弹出目录选择器，命令在所选目录下启动，并通过 `sandbox-exec` 限制目录访问
- `net`：通过 `sandbox-exec` 关闭网络
- `filepick_net`：同时启用目录选择与网络关闭
- 当前版本真正的限制能力来自 runner + helper + `sandbox-exec`
- 构建过程虽然会生成 `app.entitlements.plist` / `inherit.entitlements.plist`，但默认内容为空，现阶段主要用于统一签名产物结构，不承担实际权限收口

## 文件选择模式的路径权限

`filepick` 和 `filepick_net` 会先拒绝访问 `/Users`、`/Volumes`、`/private`，再按最小范围重放开运行命令所需路径。选择工作目录不意味着只能运行该目录中的可执行文件；系统工具和已安装开发工具仍可启动，但它们保持只读。

| 路径类别 | `filepick` | `net` | `filepick_net` |
| --- | --- | --- | --- |
| 用户选择目录及其 symlink 真实路径 | 可读写 | 不适用 | 可读写 |
| 系统工具、运行库、Homebrew、Xcode | 只读 | 基础默认文件策略 | 只读 |
| `/private/etc`、`/private/dev`、`/private/var/{select,run,db}` | 只读 | 基础默认文件策略 | 只读 |
| `/private/tmp`、`/tmp` | 可读写 | 基础默认文件策略 | 可读写 |
| 网络 | 允许 | 拒绝 | 拒绝 |

### 特殊路径清单

以下系统和工具路径在文件选择模式下允许读取、加载依赖和执行，但明确拒绝写入：

- `/bin`、`/sbin`、`/usr/bin`、`/usr/sbin`
- `/usr/lib`、`/usr/libexec`、`/System/Library`、`/Library/Apple/System/Library`
- Intel Homebrew：`/usr/local/bin`、`/usr/local/sbin`、`/usr/local/lib`
- Apple Silicon Homebrew：`/opt/homebrew/bin`、`/opt/homebrew/sbin`、`/opt/homebrew/lib`
- Command Line Tools：`/Library/Developer/CommandLineTools/usr/bin`
- Xcode：`/Applications/Xcode.app/Contents/Developer/usr/bin`
- `/private/etc`、`/private/dev`、`/private/var/select`、`/private/var/run`、`/private/var/db`

以下路径在文件选择模式下可读写：

- 本次 `--allowed-dir` 指定或在目录选择器中选中的目录
- 该目录经符号链接解析后的真实路径
- `~/Library/Containers/cn.deepright.integration/Data/Library/Application Support/deepright`
- `os.UserConfigDir()/CLI_SANDBOX`
- `/private/tmp`、`/tmp`

为避免 macOS 默认 `TMPDIR` 指向 `/private/var/folders` 而扩大可访问范围，`filepick` 和 `filepick_net` 的子进程会使用：

```text
ZDOTDIR=<选择目录>
TMPDIR=/tmp
```

不要依赖选中目录的父目录访问；`..`、父目录枚举或祖先目录元数据读取仍可能被拒绝。路径放行也不会修改 `PATH`，若要直接调用 Homebrew 工具，宿主环境仍应包含 `/opt/homebrew/bin` 或 `/usr/local/bin`。

## 打包并签名

```bash
cd /path/to/deepright/cli/module/cli-get/sandbox/mac
go run . \
  --sandbox-src .. \
  --output-dir ./dist \
  --app-name CLI_SANDBOX \
  --bundle-id cn.deepright.cli-sandbox.net \
  --mode net
```

如果不传 `--identity`，会自动优先选择：

1. `Developer ID Application`
2. `Apple Development`

如果只想本地出 unsigned 产物：

```bash
DEEPRIGHT_SKIP_SIGN=1 ./build.sh
```

## build.sh 证书约定

`DEEPRIGHT_KEY` 需要指向证书目录。脚本会自动导入目录下所有：

- `*.cer`
- `*.p12`

并会优先通过环境变量读取密码：

- `DEEPRIGHT_P12_PASSWORD`
- `DEEPRIGHT_KEYCHAIN_PASSWORD`
- `DEEPRIGHT_IDENTITY`

如果没有环境变量，则会尝试读取目录中的可选文本文件：

- `DeepRight_p12_password.txt`
- `p12_password.txt`
- `password.txt`
- `identity.txt`
- `developer_id_identity.txt`

## 常用参数

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--sandbox-src` | `..` | `CLI_SANDBOX` Go 模块路径 |
| `--output-dir` | `./dist` | `.app` 输出目录 |
| `--app-name` | `CLI_SANDBOX` | 应用名 |
| `--bundle-id` | `cn.deepright.cli-sandbox` | Bundle Identifier，同时影响容器路径 |
| `--mode` | 空 | 必填；`filepick` / `net` / `filepick_net` |
| `--identity` | 自动选择 | 指定 codesign identity |
| `--keychain` | 空 | 指定签名时使用的 keychain |
| `--version` | `1.0.0` | `CFBundleShortVersionString` |
| `--build-number` | `1` | `CFBundleVersion` |
| `--network-client` | `true` | 保留参数；当前版本不会把它写入 entitlements |
| `--network-server` | `true` | 保留参数；当前版本不会把它写入 entitlements |
| `--user-selected-read-only` | `false` | 保留参数；当前版本不会把它写入 entitlements |
| `--user-selected-read-write` | `false` | 保留参数；当前版本不会把它写入 entitlements |
| `--downloads-read-only` | `false` | 保留参数；当前版本不会把它写入 entitlements |
| `--downloads-read-write` | `false` | 保留参数；当前版本不会把它写入 entitlements |
| `--hardened-runtime` | `false` | 为 app 签名时增加 `--options runtime` |
| `--skip-sign` | `false` | 跳过 codesign，直接输出 unsigned app |
| `--verify-only` | `false` | 只校验现有 `.app` |
| `--app-path` | 空 | `--verify-only` 时要验证的 `.app` 路径 |

## 运行

构建完成后通过对应模式 bundle 内主二进制执行单条命令：

```bash
/path/to/deepright/cli/module/cli-get/sandbox/release/mac/arm/net/CLI_SANDBOX.app/Contents/MacOS/CLI_SANDBOX \
  --cmd "cat hello.txt | wc -l"
```

也可以继续附带：

- `--shell /bin/zsh`
- `--timeout 5000`
- `--log-file /absolute/path/to/sandbox.log`
- `--allowed-dir /absolute/path/to/workspace`

## 日志

如果没有显式传 `--log-file`，`.app` 主进程会把 helper 的日志默认落到容器里：

```text
~/Library/Containers/<bundle-id>/Data/Library/Logs/CLI_SANDBOX/sandbox.log
```

## 验证

只验证已生成的 `.app`：

```bash
go run . \
  --verify-only \
  --app-path ./dist/CLI_SANDBOX.app
```

底层使用：

```bash
codesign --verify --deep --strict --verbose=2 ./dist/CLI_SANDBOX.app
```

## 构建日志

SQLite 文件名固定为：

```text
data
```

当前会记录到表 `mac_build_log`：

- `action`
- `status`
- `app_path`
- `identity`
- `detail`
- `created_at`

## 限制说明

- `filepick` 目录选择依赖桌面会话；无界面环境建议用 `CLI_SANDBOX_ALLOWED_DIR=/absolute/path` 或 `--allowed-dir` 直接提供已选目录
- `net` / `filepick_net` 的网络关闭依赖系统自带 `/usr/bin/sandbox-exec`
- `net` 模式只关闭网络，不启用文件选择路径限制，也不会覆盖 `ZDOTDIR` 或 `TMPDIR`
- `filepick` / `filepick_net` 即使允许读取系统工具目录，也会拒绝写入这些工具目录
- `--verify-only` 只能验证已有 bundle；完整签名验证仍需要本机存在可用证书或 keychain
