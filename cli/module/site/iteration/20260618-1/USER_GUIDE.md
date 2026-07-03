# Site 迭代手册（20260618-1）

## 本次更新

- 前端会话沙盒交互保持不变，仍统一调用：
  - `/api/sandbox_status?agentId=xxx&chatId=yyy`
  - `/api/sandbox=filepick|net|filepick_net|off`
- 后端现在会按当前系统自动选择对应的沙盒实现：
  - macOS：原有 `CLI_SANDBOX.app`
  - Windows/WSL：新的 WSL `bubblewrap` helper
- 因此前端无需区分平台，已有 `目录权限 / 离线执行 / 双重限制` 三个入口继续复用

## 用户可见变化

- 在 WSL 环境下，点击 `目录权限` 或 `双重限制` 时，后端会尝试拉起 WSL 目录授权流程
- 如果 WSL 端未安装 `bubblewrap`，前端会收到后端返回的错误提示，不会影响其它会话功能

## 同步结果

- 本轮主要是后端沙盒实现切换，站点交互协议不变
- 现有沙盒引导、浮层展示、状态刷新逻辑都可以直接复用
