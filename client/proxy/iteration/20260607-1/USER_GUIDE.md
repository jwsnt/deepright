# 迭代说明

本次迭代为 `proxy` 增加了与 `integration` 对齐的沙盒状态开关。

## 新增能力

- 新增启动参数 `--sandbox`，默认 `false`
- 新增 `GET/POST /api/sandbox?sandbox=true|false`
  - 不带参数时返回当前状态
  - 带参数时会更新当前进程内的沙盒状态

## 目的

- 与 `integration` 的 `/api/sandbox` 保持一致的 HTTP 协议
- 让统一前端、调试脚本和上层 CLI 在 `proxy` / `integration` 间切换时无需改调用方式

## 测试

- 新增 `/api/sandbox` 切换测试
- 新增 `--sandbox` 启动参数默认值与显式开启测试
