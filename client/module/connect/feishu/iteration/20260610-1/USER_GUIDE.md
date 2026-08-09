# 20260610-1 USER_GUIDE

本次迭代只调整 `feishu param` 的固定返回值，用于让 Integration/Connect 在展示插件参数时直接看到字段说明，而不是只拿到字段名列表。

## 变更内容

执行：

```bash
./feishu param
```

固定返回：

```json
[{
  "appId": "飞书开放平台（https://open.feishu.cn/app）中应用凭证的App ID ",
  "appSecret": "App Secret"
}]
```

## 使用说明

- `param` 返回的是“字段说明示例”，不是运行时配置值
- `meta-create` / `meta-update` 里的 `--meta` 仍然使用同名字段传真实值
- `--meta` 的字段集合需要和 `param` 返回对象里的 key 保持一致

示例：

```bash
./integration connect meta-create \
  --key feishu \
  --meta '{"appId":"'$FEISHU_APP_ID'","appSecret":"'$FEISHU_APP_SECRET'"}' \
  --callback ignored \
  --agent a \
  --model deepseek
```
