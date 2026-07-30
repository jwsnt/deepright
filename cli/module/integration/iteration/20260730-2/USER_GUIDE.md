# macOS `.app` 静态配置与服务地址使用手册

macOS 的 `DeepRight.app` 内文件受代码签名保护。服务运行后不得修改 `Contents/MacOS`、`Contents/Resources` 或 `Contents/_CodeSignature`；修改会使 macOS 在下一次启动前判定签名无效。

## 配置与可变状态

| 内容 | 位置 | 用途 | 是否由服务写入 |
| --- | --- | --- | --- |
| 静态主应用配置 | `DeepRight.app/Contents/Resources/config/config.json` | 发布包默认启动参数、`miniapp`、`skills_git_install`、`install_app` 等静态功能配置 | 否 |
| 用户服务地址 | `~/Library/Containers/cn.deepright.integration/Data/Library/Application Support/deepright/data` 中的 SQLite `integration_persistent_settings` 表 | 用户主动保存的 `host` 覆盖项 | 是 |
| 其它运行状态 | 同一用户级运行目录中的 SQLite、日志、PID、状态文件等 | 每次运行产生的可变状态 | 是 |

这里的主应用 `config.json` 不等于 Agent 工作目录中的 `AgentId/config.json`；两者互不读取、互不覆盖。

## 启动与升级

macOS `.app` 每次启动都读取包内的 `Contents/Resources/config/config.json`，不会创建或使用运行目录中的 `config/config.json`。因此发布新版后，其中新增或更新的 `miniapp`、`skills_git_install`、`install_app` 等配置会立即生效，不需要删除旧配置文件。

服务地址的优先级是：

1. 显式 `--host` 参数；
2. SQLite 中由用户保存的 `host`；
3. 包内静态 `config.json` 的 `host`；
4. 内置默认值 `https://www.deepright.cn`。

`app`、数据库路径、PID、日志、端口、超时、缓存、重试和设备等启动期值会在每次启动时重新解析，不会写回 `config.json`。应用关闭时也不需要清理配置文件：运行时 `config.json` 根本不会被创建；PID 和浏览器复用标记等临时状态仍由既有运行目录机制清理。

如果旧版本曾留下用户运行目录中的 `config/config.json`，新版会忽略它，以避免旧完整快照遮蔽新版发布包配置。

## 服务地址持久化

`/api/host` 和 `integration host` 可查看或修改服务地址。地址必须是没有查询串或片段的绝对 `http` 或 `https` URL。

- `POST` / `PUT /api/host` 或 `integration host set --value <URL>`：立即切换当前地址，并把规范化后的地址写入 SQLite。
- `DELETE /api/host`、`POST /api/host?reset=true` 或 `integration host reset`：删除 SQLite 覆盖项，立即恢复包内静态 `config.json.host`；静态值不存在时使用内置默认值。
- SQLite 写入失败时，接口返回错误，当前地址保持不变。

所有写入都在用户级运行目录的 SQLite 内完成，绝不修改 `.app` 资源目录，因此不会破坏签名。

## WSL、Linux 与目录发布

WSL、Linux 和不在 `.app` 包内运行的 macOS 命令行二进制仍读取可执行文件同级的 `config/config.json` 作为静态默认配置。服务启动同样不会把派生参数写回该文件；用户主动设置的服务地址存放在相应共享 SQLite 中。

## `miniapp` 运行配置

主应用静态 `config/config.json` 可选配置：

```json
"miniapp": {
  "build": "请使用 @__internal_cli 为 $name 的 $function 构建迷你应用",
  "function": "全部功能"
}
```

`GET /api/runtime_config` 只在成功响应的受控 `config` 对象中透传完整 `miniapp` 对象，供 Site 组装当前会话的构建请求。Site 将 `$name` 替换为用户输入的 CLI 名称或 Git 地址，将 `$function` 替换为功能描述；功能为空时使用 `function`。

接口只读取当前发布包（或目录发布）的静态主应用配置，不读取 Agent 工作目录或旧运行时配置，不修改任何配置，也不执行 CLI、克隆仓库或构建应用。`provider`、模型密钥和其它未列入白名单的配置不会暴露给浏览器。
