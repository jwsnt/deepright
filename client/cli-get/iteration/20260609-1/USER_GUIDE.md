# 迭代说明

本次迭代把 `cli-get` 的会话级沙盒开关从旧布尔值升级为文本枚举，并接入新的三模式 `CLI_SANDBOX`。

## 行为变更

- `sandbox_exe` 只认 3 个枚举值：
  - `filepick`
  - `net`
  - `filepick_net`
- `sandbox_exe` 为空、缺失或不是以上 3 个值时，继续沿用原链路：
  - `cli/get -> 本地 Shell 执行 -> cli/pub`
- `sandbox_exe` 命中以上 3 个值时，切换为对应模式的 `CLI_SANDBOX`：
  - `cli/get -> CLI_SANDBOX --cmd -> cli/pub`
- 如果任务里带 `subOps.exempted=true`，即使当前会话已开启沙盒，也会直接回退到原链路

## 状态存储

- `sandbox_exe` 状态保存在当前工作目录 SQLite `data` 的 `cli_sandbox_state` 表
- 主键为 `agent_id + chat_id`
- 关闭沙盒时会直接删除对应状态行
- 首次运行新版本时，如果检测到旧版布尔字段结构，会直接删除旧表并重建为文本枚举结构

## `sandbox_app` 解析

- 优先使用命令行 `--sandbox_app`
- 未传时读取主程序侧 `config/config.json` 中的 `sandbox_app`
- macOS 下会按模式解析到：
  - `<anchor-parent>/filepick/CLI_SANDBOX.app/Contents/MacOS/CLI_SANDBOX`
  - `<anchor-parent>/net/CLI_SANDBOX.app/Contents/MacOS/CLI_SANDBOX`
  - `<anchor-parent>/filepick_net/CLI_SANDBOX.app/Contents/MacOS/CLI_SANDBOX`

## 完成情况

- 已完成 `sandbox_exe` 三模式枚举接入
- 已完成 `subOps.exempted=true` 的沙盒豁免
- 已完成按 `AgentId + Chat` 读取/写入会话级沙盒状态
- 已完成按模式解析独立 `CLI_SANDBOX.app`
- 已保持未开启沙盒时的原有执行链路不变
- 已更新主模块用户手册与本需求目录下用户手册
- “尽可能缩小包体积”没有仓库内的量化阈值；当前实现已经改为三套独立模式 bundle，但最终交付体积仍建议在正式打包产物上再做一次人工复核

## 已验证

已执行：

```bash
cd /path/to/deepright/cli/module/cli-get
go test ./...
```

本轮测试覆盖了：

- `sandbox_exe` 模式读取
- `subOps.exempted` 解析与跳过沙盒
- `sandbox_app` 路径解析
- 沙盒执行链路与发布结果
- 关闭沙盒时删除会话状态

## 交付建议

- 如果要随 `integration.app` 一起发布，先构建 `cli/module/build.sh`
- 如果只单独验证 `cli-get`，需要保证 `sandbox_app` 指向三模式产物的锚点路径或 `.app` 路径
