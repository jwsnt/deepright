# 20260504-2 User Guide

## 目标

本次迭代为 `feishu` CLI 增加两个固定输出命令，方便外部模块在未启动长连接时也能读取插件展示信息和 `meta` 参数结构：

- `param` 固定返回 `["appId","appSecret"]`
- `name` 固定返回 `{"key":"feishu","name":"飞书"}`

## 使用方式

在 `/path/to/deepright/cli/module/connect` 目录，推荐先把独立二进制编译到 `../plugins/feishu`：

```bash
go build -o ../plugins/feishu ./feishu
```

之后即可直接调用：

```bash
../plugins/feishu param
../plugins/feishu name
```

也可以通过顶层 `connect` 子命令调用：

```bash
./connect feishu param
./connect feishu name
```

## 固定返回值

```bash
../plugins/feishu param
```

```json
[
  "appId",
  "appSecret"
]
```

```bash
../plugins/feishu name
```

```json
{"key":"feishu","name":"飞书"}
```
