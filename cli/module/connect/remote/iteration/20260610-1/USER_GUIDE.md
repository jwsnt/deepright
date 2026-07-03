# Remote 迭代 20260610-1 使用手册

## 变更说明

本次迭代只调整 `remote param` 的固定返回值。

`param` 不再返回纯字符串数组，而是返回一个带字段说明的对象数组，便于容器或其他模块直接展示参数用途。

## 命令用法

```bash
./remote param
```

返回值固定为：

```json
[{"exec_timeout":"选填。SSH执行超时。","scp_timeout":"选填。SCP执行超时。"}]
```

## 字段说明

- `exec_timeout`：选填。说明 SSH 执行命令时的超时参数。
- `scp_timeout`：选填。说明 SCP 传输时的超时参数。

## 行为说明

- `param` 返回值固定，不依赖运行时环境、数据库或当前会话。
- 返回结构继续保持数组，兼容插件元数据采集侧的通用处理方式。
- 当前数组固定只返回一个对象，集中描述 Remote 暴露的两个可选参数。

## 与主手册关系

- 主手册：[../../USER_GUIDE.md](../../USER_GUIDE.md)
- 主需求：[../../REQUIREMENT.md](../../REQUIREMENT.md)
- 当前迭代需求：[REQUIREMENT.md](REQUIREMENT.md)
