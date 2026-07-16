# 20260716-1 USER_GUIDE

## browser instance init 超时配置

本次规则适用于 macOS、Windows、WSL / WSL2 和 Linux：

```json
{
  "browser": {
    "init_timeout": 300
  }
}
```

- 配置文件为 integration 运行目录的 `config/config.json`
- `browser.init_timeout` 单位为秒，必须是正整数
- 未配置 `browser.init_timeout` 时使用默认值 `300` 秒
- 不允许通过 `browser instance init` 的命令行参数覆盖该值

## 前置条件与配置定位

必须先成功执行：

```bash
./browser start --connect-bin /path/to/integration
```

`start` 成功后会在 Browser 同目录写入 `browser_runtime.json`。`instance init` 只使用其中记录的 integration 路径定位 `config/config.json`。

- 未执行或未成功执行 `start` 时，`init` 会立即提示先执行 `browser start`
- 不会从 Browser 二进制目录、当前工作目录或其他猜测路径读取配置
- `config/config.json` 缺失、JSON非法，或 `browser.init_timeout` 不是正整数时，`init` 立即失败
- 配置在关闭旧CDP或修改实例状态前校验；配置错误不会关闭旧浏览器

## 超时范围与清理

`browser.init_timeout` 是整条 `browser instance init` 的总时限，涵盖：旧实例关闭、profile准备和复制、Chrome启动、CDP就绪检测。

超时后会终止本次新启动的 Chrome 并清理本次运行状态；实例profile（包括 WSL 的 `chrome_xxx`）会保留，避免丢失登录态和配置。

## 使用示例

```bash
./browser start --connect-bin /path/to/integration
./browser instance init --agentId agent-a --chatId chat-001
```

`init` 在新的有头 Chrome 和 CDP 就绪后返回。通过集成页面初始化时，页面只会在启动成功后显示“完成”按钮；点击该按钮会执行 `instance shutdown`。
