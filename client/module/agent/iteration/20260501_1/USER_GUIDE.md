# Agent 迭代 20260501_1 使用手册

## 变更说明

Agent 顶层元数据新增 `git` 字段，用于返回当前设备已安装 git 可执行文件的绝对路径；如果未安装或系统无法获取，则返回空字符串。

## 输出示例

```json
{
  "timezone": "Asia/Shanghai",
  "deviceId": "device-001",
  "terminal": "/bin/zsh",
  "git": "/usr/bin/git",
  "gateway": "aa:bb:cc:dd:ee:ff",
  "sys": "Darwin 24.0.0",
  "app": "/path/to/agent-scanner",
  "agents": []
}
```

## 获取规则

- macOS、Linux：优先通过 `PATH` 查找，并回退使用 `command -v git`
- Windows：优先通过可执行文件后缀查找，并回退使用 `where git`
- 任一步骤获取失败时，`git` 返回空字符串

## 子模块调用

```go
data, err := GetAgentOutput("/path/to/agents", "", 120*time.Second)
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(data))
```

返回结果中的 `metadata.git` 可直接供 proxy、integration 等上层模块复用。
