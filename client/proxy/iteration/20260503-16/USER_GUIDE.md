# 20260503-16 使用手册

## 目标

本次迭代为 `proxy` 新增插件运行状态查询能力：

- 新增 `GET /api/plugins/status?name=xxx`
- 新增 `proxy plugins status --name xxx` CLI
- `integration` 同步提供相同 HTTP 接口与顶层 CLI 收口
- 底层复用 `connect list-plugins` 同一套插件扫描逻辑定位插件

## HTTP 接口

请求：

```text
GET /api/plugins/status?name=feishu
```

也支持覆盖默认 PID 文件：

```text
GET /api/plugins/status?name=feishu&pid-file=../plugins/feishu.pid
```

成功响应示例：

```json
{
  "status": 0,
  "data": {
    "key": "feishu",
    "name": "飞书",
    "path": "/abs/path/plugins/feishu",
    "pid": 12345,
    "pidFile": "/abs/path/plugins/feishu.pid",
    "started": true
  }
}
```

失败响应示例：

```json
{
  "status": 1,
  "content": "name is required"
}
```

参数说明：

- `name`：必填，支持插件展示名、插件二进制文件名和插件绝对/相对路径
- `pid-file`：可选，显式指定 PID 文件路径；未传时默认使用插件目录下同名 `.pid` 文件

## CLI

查询插件状态：

```bash
./proxy plugins status --name feishu
```

指定 PID 文件：

```bash
./proxy plugins status --name feishu --pid-file ../plugins/feishu.pid
```

`integration` 顶层也支持同样能力：

```bash
./integration plugins status --name feishu
```

## 行为规则

- 先按 `connect` 的插件发现逻辑解析出插件 `key`、展示名和二进制路径
- 再根据 PID 文件内容判断对应进程是否仍然存活
- 若进程不存在、PID 文件不存在或 PID 非法，则返回 `started=false`、`pid=0`
- HTTP 与 CLI 复用同一套 `connectsvc.PluginStatusByName` 逻辑

## 完成情况

- `proxy` 已支持 `GET /api/plugins/status`
- `proxy` 已支持 `plugins status` CLI
- `integration` 已同步注册 `/api/plugins/status`
- `integration` 已同步提供 `integration plugins status`
- `proxy` 与 `integration` 主手册已同步补充说明
- 自动化测试已覆盖 `connectsvc`、`proxy`、`integration` 三层状态查询场景
