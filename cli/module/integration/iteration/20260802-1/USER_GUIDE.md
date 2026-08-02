# 迷你应用调试运行配置使用手册

迷你应用调试使用主应用发布包中的 `config/config.json.miniapp.debug` 模板。默认配置如下：

```json
"miniapp": {
  "build": "请使用 [SKILL:__internal_cli] 为 $name 的 $function 构建迷你应用 $reference",
  "reference": " 参考READ_ME.md: $reference",
  "function": "全部功能",
  "debug": "请修复 $path，问题是 $reason",
  "recover": 30
}
```

用户确认调试时，Site 会重新读取运行时配置，并将模板内全部 `$path` 替换为所选 `app/` 目录 HTML 文件的绝对路径，将全部 `$reason` 替换为修复原因。模板必须是同时包含这两个变量的非空字符串；缺失、类型不正确或变量不完整时，页面会显示错误且不会发送请求。

`GET /api/runtime_config` 只读地透传完整受控 `miniapp` 对象。Integration 始终读取当前发布包的配置，不读取 Agent 工作目录配置，不写回配置、不执行修复命令，也不为调试新增任务接口。修复请求和附件继续走普通当前会话消息链路。

macOS 发布时，应同步 `DeepRight.app/Contents/Resources/config/config.json` 与 Site 页面资源；资源变更后重新签名并严格验证应用，避免运行中的页面读取到旧模板。
