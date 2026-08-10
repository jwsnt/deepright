本轮迭代把 `GET /install_app` 的配置来源升级为主应用 `config/config.json` 的按操作系统结构，同时保留 `--install_app` 作为额外追加项。

适用规则：

- Linux 读取 `install_app.linux`
- macOS 读取 `install_app.mac`
- Windows 和 WSL 读取 `install_app.wsl`
- `--install_app` 依旧使用逗号分隔字符串，并与当前系统配置、自动探测结果统一去重合并
- 每个 `install_app` 元素都表示一个本地应用名；当前系统如果已安装该应用，就不会出现在 `/install_app` 返回中
- 应用安装状态会缓存 5 分钟

`config/config.json` 示例：

```json
{
  "install_app": {
    "linux": ["node", "python"],
    "wsl": ["node", "python", "docker"],
    "mac": ["node", "python", "xcode-select"]
  }
}
```

示例：

```bash
./proxy serve --agent-dir ./agents --install_app git,python3
curl http://127.0.0.1:8080/install_app
```

接口返回会合并：

- 当前机器自动探测缺失的 `git`、`python3`
- `config/config.json` 中当前操作系统对应的数组
- `--install_app` 传入的额外条目
