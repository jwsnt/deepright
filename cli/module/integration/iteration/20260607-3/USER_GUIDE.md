# 迭代说明

本次迭代补齐了 `integration` 在 macOS 下的固定运行目录规则，避免 `.app` 包启动时把运行态数据误写到 `Contents/MacOS`，同时确保插件继续从应用资源目录读取。

## 新增能力

- macOS 下共享 sqlite 固定读写 `~/Library/Application Support/deepright/data`
- macOS 下 `knowledge` 目录固定为 `~/Library/Application Support/deepright/knowledge`
- macOS 下 `--agent-dir` 默认固定为 `~/Library/Application Support/deepright/agent`
- macOS `.app` 形态下，插件固定读取 `integration.app/Contents/Resources/plugins`
- macOS `.app` 形态下，`config`、`site` 等资源目录统一按 `Contents/Resources` 解析

## 兼容性

- 非 macOS 系统继续沿用原有目录规则
- macOS 下插件二进制仍然随应用包分发，不会复制到运行时目录
- macOS 下运行态数据与应用资源目录彻底分离，避免升级应用包时污染运行数据

## 测试

- 补充了 `.app` 包资源目录解析测试
- 修复并覆盖了插件目录固定指向 `Contents/Resources/plugins` 的测试
- 保留并通过了 macOS 默认 `agent-dir`、共享 `data` 路径相关测试
