# 20260715-1 USER_GUIDE

## 本次变更

- Browser 新增运行时文件 `browser_runtime.json`，放在 Browser 插件运行目录下
- `browser start` 是唯一允许接收 `--connect-bin` 的命令
- `browser start` 成功后才会写入或覆盖 `browser_runtime.json`
- 除 `browser start` 外，其余 Browser 命令传入 `--connect-bin` 都会立即报错
- `integration plugins exec` 和 `/api/plugins/exec` 在调用 Browser 时，也同步遵守这条规则；不会再给 `browser instance init`、`goto`、`eval` 等命令偷偷补 `--connect-bin`
- Browser 之后读取 integration/runtime 路径时，统一优先从 `browser_runtime.json` 读取
- `browser stop` 会 best-effort 删除 `browser_runtime.json`，删除失败只记日志，不影响 stop 返回

## browser_runtime.json

文件位置：

```text
<browser运行目录>/browser_runtime.json
```

文件内容示例：

```json
{
  "connectBin": "/Applications/DeepRight.app/Contents/MacOS/integration"
}
```

说明：

- 只有 `browser start --connect-bin ...` 成功后才会生成或覆盖这个文件
- `browser start` 失败时不会覆盖旧文件
- `browser instance shutdown` 不会删除这个文件
- `browser stop` 才会在停机流程结束后 best-effort 删除这个文件

## integration 转发约束

以前的错误现象：

```text
run plugin browser instance init failed: exit status 1: --connect-bin is only supported by `browser start`
```

现在的规则：

- `integration plugins start --key browser` 可以继续为 Browser 自动补 `--connect-bin`
- `integration plugins exec --key browser --command 'instance init'` 不再自动补 `--connect-bin`
- `/api/plugins/exec?key=browser&command=instance+init...` 也不再自动补 `--connect-bin`
- 这样 Browser 的非 `start` 命令会统一从 `browser_runtime.json` 读取 runtime 信息，而不是依赖调用方继续传参

## Chrome Profile 复制修复

`browser instance create/init` 在准备受管 Chrome Profile 时，会从系统 Chrome User Data 克隆登录态目录。

本次补充的跳过规则：

- `RunningChromeVersion`
- `SingletonLock`
- `SingletonCookie`
- `SingletonSocket`
- `DevToolsActivePort`

这些文件都属于纯运行态文件，不属于登录态。

修复后的效果：

- 重复点击初始化时，不会再因为这些 symlink / lock 文件已存在而直接失败
- `instance init` 的重试场景不再容易因为 `RunningChromeVersion: file exists` 变成 500

## 验证示例

```bash
./browser start --connect-bin /path/to/integration
./browser instance init --agentId agent-a --chatId chat-001
./browser --session agent-a@chat-001 goto https://example.com
./browser stop
```

预期：

- `start` 成功后生成 `browser_runtime.json`
- `instance init` 不需要再传 `--connect-bin`
- `goto` 不需要再传 `--connect-bin`
- `stop` 结束后 best-effort 删除 `browser_runtime.json`
