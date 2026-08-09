# 迭代手册

## 本次变更

- 收口 Agent 侧 `git` 实时探测能力
- `integration` 的 `/v1/chat/completions`、`cli/get`、`cli/pub`、cron 执行链路都会使用实时 `git`
- 新增统一接口 `GET /install_app`

## 接口说明

### `GET /install_app`

示例：

```bash
curl http://127.0.0.1:8080/install_app
```

返回当前机器待安装应用的字符串数组。

当前已支持的检测项：

- `git`

若 git 未安装，返回：

```json
["git"]
```

若当前支持的应用都已安装，返回：

```json
[]
```

## Metadata 行为

以下链路中的 `metadata.git` 都会每次实时探测：

- `/v1/chat/completions`
- `cli/get`
- `cli/pub`
- integration 内部 cron 执行请求

说明：

- `skills` 仍然保持每次实时扫描
- `git` 现在同样不再受 `--agent-cache` 影响

## 验证方式

```bash
cd /path/to/deepright/cli/module/integration
go test -run 'TestProxyChatCompletions|TestHandleInstallAppReturnsMissingGitWhenNeeded|TestCliGetHeartbeatAndPublishIncludePluginMetadata'
```
