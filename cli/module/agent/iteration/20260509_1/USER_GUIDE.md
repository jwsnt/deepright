# Agent 迭代 20260509_1 使用手册

## 变更说明

Agent 顶层元数据新增 `plugins` 字段，用于返回当前应用已配置且已启动插件的 `key` 列表。

## 输出示例

```json
{
  "timezone": "Asia/Shanghai",
  "deviceId": "device-001",
  "terminal": "/bin/zsh",
  "plugins": ["feishu", "email"],
  "git": "/usr/bin/git",
  "gateway": "aa:bb:cc:dd:ee:ff",
  "sys": "Darwin 24.0.0",
  "app": "/path/to/agent-scanner",
  "agents": []
}
```

## 获取规则

- 固定优先复用当前应用二进制的 `list-meta`
- 从 `list-meta` 返回结果中读取每个已配置插件的 `key`
- 再根据插件运行目录中的 `<plugin-key>.pid` 判断对应进程是否仍存活
- 返回顺序按插件 `key` 排序
- 若没有任何同时满足“已配置且已启动”的插件，则 JSON 中不添加 `plugins`

## 子模块调用

```go
data, err := GetAgentOutput("/path/to/agents", "", 120*time.Second)
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(data))
```

返回结果中的顶层 `plugins` 可直接供 integration 等上层模块使用。
