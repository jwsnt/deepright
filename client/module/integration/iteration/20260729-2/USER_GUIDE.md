# 快捷回复运行配置使用手册

主应用资源目录的 `config/config.json` 可配置可选字段 `shortcut`：

```json
"shortcut": ["好的", "同意", "执行"]
```

Integration 的 `GET /api/runtime_config` 会在成功响应的 `config` 对象中受控透传该字段，供 Site 展示快捷回复列表：

```json
{
  "status": 0,
  "config": {
    "shortcut": ["好的", "同意", "执行"]
  }
}
```

该接口只读取配置，不会补写、修改或持久化 `shortcut`。字段缺失、为空或格式不符合 Site 的列表规则时，接口仍正常返回；Site 会自行忽略无效项。此变更不会扩大运行配置响应的权限范围，`provider`、模型密钥和其它未白名单字段不会暴露给浏览器。
