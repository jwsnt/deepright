# Integration 迭代手册（20260615-2）

## 本次更新

- 主应用目录下的 `config/` 现在拆分为两类用途
- `config/config.json` 只用于 `integration` 主应用启动配置
- `config/` 下除 `config.json` 外的模板文件和目录，例如 `SOUL.md`、`USER.md`、`skills/`，用于创建新 Agent 或补齐空 `agent-dir` 时复制到 Agent 工作目录
- 打包脚本会继续把整个 `config/` 目录打包到交付物中，确保主应用配置和 Agent 模板都随应用一起发布
- 新建 Agent 后会单独初始化一个空的 `config.json`，不会继承主应用 `config/config.json` 的内容

## 目录语义

主应用目录示例：

```text
config/
├── config.json
├── SOUL.md
├── USER.md
└── skills/
```

语义说明：

- `config/config.json`
  - 给 `integration` 主程序读取启动参数
  - 打包后保留在应用目录，或 macOS `.app` 的 `Contents/Resources/config/config.json`
- `config/` 中其他文件
  - 作为默认 Agent 模板
  - 在 `GET /api/agent/init?name=...` 或启动时自动补齐 `DEF_AGENT` 时复制到 Agent 目录

## 行为变化

- 创建新 Agent 时，不再把主应用 `config/config.json` 复制进 Agent 工作目录
- 新 Agent 的 `config.json` 会固定初始化为：

```json
{}
```

- 如果 `agent-dir` 为空，启动时自动创建的 `DEF_AGENT/` 也遵循同样规则
- `DEF_AGENT/skills` 仍会自动补齐

## 打包说明

- `build.sh` 会把 `module/config` 整体复制到 release 目录下的 `config/`
- macOS `.app` 打包时，也会把 release 中的 `config/` 复制到 `Contents/Resources/config/`
- 因此同一份 `config/` 同时承载：
  - 主应用启动配置入口 `config/config.json`
  - 默认 Agent 模板资源

## 使用示例

主应用 `config/config.json` 示例：

```json
{
  "host": "https://www.deepright.cn",
  "agentDir": "agent",
  "default_dir": "config",
  "site": "site"
}
```

当调用：

```bash
./integration
```

会发生：

- 主程序从 `config/config.json` 读取启动参数
- 如果 `agent-dir` 为空，会用 `config/` 中除 `config.json` 外的内容创建 `DEF_AGENT/`
- `DEF_AGENT/config.json` 单独写入空对象 `{}`，不会复制主应用配置

## 同步结果

- `integration/main.go` 已区分主应用 `config/config.json` 和 Agent 模板复制逻辑
- `integration/main_test.go` 已覆盖新建 Agent、补齐 `DEF_AGENT` 时不继承主应用 `config.json` 的场景
- `build.sh` 已保持 `config/` 目录进入 release 和 macOS `.app` 资源目录
