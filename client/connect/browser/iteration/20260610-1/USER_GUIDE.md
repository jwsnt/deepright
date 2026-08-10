# 20260610-1 使用手册

## 目标

本次迭代收敛 Browser 插件对外元信息，`param` 命令不再返回简单参数名列表，而是固定返回带说明的参数定义。

## browser param

执行：

```bash
./browser param
```

固定返回：

```json
[{"headless":"选填。默认为true，使用无头浏览器静默访问，也可切换为false开启可视化访问"},{"chrome":"选填。Chrome浏览器地址，默认使用系统路径。"}]
```

说明：

- `headless` 表示插件默认浏览器是否无头启动
- `chrome` 表示 Chrome 浏览器可执行文件路径
- 返回顺序固定为 `headless`、`chrome`

## 日志

- 插件日志继续固定写在 Browser 同目录的 `browser.log`
- `fetch`、`store`、`start` 仍复用同一套 `cookie_path` 识别和校验链路
