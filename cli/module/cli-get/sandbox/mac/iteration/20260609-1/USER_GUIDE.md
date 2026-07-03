# 迭代说明

本次迭代把 macOS 侧 `CLI_SANDBOX` 改成三套独立 bundle，并保持宿主调用方式不变，仍然通过命令行 `--cmd` 执行后等待返回。

## 新增产物

构建后会生成以下 bundle：

```text
../release/mac/arm/filepick/CLI_SANDBOX.app
../release/mac/arm/net/CLI_SANDBOX.app
../release/mac/arm/filepick_net/CLI_SANDBOX.app
../release/mac/x86/filepick/CLI_SANDBOX.app
../release/mac/x86/net/CLI_SANDBOX.app
../release/mac/x86/filepick_net/CLI_SANDBOX.app
```

对应模式行为：

- `filepick`：执行前选择目录，未选择视为权限拒绝
- `net`：关闭网络
- `filepick_net`：目录选择与关闭网络同时启用

## 构建入口

- `cli/module/cli-get/sandbox/build.sh`
  - 负责产出三模式 `CLI_SANDBOX.app`
- `cli/module/build.sh`
  - 会联动调用上面的构建脚本，并把三模式 bundle 打包进最终 `integration.app/Contents/Helpers/<mode>/CLI_SANDBOX.app`

## 实际实现说明

- `.app` 内包含一个 runner 和一个 helper
- runner 负责读取 bundle 内置模式、目录授权缓存和日志路径
- helper 负责真正执行命令
- 当前版本的权限限制主要依赖 `sandbox-exec`
- 构建阶段会输出 `app.entitlements.plist` 和 `inherit.entitlements.plist`，但默认内容为空，现阶段不靠这些 entitlements 实现目录或网络限制

## 完成情况

- 已完成三种模式拆分为独立 bundle
- 已完成宿主侧保持 `--cmd` 调用方式不变
- 已完成 `sandbox/build.sh` 与 `cli/module/build.sh` 联动
- 已完成 arm64 / amd64 两套目录输出
- 已完成构建/验证日志写入当前目录 SQLite `data`
- 签名流程代码已接入，但是否完成正式签名仍依赖本机证书目录和 keychain，属于交付环境验证项

## 已验证

已执行：

```bash
cd /path/to/deepright/cli/module/cli-get/sandbox/mac
go test ./...
```

另已执行：

```bash
cd /path/to/deepright/cli/module/cli-get/sandbox
go test ./...
```

本轮验证覆盖了：

- 打包配置归一化
- `Info.plist` 内容
- 构建日志落库
- `filepick` 目录缓存与锁等待
- `sandbox-exec` 模式下的命令执行

## 使用建议

- 本地只看产物结构时，可用 `DEEPRIGHT_SKIP_SIGN=1 ./build.sh`
- 需要完整签名时，再提供 `DEEPRIGHT_KEY`、`DEEPRIGHT_P12_PASSWORD`、`DEEPRIGHT_KEYCHAIN_PASSWORD` 等证书参数
