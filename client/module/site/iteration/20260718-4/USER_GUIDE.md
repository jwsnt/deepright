# Site 迭代手册（20260718-4）

## 本次更新

- 在设置中新增模型并选择支持的服务商后，密钥输入框右侧会显示 `CURL` 小图标；若同时有复制入口，复制图标会自动左移，CURL 位于最右侧。
- 该入口支持 `gemini`、`openai`、`kimi`、`bigmodel`、`xiaomi`、`deepseek`、`anthropic` 和 `minimax`。已保存模型、`seedream` 与 `deepright` 不显示该入口。
- 点击入口会临时收起设置内容并打开 CURL 导入卡片。取消导入只会关闭该卡片，不会丢失设置中已填写但尚未保存的内容。
- 解析成功后，URL、API Key 和模型会回填至当前新增模型。模型名称会同步填写基础模型、快速响应、深度思考以及该服务商支持的多模态模型字段。

## 使用说明

1. 打开设置，在“模型与密钥”区域点击“添加模型”。
2. 选择一个支持 CURL 导入的服务商。
3. 点击密钥输入框右侧的 `CURL` 图标，粘贴该服务商的 CURL 请求命令。
4. 点击“解析”。成功后返回模型配置，按需要核对或修改回填内容。
5. 点击设置底部的“保存”后，新增模型配置才会持久化。

## 协议与限制

- Gemini 从请求 URL 的 `models/{model}`、URL 查询参数 `key` 或 `x-goog-api-key` 请求头读取信息；密钥会写入密钥字段，不会保留在 URL 字段中。
- OpenAI 兼容服务从 JSON 请求体的 `model` 和 `Authorization: Bearer ...` 请求头读取信息，并兼容 `api-key`、`x-api-key` 请求头。
- Anthropic 与 MiniMax 从 JSON 请求体的 `model` 和 `x-api-key` 请求头读取信息，也可使用 Bearer 请求头作为回退。
- 导入仅在浏览器内解析文本，不会执行 CURL 命令或访问命令中的 URL。
- 缺少 URL、模型或 API Key，或请求体不是可解析 JSON 时，会提示错误并保留原模型配置不变。
- 如果命令使用 `$OPENAI_API_KEY` 这类 shell 变量，需要先替换为实际值再解析；页面不会读取本机 shell 环境变量。
- Vertex CURL 请求不在支持范围内。
