# 20260610-1 USER_GUIDE

本次迭代只调整 `email param` 的固定返回值，用于让 Integration/Connect 在展示插件参数时直接看到字段说明，而不是只拿到字段名列表。

## 变更内容

执行：

```bash
./email param
```

固定返回：

```json
[{
  "email": "邮箱地址，如hello_world@gmail.com",
  "email_pop3": "邮箱的pop3地址，如pop.gmail.com",
  "email_smtp": "邮箱的smtp地址，如smtp.gmail.com",
  "email_password": "邮箱的密码",
  "email_whitelist": "以逗号分隔的收件人白名单，如a@gmail.com,b@gmail.com。",
  "email_pop3_interval": "每次扫描待处理邮件的间隔秒数，默认300"
}]
```

## 使用说明

- `param` 返回的是“字段说明示例”，不是运行时配置值
- `meta-create` / `meta-update` 里的 `--meta` 仍然使用同名字段传真实值
- `--meta` 的字段集合需要和 `param` 返回对象里的 key 保持一致

示例：

```bash
./integration connect meta-create \
  --name email \
  --meta '{"email":"'$EMAIL'","email_pop3":"'$EMAIL_POP3'","email_smtp":"'$EMAIL_SMTP'","email_password":"'$EMAIL_PASSWORD'","email_whitelist":"","email_pop3_interval":"300"}'
```
