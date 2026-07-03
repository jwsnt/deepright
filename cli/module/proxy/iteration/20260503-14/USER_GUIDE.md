# 20260503-14 使用手册

## 目标

本次迭代为 `proxy` 新增插件启动/停止能力：

- 新增 `POST /api/plugins/start?name=xxx`
- 新增 `POST /api/plugins/stop?name=xxx`
- 新增 `proxy plugins start` 与 `proxy plugins stop` CLI
- 插件仍保持独立可执行文件，`proxy` 只负责定位并调用插件

## HTTP 接口

### 启动插件

请求：

```text
POST /api/plugins/start?name=feishu&connect-bin=./connect
```

成功响应示例：

```json
{
  "status": 0,
  "data": {
    "path": "/abs/path/plugins/feishu",
    "command": ["start", "--connect-bin", "./connect"],
    "output": {
      "status": "started",
      "pid": 56157
    }
  }
}
```

### 停止插件

请求：

```text
POST /api/plugins/stop?name=feishu&pid-file=../plugins/feishu.pid
```

成功响应示例：

```json
{
  "status": 0,
  "data": {
    "path": "/abs/path/plugins/feishu",
    "command": ["--stop", "--pid-file", "../plugins/feishu.pid"],
    "output": {
      "status": "stopped"
    }
  }
}
```

失败响应示例：

```json
{
  "status": 1,
  "content": "plugin not found: feishu"
}
```

参数说明：

- `name`：必填，支持插件展示名、插件文件名或插件路径
- 除 `name` 之外的查询参数都会透传给插件，转换为命令行参数
- 例如 `connect-bin=./connect` 会传递为 `--connect-bin ./connect`

## CLI

启动插件：

```bash
./proxy plugins start --name feishu --connect-bin ./connect
```

停止插件：

```bash
./proxy plugins stop --name feishu --pid-file ../plugins/feishu.pid
```

## 行为规则

- HTTP 与 CLI 复用同一套 `connectsvc.RunPluginAction` 逻辑
- 启动时优先尝试 `<plugin> start`，失败后自动回退到 `<plugin> --start`
- 停止时优先尝试 `<plugin> stop`，失败后自动回退到 `<plugin> --stop`
- 返回值中的 `path` 为最终执行的插件绝对路径
- 返回值中的 `command` 为实际执行成功的参数形式
- 插件输出若是合法 JSON，会自动解析为 JSON；否则按普通文本返回

## 完成情况

- `proxy` 已支持 `/api/plugins/start` 与 `/api/plugins/stop`
- `proxy` 已支持 `plugins start` 与 `plugins stop` CLI
- `proxy` 主手册已同步补充插件启动/停止说明
- 自动化测试已覆盖子命令模式与 `--start` / `--stop` 回退模式
