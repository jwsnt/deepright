# Integration 迭代 20260807-1：多模态服务商引用

模型配置的 `__model_multi_input` 和 `__model_multi_output` 现在可保存为 `@服务商名称`，例如 `@deepright`。

- Proxy 会原样将该值写入请求 metadata。
- 普通对话、CLI/GET、飞书、邮件和备忘录任务使用同一份服务商配置。
- 删除服务商时，所有指向它的多模态输入和输出引用都会自动清空。
