# 20260527-5 使用手册

## 目标

本次迭代为设置面板补充新的模型配置项 `xiaomi`。

## 新增模型

- `xiaomi`
  - `__url=https://api.xiaomimimo.com/v1/chat/completions`
  - `__model=mimo-v2-flash`
  - `__model_fast=mimo-v2-flash`
  - `__model_thinking=mimo-v2.5-pro`
  - `__model_multi_input=mimo-v2.5`
  - 不支持 `__model_multi_output`

## 页面行为

- 设置里的模型下拉框现在可以直接选择 `xiaomi`
- 选中 `xiaomi` 后，会自动回填上述默认客户化配置
- `xiaomi` 支持客户化配置展开、重置、清空和保存
- `xiaomi` 的多模态输出字段保持禁用态，不会被自动回填为无效值

## 兼容性

- 现有模型保存、删除、本地回显、重新打开设置后的默认值补全逻辑保持不变
- `xiaomi` 会自动参与已配置模型列表、模型选择入口和蜂群模型可用性判断
