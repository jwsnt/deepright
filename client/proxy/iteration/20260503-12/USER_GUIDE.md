# 20260503-12 使用手册

## 目标

本次迭代将 `Connect` 能力整合进 `proxy` 单可执行文件：

- `proxy` 启动后直接内嵌提供 Connect HTTP 服务
- 新增 `proxy connect ...` 子命令
- 继续保持插件可执行文件独立，不并入 `proxy`
- 已同步覆盖 `20260503-9`、`20260503-10`、`20260503-11` 相关能力

## 使用方式

启动服务：

```bash
./proxy serve --agent-dir ./agent --port 9876
```

启动后，当前进程会同时提供以下 Connect HTTP 路径：

- `GET /api/connect/health`
- `GET|POST|PUT|DELETE /api/connect/meta`
- `GET|POST /api/connect/request`
- `GET|POST /api/connect/response`

示例：

```text
POST /api/connect/meta?name=飞书&meta={"token":"abc"}&stream=true&callback=./feishu&agent=A&model=OpenAI
```

## CLI

通过 `proxy connect` 直接调用内嵌 Connect 子模块：

```bash
./proxy connect meta-create --name 飞书 --meta '{"token":"abc"}' --callback ./feishu --agent A --model OpenAI
```

```bash
./proxy connect meta-list
```

说明：

- `proxy connect` 会复用 `connectsvc` 的同一套校验与存储逻辑
- 默认数据库仍为当前目录下的 `data`
- 默认会继承 `agent-dir` 与 `connect-cache` 配置

## 整合规则

- `Connect` 不再要求先独立启动 `./connect`
- `proxy` 启动后即可直接服务站点、插件配置、插件日志与 Connect 元数据操作
- 插件可执行文件仍保持独立，日志与回调路径解析逻辑不变
- `integration` 已同步保持相同接口族

## 同步结果

- `proxy` 主手册已补充内嵌 Connect 服务说明
- `proxy` 已新增 `/api/connect/*` 路径与 `proxy connect` CLI
- `proxy` 测试已补充嵌入式 Connect handler 验证
