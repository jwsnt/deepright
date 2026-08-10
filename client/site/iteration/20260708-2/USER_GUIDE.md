# 20260708-2 用户手册

本次迭代为 Seedream 增加了特殊处理，目标是让它继续可配置，但不再作为普通模型直接出现在各类模型选择列表中，同时把相关技能上报统一收口。

## 模型配置

- `seedream` 仍然可以在设置里的 `模型与密钥` 中配置
- 可配置项仍然包括 `token`、`__url` 和 `__model_multi_output`
- 如果用户没有手动填写 `__url` 或 `__model_multi_output`，系统会继续按默认值补全

## 模型选择限制

- `seedream` 不会出现在居中会话输入框的模型选择列表中
- `seedream` 不会出现在右侧备忘录输入框的模型选择列表中
- `seedream` 不会出现在插件浮层的模型选择列表中
- 因此它不再作为普通聊天模型、备忘录模型或插件模型被直接选用

## 技能上报

- Seedream 对应能力统一改为内部技能 `__internal_seedream`
- 不再使用旧名字 `image-seedream`
- `__internal_seedream` 只有在 Seedream 已完成可用配置时才会上报
- 上报范围包括 `/api/skills`
- 上报范围包括前端 `@技能` 列表
- 上报范围包括转发 `/v1/chat/completions` 时的 `metadata.agents[].skills`
- 上报范围包括 `cli/get` 时的 `metadata.agents[].skills`

## 可用配置判定

- 判定来源按 `integration token --provider seedream`
- `token` 必须真实存在
- `__url` 与 `__model_multi_output` 可以来自显式配置，也可以来自系统默认值
- 只要三者最终都可用，`__internal_seedream` 才会被上报
- 如果任一条件不满足，系统会把 `__internal_seedream` 从上述所有上报链路中移除
