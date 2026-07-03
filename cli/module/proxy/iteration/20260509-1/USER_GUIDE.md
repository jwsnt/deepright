# 20260509-1 使用手册

## 目标

本次迭代补充了 `proxy` 转发 `/v1/chat/completions` 时的 `metadata` 内容：

- 继续复用 Agent 模块输出的完整元数据
- 当 Agent 元数据中存在 `plugins` 时，转发请求里的 `metadata.plugins` 会一并带上
- 若当前环境没有任何已配置且已启动插件，则不额外补 `plugins`
- 原有 `metadata` 追加覆盖规则保持不变，不影响现有字段

## 行为说明

`proxy` 在接收客户端的 `/v1/chat/completions` 请求后，仍会先读取 Agent 元数据，再把结果注入到请求体的 `metadata` 中。

本次变化后，若 Agent 元数据类似：

```json
{
  "timezone": "Asia/Shanghai",
  "deviceId": "device-001",
  "terminal": "/bin/zsh",
  "plugins": ["browser", "feishu"],
  "agents": [
    {
      "agentId": "A"
    }
  ]
}
```

则 `proxy` 转发到上游 `/v1/chat/completions` 的请求中，`metadata` 会包含：

```json
{
  "timezone": "Asia/Shanghai",
  "deviceId": "device-001",
  "terminal": "/bin/zsh",
  "plugins": ["browser", "feishu"],
  "agents": [
    {
      "agentId": "A"
    }
  ]
}
```

如果前端原本已经传了 `metadata`，仍按现有“追加覆盖”规则处理：

- Agent 扫描出的字段先作为基础值写入
- 前端请求中已存在的同名字段继续覆盖基础值
- 前端未传的字段，例如本次新增的 `plugins`，会自动补入

## `plugins` 来源

- `proxy` 不单独维护插件列表
- `plugins` 直接复用 Agent 模块最新元数据输出
- Agent 侧会优先复用 `connect meta-list` 获取已配置插件的 `key` 列表
- 再按对应插件的 `<plugin-key>.pid` 与进程存活状态过滤出真正已启动的项
- 返回顺序会按插件 `key` 排序
- 若没有任何已配置且已启动插件，则顶层元数据中不包含 `plugins`

## 使用方式

本次迭代没有新增独立 CLI 命令或 HTTP 路径。

只要像以前一样启动 `proxy` 服务并调用：

```text
POST /v1/chat/completions
```

即可自动获得带 `plugins` 的 `metadata` 转发能力。

## 兼容说明

- 不修改 `/v1/chat/completions` 的路径、SSE 转发方式和响应格式
- 不改变已有 `metadata` 合并策略
- 不新增新的数据库表或额外配置项
- `plugins` 仅表示当前 Agent 元数据里识别到的已配置且已启动插件 `key` 列表；单个插件详细状态仍可继续通过 `/api/plugins/status` 查看
