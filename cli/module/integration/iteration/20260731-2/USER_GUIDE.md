# 构建迷你应用参考文档使用手册

“构建迷你应用”浮层在功能列表下方提供一个默认收起的“参考文档（可选，仅 .md）”字段。每次构建只能指定一个 Markdown 文档：点击小型虚拟文件系统图标会展开字段和目录选择器，可在当前会话 Agent 的工作区中像左侧虚拟文件系统一样逐层展开目录并选择 `.md` 文件；目录仅用于浏览。也可先展开字段，再直接粘贴 `.md` 文件的绝对路径。非 `.md` 路径不会发送。

该选择器仅读取目录和文件名，不会上传、复制、修改或删除文件，也不会在浏览器中读取文件内容。若要引用工作区外的文件，可直接输入其绝对路径。取消、点击遮罩或按 `Esc` 会保留尚未提交的路径；构建请求发送成功后才会清空草稿。

主应用发布包的静态 `config/config.json` 可配置参考文档模板：

```json
"miniapp": {
  "build": "请使用 [SKILL:__internal_cli] 为 $name 的 $function 构建迷你应用 $reference",
  "reference": "（READ_ME.md: $reference）",
  "function": "全部功能",
  "recover": 30
}
```

确认构建时，Site 会重新读取当前发布包中的完整 `miniapp` 配置。它始终将 `build` 的 `$name` 和 `$function` 替换为表单内容；功能为空时使用 `function` 的值。填写了参考文档时，先用该绝对路径替换 `reference` 模板中的全部 `$reference`，再用结果替换 `build` 中全部 `$reference`。例如路径为 `/tmp/READ_ME.md` 时，默认模板会附加 `（READ_ME.md: /tmp/READ_ME.md）`。

未填写参考文档时，`build` 中的全部 `$reference` 会替换为空字符串；这时不要求配置 `reference`。填写路径后若 `reference` 缺失、不是字符串或为空，页面会显示配置错误并不会发送请求。同样，`build` 必须是包含 `$name`、`$function` 的非空字符串。

`GET /api/runtime_config` 继续只读地受控透传完整 `miniapp` 对象，不读取 Agent 工作目录配置、不修改配置，也不直接执行 CLI、读取参考文件或构建迷你应用。
