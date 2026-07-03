# 20260603-1 使用手册

## 目标

本次迭代收敛 `browser` 在 Chrome 可执行文件路径上的配置来源。

适用范围：

- `browser create`
- `browser instance create|get|list|shutdown|destroy|restart`
- `browser` 代理的 Playwright 命令在自动创建受管实例时的实例创建链路

## Chrome 路径优先级

所有上述链路统一按以下优先级解析 Chrome 可执行文件路径：

1. `integration connect meta-get --key browser` 返回的 `meta.chrome`
2. 命令行显式传入的 `--chrome`
3. WSL 默认路径 `/mnt/c/Program Files/Google/Chrome/Application/chrome.exe`
4. 当前系统自动探测到的 Chrome/Chromium 路径

说明：

- 当 `meta.chrome` 缺失、为空字符串，或 `meta-get` 本身执行失败时，会自动回退到后续优先级
- 当 `meta.chrome` 存在但路径非法或文件不存在时，会直接报错，而不是静默跳过
- WSL 的判断方式与现有实现保持一致：通过 `uname -a` 检查是否包含 `WSL` 或 `Microsoft`

## 常见示例

```bash
./browser create --agentId agent-a --chatId chat-001 --connect-bin /path/to/integration
./browser instance create --agentId agent-a --chatId chat-001 --connect-bin /path/to/integration
./browser --session agent-a@chat-001 goto https://example.com --connect-bin /path/to/integration
```

在这些调用里，如果 Browser 插件配置里已经存在：

```json
{
  "key": "browser",
  "meta": {
    "chrome": "/custom/chrome/path"
  }
}
```

则会优先使用该路径启动或复用底层 Chrome/CDP 实例。

## 日志

- 插件日志仍固定写入 `browser` 同目录下的 `browser.log`
- User Data 复制、实例创建、实例复用、后台 daemon 与受管实例清理等行为都会继续写入这份日志
