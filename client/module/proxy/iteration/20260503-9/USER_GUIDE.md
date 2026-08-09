# 20260503-9 使用手册

## 目标

本次迭代为 `proxy` 新增了插件元数据 HTTP 接口：

- 新增 `GET /api/plugins/meta`
- 返回值由 `connect list-plugins` 与 `connect meta-list` 合并生成
- `integration` 已同步暴露同一路径

## 接口说明

请求：

```text
GET /api/plugins/meta
```

响应示例：

```json
{
  "status": 0,
  "data": [
    {
      "name": "飞书",
      "param": ["appId", "appSecret"],
      "meta": {
        "appId": "cli-app",
        "appSecret": "cli-secret"
      }
    }
  ]
}
```

失败示例：

```json
{
  "status": 1,
  "content": "..."
}
```

## 行为规则

- 只支持 `GET`
- 底层直接复用 `connectsvc.ListPluginMeta`
- 插件发现规则与 `./connect list-plugins` 保持一致
- 已配置参数来自 `./connect meta-list`
- 默认扫描当前工作目录上一级的 `../plugins`
- 只读取当前层可执行文件，不扫描子目录
- 每个插件会执行两次：`name` 与 `param`
- 插件未配置时，`meta` 返回空对象 `{}`

## 同步结果

- `connect` 主手册已补充说明：`/api/plugins/meta` 会合并 `list-plugins` 与 `meta-list`
- `feishu` 文档已补充说明：会通过该接口暴露为可用插件
- `proxy` 主手册已补充接口说明
- `integration` 主手册已补充接口说明
