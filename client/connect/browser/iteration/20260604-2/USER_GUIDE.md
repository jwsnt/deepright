# 20260604-2 使用手册

## 目标

本次迭代补齐两件事：

- `browser start` 在原有生命周期前先预热 Playwright driver
- `browser fetch / browser store / browser start` 重新接入同一套 `cookie_path` 文件校验

## Playwright driver 预热

执行：

```bash
./browser start
```

现在会先检查当前插件同目录下的 `playwright/driver`：

- 如果目录已经完整，直接进入原有 `start` 流程
- 如果目录缺失或内容不完整，会尝试安装当前系统对应的 Playwright driver 到 `./playwright/driver`
- 这一步无论成功、失败还是超时，都只会记录到 `browser.log`，不会阻塞后续原有 `start` 生命周期
- Browser 的构建产物不再预打包 `playwright/driver`

## Cookie 文件链路

`cookie_path` 现在重新生效，支持两种来源：

- 命令行 `--cookie_path`
- Browser 插件配置 `meta.cookie_path`

规则：

- 显式命令行优先
- 相对路径按 Browser 当前运行根目录解析
- `store` 校验路径可写；文件缺失时自动创建为 `[]`
- `fetch` 校验路径可读，并输出该 Cookie 文件内容
- 未配置 `cookie_path` 时，`fetch` 返回 `[]`，`store` 返回 `OK`

示例：

```bash
./browser store --cookie_path ./cookies.json
./browser fetch --cookie_path ./cookies.json
```

## start 的 Cookie 校验

当 `cookie_path` 已配置时：

```bash
./browser start
```

会在继续原有 daemon 生命周期之前，先按同一路径执行一次 `store + fetch` 校验：

- 任一步失败，当前 `start` 直接终止
- 失败原因会写入同目录 `browser.log`
- 校验通过后，才继续原有 `start`、WSL follow-up `init` 和 daemon 拉起逻辑
