# Proxy 迭代手册（20260517-2）

## 本次更新

- `/api/plugins/meta` 新增读取插件 `scope` 命令
- 插件的 `name`、`param`、`scope`、`command`、`help` 改为并发探测，缩短接口总耗时
- 该接口继续保持实时读取，不依赖旧缓存结果

## 当前行为

1. 请求 `GET /api/plugins/meta` 时，服务会实时扫描当前插件目录
2. 每个插件会并发执行 `name`、`param`、`scope`、`command`、`help`
3. 返回值中新增 `scope` 字段，表示插件支持的容器配置项
4. 如果插件没有实现 `scope` 命令，则默认返回 `["reuse","agent","provider","thinking"]`
5. 如果插件显式返回空数组 `[]`，则表示完全不支持容器配置

## 说明

- 本次迭代手册对应当前目录下的 `REQUIREMENT.md`
- 当前目录交付文件为：`REQUIREMENT.md`、`USER_GUIDE.md`
