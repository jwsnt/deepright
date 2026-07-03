# 20260504-3 User Guide

## 目标

本次迭代为 `connect` 新增 `list-plugins` 命令，用来读取 `../plugins` 目录当前层所有插件二进制的展示信息。

每个插件需要支持：

- `<plugin> name`
- `<plugin> param`

`connect` 会把这两个命令的结果合并成统一 JSON 输出。

## 使用方式

在 `/path/to/deepright/cli/module/connect` 目录执行：

```bash
./connect list-plugins
```

如果希望调整缓存时长，可以显式传入：

```bash
./connect list-plugins --connect-cache 30000
```

## 返回格式

假设 `../plugins` 目录中有两个插件：

- `a name` 返回 `"IM"`，`a param` 返回 `["key"]`
- `b name` 返回 `"APP"`，`b param` 返回 `["token","ticket"]`

那么：

```bash
./connect list-plugins
```

会返回：

```json
[
  {
    "name": "IM",
    "param": [
      "key"
    ]
  },
  {
    "name": "APP",
    "param": [
      "token",
      "ticket"
    ]
  }
]
```

## 规则说明

- 只扫描 `../plugins` 当前层文件，子孙目录不会进入
- 仅处理二进制可执行文件
- 每次扫描会调用插件的 `name` 和 `param` 命令
- 扫描结果会缓存到本地文件
- 缓存时间由 `--connect-cache` 控制，默认 `10000` 毫秒
- 当 `--connect-cache` 小于等于 `0` 时，本次执行直接实时扫描
