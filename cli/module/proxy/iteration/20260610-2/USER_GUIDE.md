# Proxy 迭代手册（20260610-2）

## 本次更新

- `GET /api/plugins/meta` 中每个插件的 `param` 已统一改为对象数组格式
- 新格式示例：`[{"appId":""},{"appSecret":""}]`
- 每一项对象的 key 是参数名，value 是占位提示；未提供提示时返回空字符串
- `proxy` 主手册中的接口示例已同步切换到新结构

## HTTP 用法

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
- 插件如果没有参数，会返回空数组 `[]`

## 同步结果

- 已确认 `proxy` 侧 `/api/plugins/meta` 实现和测试断言使用新 `param` 结构
- `proxy/USER_GUIDE.md` 已同步更新
- 本迭代手册对应当前目录下的 `REQUIREMENT.md`
