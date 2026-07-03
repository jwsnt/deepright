# CLI 项目使用手册

## 简介

当前目录是多模块 CLI/HTTP 集成工程，主要能力集中在 `module/` 下。

本次已完成并联动更新的核心模块：

- `module/agent`
- `module/skills`
- `module/proxy`
- `module/integration`
- `module/site`

推荐最终使用入口是 `module/integration`，它会把代理服务、站点、定时任务、Connect 能力统一收口到一个主二进制。

## 目录说明

```text
module/
├── skills/        SKILL.md 扫描与解析告警
├── proxy/         独立代理服务与本地接口
├── integration/   最终统一主程序
└── site/          前端单页站点
```

## 推荐启动方式

优先使用 `integration`：

```bash
cd /path/to/deepright/cli/module/integration
go build -o integration ./
./integration --agent-dir ./agent --site ../site
```

启动后主要入口包括：

- 聊天与本地接口：`http://127.0.0.1:8080`
- 前端站点：`http://127.0.0.1:8080/site/#app`
- 知识库：`http://127.0.0.1:8080/knowledge`
- SKILL 告警：`http://127.0.0.1:8080/skills_warning`
- 已安装应用：`http://127.0.0.1:8080/install_app`

## 本次联动更新

- `agent`
  - Agent metadata 中的 `git` 路径改为每次实时探测，不再跟随共享缓存返回旧值
  - Agent metadata 中的 `skills[].compatibility` 兼容字符串和数组两种写法，并统一输出为字符串
- `proxy`
  - `/v1/chat/completions` 注入的 metadata 会自动继承实时 `git`
  - 注入的 `metadata.agents[].skills[].compatibility` 统一为字符串格式
  - 新增 `GET /install_app`
  - `GET /api/plugins/meta` 不再复用插件发现缓存，每次请求都会实时重新扫描插件并读取最新已保存 meta
- `integration`
  - `/v1/chat/completions`、`cli/get`、`cli/pub` 以及 cron 执行链路都会继承实时 `git`
  - 最终对外暴露的 `skills[].compatibility` 已统一规范为字符串
  - 新增统一收口接口 `GET /install_app`
  - `GET /api/plugins/meta` 与 `proxy` 保持一致，改为每次请求实时重新扫描插件并读取最新已保存 meta
- `site`
  - 左上角即时通讯插件入口在页面可见时会每 `30` 秒自动重新请求 `/api/plugins/meta`
  - 自动刷新会同步插件列表变化，并刷新插件入口与扇形按钮的运行状态展示

如果只想单独调试代理层，也可以使用 `proxy`：

```bash
cd /path/to/deepright/cli/module/proxy
go build -o proxy ./
./proxy --agent-dir ./agent --site ../site
```

## Skills Warning 链路

这次新增了一条从扫描到前端展示的完整链路：

1. `skills` 每次实时扫描 `SKILL.md`，并支持把解析失败写入 `data` sqlite 的 `skills_warning` 表。
2. `proxy` 提供 `GET /skills_warning` 和 `proxy skills-warning`。
3. `integration` 提供 `GET /skills_warning` 和 `integration skills-warning`，作为统一对外入口。
4. `site` 左侧虚拟文件系统每 `30` 秒轮询一次 `/skills_warning`，并高亮错误路径与错误文件。

另外，`SKILL.md` 中的 `compatibility` 现在支持两种写法：

```yaml
compatibility: macOS (Darwin)
```

```yaml
compatibility:
  - macOS (Darwin)
  - zsh shell
```

无论原始写法如何，最终在 `skills / agent / proxy / integration` 的 JSON 输出中都会统一变成：

```json
"compatibility": "macOS (Darwin); zsh shell"
```

告警字段包括：

- `path`：错误 `SKILL.md` 绝对路径
- `reason`：解析失败原因
- `time`：最近一次扫描时间

## 常用命令

`skills`：

```bash
cd /path/to/deepright/cli/module/skills
./skill-scanner ./test-case
./skill-scanner warning-scan --interval 0 ./test-case
./skill-scanner warning-list
```

`proxy`：

```bash
cd /path/to/deepright/cli/module/proxy
./proxy --help
./proxy skills-warning
./proxy skills-warning --refresh --root ./agent
```

`integration`：

```bash
cd /path/to/deepright/cli/module/integration
./integration --help
./integration skills-warning
./integration skills-warning --refresh
```

## 相关手册

详细模块说明请继续查看：

- [skills/USER_GUIDE.md](module/skills/USER_GUIDE.md)
- [proxy/USER_GUIDE.md](module/proxy/USER_GUIDE.md)
- [integration/USER_GUIDE.md](module/integration/USER_GUIDE.md)
- [site/USER_GUIDE.md](module/site/USER_GUIDE.md)
