# Integration 迭代 20260615-3 使用手册

## 变更说明

- `integration.app` 双击启动时，会先检查安装包内 `plugins/` 与运行时 `~/Library/Application Support/deepright/plugins/` 的插件二进制是否一致
- 只有插件 `MD5` 不一致时，才会更新运行时插件
- 如果检测到插件需要更新，但当前 `8080` 端口已被占用，则不会覆盖运行中的插件二进制，而是弹窗提示“有插件需要更新，请重启应用”
- 如果检测到插件需要更新，且当前未启动，则会先完成插件同步，再继续启动应用
- 插件更新使用“临时文件 + rename”的原子替换方式，避免运行中可执行文件被直接覆盖

## CLI

新增命令：

```bash
integration plugins sync-bundled --check
integration plugins sync-bundled
```

说明：

- `--check`：只检查安装包插件与运行时插件是否一致，不执行同步
- 不带 `--check`：同步所有 `MD5` 不一致的插件二进制

返回示例：

```json
{
  "status": 0,
  "data": {
    "bundledPluginDir": "/Applications/DeepRight.app/Contents/Resources/plugins",
    "runtimePluginDir": "/Users/demo/Library/Application Support/deepright/plugins",
    "needsUpdate": true,
    "pending": [
      {
        "name": "browser",
        "sourcePath": "/Applications/DeepRight.app/Contents/Resources/plugins/browser",
        "targetPath": "/Users/demo/Library/Application Support/deepright/plugins/browser",
        "sourceMD5": "xxx",
        "targetMD5": "yyy"
      }
    ],
    "updated": [],
    "checkOnly": true
  }
}
```

## 启动行为

- 双击 `integration.app` 时，如果本机已有运行中的 Integration，并且检测到插件需要更新：
  - 弹窗提醒重启应用
  - 不覆盖运行时插件
  - 仍会打开 `http://localhost:8080/site/#app`
- 双击 `integration.app` 时，如果当前未启动，并且检测到插件需要更新：
  - 先同步插件
  - 再启动 Integration
  - 最后打开 `http://localhost:8080/site/#app`
