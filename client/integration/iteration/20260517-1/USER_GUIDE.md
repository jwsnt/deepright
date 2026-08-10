# Integration 迭代手册（20260517-1）

## 本次更新

- `integration` 收口的 `/api/plugins/meta` 改为每次实时读取插件目录
- 插件元数据读取新增 `scope` 字段
- 插件 `name`、`param`、`scope`、`command`、`help` 使用共享实现并发探测

## 当前行为

1. 请求 `GET /api/plugins/meta` 时，`integration` 不再复用旧缓存插件结果
2. 每个插件会并发执行 `name`、`param`、`scope`、`command`、`help`
3. 返回值中的插件对象新增 `scope` 字段
4. 插件未实现 `scope` 命令时，默认返回 `["reuse","agent","provider","thinking"]`
5. 该行为与 `proxy` 复用同一份共享实现，避免多处维护

## 说明

- 本次迭代手册对应当前目录下的 `REQUIREMENT.md`
- 当前目录交付文件为：`REQUIREMENT.md`、`USER_GUIDE.md`
