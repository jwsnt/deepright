# 20260524-2 使用说明

## 迭代目标

本次迭代把 `proxy` 的默认模板新建 Agent 行为收口到 `integration`，并补齐 release 构建产物中的 `config/` 目录。

## 行为说明

- 新增服务启动参数 `--default-dir`
- 未显式传参时，默认使用应用启动目录下的 `config/`
- `GET /api/agent/init?name=xxx` 创建 Agent 后，会把 `default-dir` 目录中的全部内容复制到该 Agent 目录
- `module/build.sh` 现在会把 `module/config` 整体复制到 `module/release/config`
- `default-dir` 缺失、不是目录或复制失败时，请求会直接失败，并回滚刚创建的 Agent 目录

## 验收建议

1. 在 `module/` 目录执行 `sh ./build.sh`，确认 `module/release/config` 存在。
2. 从 `module/release` 启动 `./integration` 后调用 `/api/agent/init?name=test-agent`，确认新 Agent 会带上 `release/config` 中的默认模板文件。
3. 把 `default-dir` 指向不存在目录后重试，确认接口返回失败且不会留下半成品 Agent 目录。
