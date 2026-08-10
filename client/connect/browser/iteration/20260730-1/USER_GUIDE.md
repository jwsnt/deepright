# 20260730-1 USER_GUIDE

## WSL profile 隔离规则

本次变更仅作用于 Windows WSL / WSL2。macOS、Linux 与 Windows 原生环境的系统 Chrome profile 复制规则不变。

### browser start / stop

在 WSL 下，`browser start` 不会：

- 枚举、结束或等待 Windows 宿主机的 Chrome 进程
- 读取或复制 `%LOCALAPPDATA%\Google\Chrome\User Data`
- 创建、删除、刷新或使用 `C:\ProgramData\deepright\chrome_def`
- 删除任何 Chrome profile 的 `Singleton*`、`DevToolsActivePort`、`*.lock` 或 `*-journal` 文件

`browser stop` 也不会删除 `chrome_def`、`chrome_*` profile 或其内部锁/运行态文件。

## Browser 运行目录

Browser 不再要求静态 `config/config.json` 保存启动期派生的 `app-dir`。当配置中没有兼容旧版本的 `app-dir` 或 `app` 时，Browser 与 Integration 使用相同的固定运行目录：

- macOS：`~/Library/Containers/cn.deepright.integration/Data/Library/Application Support/deepright`
- WSL：`~/deepright`

因此，macOS `.app` 或 WSL 发布包中的静态 `config/config.json` 缺少这些字段时，`browser start` 仍会使用对应目录下的 `plugins`。

### instance create / init

WSL 下新建实例仍使用：

```text
C:\ProgramData\deepright\chrome_<随机后缀>
```

该目录会以空目录直接传入 Chrome 的 `--user-data-dir`。不会从系统 Chrome、`chrome_def` 或其他 profile 复制 Cookies、登录态、扩展、配置或缓存。

已有实例 profile 在重启时会继续复用；插件不会为重启而删除其中的锁或运行态文件。
