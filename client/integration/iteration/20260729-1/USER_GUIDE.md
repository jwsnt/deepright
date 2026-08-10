# 应用安装检查服务使用手册

在主应用资源目录的 `config/config.json` 中配置 `install_app`，即可让 Integration 和 Proxy 检查当前运行环境是否缺少指定应用：

```json
"install_app": {
  "interval": 60,
  "content": "请安装 $namelist",
  "linux": ["python3", "node", "curl", "wget", "git", "npm"],
  "wsl": ["python3", "node", "curl", "wget", "git", "npm"],
  "mac": ["python3", "git", "npm"]
}
```

- `interval` 是正整数，单位为分钟；缺失或无效时按 `60` 分钟处理。
- `content` 是交给 Site 的会话请求模板。Site 会将其中每一处 `$namelist` 替换为缺失应用名称，名称之间用 `、` 分隔；缺失或空白时使用 `请安装 $namelist`。
- `linux` 用于普通 Linux，`wsl` 用于 Windows 和 WSL，`mac` 用于 macOS。每个列表只在相应平台生效；空白名称和重复名称会被忽略。
- WSL 只认 WSL 内可直接执行的命令；Windows 宿主机的 `.exe` 或 `/mnt/c` 中的软件不会让应用被误判为已安装。
- 此对象是应用清单的唯一来源：macOS、Linux 和 Windows／WSL 都不支持 `--install_app`；Integration 或 Proxy 启动、重启时不会写回或覆盖它。

`GET /install_app` 继续返回兼容旧客户端的缺失应用 JSON 字符串数组，并使用既有短期缓存。新页面使用 `GET /install_app?details=1`，其响应为：

```json
{
  "apps": ["node", "npm"],
  "interval": 60,
  "content": "请安装 $namelist"
}
```

详情响应不缓存，并在每次扫描时重新判断可执行文件是否已存在。因此用户安装应用后，下一次按配置周期的扫描会反映该变化。接口仅报告缺失项并提供模板，不会执行安装命令。
