# CLI-Get 迭代 20260509-2 使用手册

## 变更说明

本次迭代为 `cli/get` 与 `cli/pub` 的 Agent 元数据补充 `plugins` 字段说明，并同步主手册。

## `plugins` 字段规则

- `plugins` 位于 Agent 元数据顶层，与 `timezone`、`deviceId`、`terminal`、`gateway`、`sys`、`app`、`agents` 同级
- 字段值为字符串数组，数组项是插件 `key`，不是展示名
- 数据来源优先为当前应用二进制提供的已配置插件列表
- `cli-get` 先复用当前二进制的 `list-meta` 命令获取插件 key，再过滤出对应插件进程已启动的项，并按字典序写入 `plugins`
- 若没有任何同时满足“已配置且已启动”的插件，则不写入该字段

## 请求示例

```json
{
  "model": "",
  "messages": [
    {
      "role": "user",
      "content": ""
    }
  ],
  "metadata": {
    "timezone": "Asia/Shanghai",
    "deviceId": "device-001",
    "terminal": "/bin/zsh",
    "plugins": ["browser", "feishu"],
    "gateway": "aa:bb:cc:dd:ee:ff",
    "sys": "Darwin 24.5.0",
    "app": "/path/to/cli-get",
    "agents": []
  }
}
```

## 补充说明

- `cli/get` 与 `cli/pub` 使用同一份 Agent 元数据，因此 `plugins` 的行为完全一致
- 当前目录独立编译出的 `cli-get` 本身没有提供 `list-meta` CLI，因此单独运行时通常不会携带 `plugins`
- 当该能力被集成到支持 `list-meta` 的统一二进制中时，`plugins` 会随统一 CLI 的已配置插件列表自动探测运行状态后再上报
