# 20260731-1 USER_GUIDE

## Chrome profile 定期清理

在主应用的 `config/config.json` 中配置：

```json
{
  "browser": {
    "init_timeout": 300,
    "clear": 72,
    "scan": 2
  }
}
```

- `clear` 是 profile 过期时间（小时），`scan` 是扫描周期（小时）。
- Browser 插件成功执行 `start` 后，会在后台立即扫描一次，随后每 `scan` 小时扫描一次；不会延迟 `start` 返回。
- macOS 扫描固定 Agent 目录 `~/Library/Containers/cn.deepright.integration/Data/Library/Application Support/deepright/agent/<agent>/` 下名称为 `chrome_` 加非空后缀的目录，名称不区分大小写。
- WSL 仅扫描 Windows 宿主机 `C:\ProgramData\deepright\profiles\chats` 下的直接 Chat Profile 目录。
- 最后修改时间超过 `clear` 小时的目录会被删除；未过期目录、普通文件和符号链接不会删除。
- 原生 Linux 和 Windows 原生不执行这项清理。
- `clear`、`scan` 缺失或无效时，任务不会执行，原因记录到 Browser 同目录的 `browser.log`；插件仍会正常启动。
- `browser stop` 会结束该后台清理任务。
