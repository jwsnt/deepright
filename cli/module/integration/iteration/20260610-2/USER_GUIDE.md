# Integration 迭代手册（20260610-2）

## 本次更新

- `GET /api/plugins/meta` 中每个插件的 `param` 已统一改为对象数组格式
- 顶层 CLI `./integration list-plugins` 也同步输出相同结构
- 新格式示例：`[{"appId":""},{"appSecret":""}]`
- 每一项对象的 key 是参数名，value 是占位提示；未提供提示时返回空字符串
- `integration` 主手册中的接口示例和 CLI 示例已同步切换到新结构

## HTTP

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
      "key": "feishu",
      "name": "飞书",
      "param": [
        {
          "appId": ""
        },
        {
          "appSecret": ""
        }
      ],
      "meta": {
        "appId": "cli-app",
        "appSecret": "cli-secret"
      }
    }
  ]
}
```

说明：

- 只支持 `GET`
- `param` 不再返回纯字符串数组
- 未配置参数值时，`meta` 仍返回空对象 `{}`

## CLI

命令：

```bash
./integration list-plugins
```

说明：

- `param` 输出结构与 `/api/plugins/meta` 一致
- 插件如果没有参数，会返回空数组 `[]`

## 同步结果

- 已确认 `integration` 侧 `/api/plugins/meta` 与 `./integration list-plugins` 使用新 `param` 结构
- `integration/USER_GUIDE.md` 已同步更新
- 本迭代手册对应当前目录下的 `REQUIREMENT.md`
