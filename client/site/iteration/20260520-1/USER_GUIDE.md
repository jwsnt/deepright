本次迭代为 Site 的“模型与密钥”设置补充了模型级客户化配置，并把多模态能力拆分为“多模态输入”和“多模态输出”两项。

### 客户化配置入口

- 每条模型记录会在删除按钮前显示客户化配置按钮
- 每次重新打开设置时，补充配置区默认都保持收起
- 点击后会在当前模型条目下展开补充配置区
- 再次点击会收起
- 同一时刻只会展开一条模型记录
- `deepright` 不显示客户化配置按钮
- 如果当前模型已经存在任意客户化配置值，例如 `__url`、`__model`、`__model_fast`、`__model_thinking`、`__model_multi_input`、`__model_multi_output` 中任一项被实际自定义过，小图标会持续闪动提示
- 展开后右下角会显示 `重置` 和 `清空` 两个按钮
- `重置` 用于把当前模型的 URL 和扩展模型字段恢复为默认值
- `清空` 用于清除当前模型的 URL 和全部扩展模型字段；保存后再次打开也不会自动回填默认客户化配置

### 可配置字段

展开后会显示：

- 模型 URL：`__url`
- 基础模型：`__model`
- 快速响应：`__model_fast`
- 深度思考：`__model_thinking`
- 多模态输入：`__model_multi_input`
- 多模态输出：`__model_multi_output`

其中多模态输入与多模态输出现在是两条独立配置链路，不再共用一个旧字段。

### 默认值与不可配置项

不同模型会自动带出默认值：

- `deepseek`
  - `__url=https://api.deepseek.com/chat/completions`
  - `__model=deepseek-v4-flash`
  - `__model_fast=deepseek-v4-flash`
  - `__model_thinking=deepseek-v4-pro`
  - 不支持配置 `__model_multi_input`、`__model_multi_output`
- `bigmodel`
  - `__url=https://open.bigmodel.cn/api/paas/v4/chat/completions`
  - `__model=glm-5.1`
  - `__model_fast=glm-4.7-flashx`
  - `__model_thinking=glm-5.1`
  - `__model_multi_input=glm-5v-turbo`
  - 不支持配置 `__model_multi_output`
- `gemini`
  - `__url=https://generativelanguage.googleapis.com/v1beta/models/#model:streamGenerateContent`
  - `__model=gemini-3.5-flash`
  - `__model_fast=gemini-3.5-flash`
  - `__model_thinking=gemini-3.1-pro-preview`
  - `__model_multi_input=gemini-3.5-flash`
  - `__model_multi_output=gemini-3.1-flash-image-preview`
- `openai`
  - `__url=https://api.openai.com/v1/chat/completions`
  - `__model=gpt-5.4`
  - `__model_fast=gpt-5.4`
  - `__model_thinking=gpt-5.4`
  - `__model_multi_input=gpt-5.4`
  - 不支持配置 `__model_multi_output`
- `anthropic`
  - `__url=https://dashscope-intl.aliyuncs.com/compatible-mode/v1`
  - `__model=claude-opus-4-6`
  - `__model_fast=claude-haiku-4-5-20251001`
  - `__model_thinking=claude-opus-4-6`
  - `__model_multi_input=claude-opus-4-6`
  - 不支持配置 `__model_multi_output`
- `kimi`
  - `__url=https://api.moonshot.cn/v1/chat/completions`
  - `__model=kimi-k2.6`
  - `__model_fast=kimi-k2-turbo-preview`
  - `__model_thinking=kimi-k2.6`
  - `__model_multi_input=kimi-k2.6`
  - 不支持配置 `__model_multi_output`
- `minimax`
  - `__url=https://api.minimaxi.com/anthropic/v1/messages`
  - `__model=MiniMax-M2.7`
  - `__model_fast=MiniMax-M2.7-highspeed`
  - `__model_thinking=MiniMax-M2.7`
  - 不支持配置 `__model_multi_input`、`__model_multi_output`
- `qwen`
  - `__url=https://dashscope-intl.aliyuncs.com/compatible-mode/v1`
  - `__model=qwen3.5-flash`
  - `__model_fast=qwen3.5-flash`
  - `__model_thinking=qwen3.6-plus`
  - `__model_multi_input=qwen3.5-flash`
  - `__model_multi_output=qwen-image-plus`

说明：

- 默认值会在切换模型时自动回填到输入框
- 不支持配置的字段会显示为禁用态，不会提交自定义值
- 保存时页面会把这些字段一并提交到 `/api/token`
- 如果点击删除的是已保存模型，页面会立即通过 `/api/config` 删除该模型的服务端持久化配置
