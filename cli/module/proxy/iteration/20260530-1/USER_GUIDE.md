# 迭代 20260530-1：`/api/deviceId`

本次迭代为 `proxy` 新增了一个只读接口：

- `GET /api/deviceId`
- 返回当前共享 Agent 元数据中的 `deviceId`
- 返回格式为：

```json
{
  "status": 0,
  "deviceId": "your-device-id"
}
```

说明：

- `deviceId` 的来源与 Agent 共享元数据保持一致
- 如果 `proxy` 无法读取 Agent 元数据，接口会返回 `status=1`
- Site 设置页里的复制 `deviceId` 按钮会优先使用这个接口
