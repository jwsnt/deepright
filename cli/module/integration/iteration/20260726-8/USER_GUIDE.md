# Integration 迭代手册（20260726-8）

## 发布文档中的本地端口

Integration 发布包中的 API、画布和设计文档会展示当前发布配置的本地服务地址。源码不写死端口，而是在以下三个文件中使用：

```text
http://localhost:#port
```

涉及文档：

- `config/app/API.md`
- `config/app/CANVAS.md`
- `config/app/DESIGN.md`

这里的 `#port` 是构建占位符，不是可直接访问的 URL，也不是 `config.json` 中应填写的值。

## 配置与构建

在构建前设置主应用配置 `config/config.json` 的顶层 `port`，例如：

```json
{
  "port": 57896
}
```

执行 `cli/module/build.sh` 时，构建脚本会复制 `config/` 到每一个目标发布目录，再把这三份**复制后的文档**中的 `http://localhost:#port` 替换为：

```text
http://localhost:57896
```

源码文档保持占位符，不会被构建过程改写。若修改了 `config/config.json.port`，需要重新执行构建，新的发布包文档才会显示新端口。

`port` 缺失或不是数字时，构建会失败并提示配置错误；不会默认回退到 `8080`，也不会产生仍含 `#port` 的发布文档。

## 运行时端口覆盖

`--port` 可以覆盖 Integration 进程实际监听的端口，例如：

```bash
./integration --port 18080
```

该覆盖不会回写 `config/config.json`，也不会修改已经打包的 Markdown 文档。因此，如果运行时使用了不同端口，应以实际启动参数为准；若希望发布文档也显示该端口，应更新 `config/config.json.port` 后重新构建。

## macOS 与 WSL2 发布位置

端口替换规则在共享构建流程中执行，语义没有平台差异：

| 发布形式 | 最终文档位置 |
|---|---|
| macOS `.app` | `DeepRight.app/Contents/Resources/config/app/` |
| Linux / Windows WSL2 安装载荷 | release 根目录的 `config/app/`，安装后随载荷复制到 WSL 工作目录 |

macOS 只是在替换后将 `config/` 复制进 App Resources 并参与签名；WSL2 只是在替换后的 Linux release 上追加安装器、rootfs 与沙盒帮助程序。两者都不会再次替换、重置或改写文档端口。
