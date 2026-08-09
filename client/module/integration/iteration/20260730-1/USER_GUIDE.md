# 迷你应用运行配置使用手册

主应用资源目录的 `config/config.json` 可配置 `miniapp`：

```json
"miniapp": {
  "build": "请使用 @__internal_cli 为 $name 的 $function 构建迷你应用",
  "function": "全部功能"
}
```

`build` 是发送给当前会话的构建请求模板。Site 将 `$name` 替换为用户输入的 CLI 名称或 Git 地址，将 `$function` 替换为用户输入的功能列表；多个功能由 Site 使用全角顿号 `、` 连接。用户没有填写有效功能时，Site 使用 `function` 的非空字符串。

`GET /api/runtime_config` 会在成功响应的受控 `config` 对象中透传完整 `miniapp` 对象：

```json
{
  "status": 0,
  "config": {
    "miniapp": {
      "build": "请使用 @__internal_cli 为 $name 的 $function 构建迷你应用",
      "function": "全部功能"
    }
  }
}
```

接口只读取主应用资源目录的 `config/config.json`，不读取 Agent 工作目录中的同名文件，也不会补写或修改配置。`miniapp` 缺失或格式无效时，接口仍按受控白名单规则返回；Site 在发送前负责显示配置错误。Integration 不执行 CLI、克隆 Git 仓库、构建应用或替换模板变量，且 `provider`、模型密钥和其它未白名单配置不会暴露给浏览器。
