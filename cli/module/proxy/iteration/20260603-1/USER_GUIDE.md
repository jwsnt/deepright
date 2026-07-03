# Proxy 迭代手册（20260603-1）

## 本次更新

- `GET /api/plugins/meta` 改为在 `proxy` 模块内直接扫描插件目录，不再依赖外部 `connect list-plugins`
- 当前只把以下文件视为插件候选：
  - 无后缀且可执行的程序
  - 后缀为 `.py`、`.js`、`.go` 的脚本文件
- 目录、隐藏文件和其他无关文件会被直接跳过
- 单个候选文件读取失败或执行 `name`、`param`、`scope`、`command`、`help` 失败时，只输出日志并跳过，不会让接口报错或崩溃

## 接口行为

请求：

```text
GET /api/plugins/meta
```

说明：

- 只支持 `GET`
- 每次请求都会实时扫描当前启动目录下的 `./plugins`
- 只扫描当前层，不递归子目录
- 返回结果仍会把“本次扫描到的插件列表”和当前已保存的 `meta` 配置合并后返回
- 如果插件未配置，则 `meta` 仍返回空对象 `{}`
- 如果某个坏文件混在 `plugins` 目录中，其他正常插件仍会继续返回

## 日志

- 跳过文件时会输出类似 `plugins meta skip <filename>: ...` 的日志
- 这类日志用于排查误放入 `plugins` 目录的压缩包、目录或异常脚本

## 同步结果

- `proxy/USER_GUIDE.md` 已同步补充本轮规则
- 本迭代手册对应当前目录下的 `REQUIREMENT.md`
