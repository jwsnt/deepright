# 20260503-11 使用手册

## 目标

本次迭代为 `proxy` 新增了插件配置入口：

- 新增 `POST /api/plugins/config`
- 新增 `proxy plugins config` CLI
- 底层复用 Connect 的元数据创建/更新能力
- 同名插件已存在配置时自动更新，不需要调用方区分 create / update

## HTTP 接口

请求：

```text
POST /api/plugins/config?key=feishu&meta=%7B%22appId%22%3A%22cli-app%22%2C%22appSecret%22%3A%22cli-secret%22%7D&stream=true&agentId=A&chatId=chat-001&model=OpenAI&thinking=true
```

成功响应示例：

```json
{
  "status": 0,
  "data": {
    "id": 1,
    "name": "飞书",
    "meta": "{\"appId\":\"cli-app\",\"appSecret\":\"cli-secret\"}",
    "stream": true,
    "callback": "/abs/path/plugins/feishu",
    "agentId": "A",
    "chatId": "chat-001",
    "model": "OpenAI",
    "thinking": true
  }
}
```

失败响应示例：

```json
{
  "status": 1,
  "content": "model not registered: OpenAI"
}
```

参数：

- `key`：必填，插件运行时主键，例如 `feishu`
- `name`：仅展示字段，不能作为运行时主键
- `meta`：可选，表单配置 JSON 字符串；默认 `{}``
- `stream`：可选，默认 `false`
- `agentId`：必填，绑定的 Agent
- `chatId`：可选，绑定的 CHAT_ID
- `model`：必填，且必须已注册 token
- `thinking`：可选，默认 `false`

## CLI

```bash
./proxy plugins config --key feishu --meta '{"appId":"cli-app","appSecret":"cli-secret"}' --agentId A --model OpenAI
```

也可以显式指定流式、会话和深度思考：

```bash
./proxy plugins config --key feishu --meta '{"appId":"cli-app"}' --stream --agentId A --chatId chat-001 --model OpenAI --thinking
```

## 行为规则

- 运行时统一按插件主键定位插件，展示名仅做兼容输入
- `callback` 自动取插件可执行文件绝对路径，不允许外部覆盖
- 未传 `chatId` 时，自动生成 `key-uuid` 形式的 CHAT_ID
- 插件不存在、Agent 不存在、模型未注册、`meta` 非法 JSON、缺少必填参数时都会直接返回失败原因
- HTTP 接口与 CLI 复用同一套 `UpsertPluginConfig` 逻辑

## 完成情况

- `POST /api/plugins/config` 已实现
- `proxy plugins config` 已实现
- 已补充自动化测试，覆盖创建、更新、默认值与失败场景
- `proxy` 主手册已同步更新
