# Integration 迭代手册（20260618-1）

## 本次更新

- 会话沙盒继续保持 3 种模式：
  - `filepick`
  - `net`
  - `filepick_net`
- macOS 仍然走原有 `CLI_SANDBOX.app` 路径，不改现有 bundle 结构
- WSL/Linux 新增独立 `bubblewrap` helper，发布后位于 `helpers/<mode>/CLI_SANDBOX`
- `/api/sandbox=*`、`/api/sandbox_status`、`integration sandbox` CLI 的协议不变
- `integration /api/cmd` 与内部 `cli/get -> exec -> cli/pub` 链路都会按当前系统自动选择对应沙盒 helper

## 运行路径

- macOS：
  - `integration.app/Contents/Helpers/<mode>/CLI_SANDBOX.app/Contents/MacOS/CLI_SANDBOX`
- WSL/Linux：
  - `<integration-exec-dir>/helpers/<mode>/CLI_SANDBOX`

## 同步结果

- `integration/main.go` 已支持双平台 helper 路径解析
- `cli/module/build.sh` 会在 linux 发布物里打包 WSL helper
- `integration/main_test.go` 已补充 linux `helpers/<mode>/CLI_SANDBOX` 解析测试
