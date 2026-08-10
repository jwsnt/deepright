# 迭代 20260530-1：`/api/deviceId` 收口

本次迭代将 `deviceId` 查询能力收口到 `integration` HTTP 服务：

- 新增 `GET /api/deviceId`
- 返回当前共享 Agent 元数据中的 `deviceId`
- 返回格式为：

```json
{
  "status": 0,
  "deviceId": "your-device-id"
}
```

说明：

- `deviceId` 的来源与 Agent 模块共享元数据保持一致
- Site 设置页的复制 `deviceId` 按钮可以直接依赖这个接口
- 该接口与 `/api/agentId` 一样，属于前端站点常用的基础只读接口
