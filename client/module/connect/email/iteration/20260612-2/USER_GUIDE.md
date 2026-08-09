# Email 迭代手册（20260520-1）

## 本次变更

本次迭代调整了邮件插件 `start` 后的 POP3 连接策略：

- 启动后只建立一次 POP3 连接
- 后续轮询优先复用这条已认证连接
- 仅在连接断开、探活失败或关键邮箱配置变化时才重新建立 POP3 连接
- POP3 扫描期间如果服务端断链并返回 `EOF`，插件会把它视为可恢复断链，自动重连后继续当前轮扫描
- POP3 域名默认解析结果不可用、TLS 握手超时或单线路由抖动时，插件会自动回退到备用解析结果继续建连
- POP3 建连、TLS 握手和首包读取都受明确超时控制，失败会写结构化错误日志而不是静默卡死
- 发生 `EOF` 重连时不会提前推进 `email.state.json` 时间线，只有邮件真正读取进入后续处理后才会更新状态
- 不改变既有的邮件解析、`add-request` 推送、SMTP 发送和日志行为

## 当前行为

1. `../plugins/email start` 启动后，会先读取 `connect meta-get --key email` 的配置
2. 首次扫描前会建立一条 POP3 长连接并完成认证
3. 后续每个轮询周期都会优先对已有连接做探活，并复用该连接继续执行 `UIDL` / `RETR`
4. POP3 建连会先尝试系统 DNS 解析结果；若默认解析不可用、TLS 握手超时或首包读取超时，则会按备用解析结果继续尝试建连
5. 如果 POP3 连接已经失效、服务端主动断开，或 `email` / `email_pop3` / `email_password` 等关键配置发生变化，则会丢弃旧连接并重新登录
6. 如果断链发生在 `UIDL`、`RETR` 等扫描中间步骤，插件会在同一轮内自动重连，并从中断位置继续读取未完成的邮件
7. 如果个别邮件 MIME 结构异常，插件会退化为“原始报文兜底读取”，避免单封坏邮件阻塞后续收件
8. 单次扫描失败不会阻塞后续轮询；网络恢复后会自动继续收信，无需人工重启
9. 邮件拉取后的准入、去重、时间线持久化、附件下载和 `connect add-request` 推送逻辑保持不变
10. SMTP 回复邮件的行为保持不变

## 配置示例

```json
{
  "email": "demo@example.com",
  "email_pop3": "pop.example.com:995",
  "email_smtp": "smtp.example.com:465",
  "email_password": "secret",
  "email_whitelist": "",
  "email_pop3_interval": "300"
}
```

说明：

- `email_pop3_interval` 控制轮询间隔
- 轮询间隔变化不会强制重建 POP3 连接，只会影响下次扫描的触发时间
- `email`、`email_pop3`、`email_password` 变化会触发重连
- `email_pop3` 为域名时，系统 DNS 与备用解析结果都会参与建连容错

## 验证方式

先创建或更新 Meta：

```bash
./integration connect meta-create \
  --name email \
  --meta '{"email":"'$EMAIL'","email_pop3":"'$EMAIL_POP3'","email_smtp":"'$EMAIL_SMTP'","email_password":"'$EMAIL_PASSWORD'","email_whitelist":"","email_pop3_interval":"300"}'
```

启动邮件插件：

```bash
../plugins/email start --connect-bin ../integration/integration
```

验证预期：

- 启动时 POP3 仅登录一次
- 正常轮询时持续复用已有 POP3 连接，而不是每轮重新登录
- 手动断开 POP3 连接后，下一个轮询周期会自动重连
- 如果默认 DNS 解析结果不可用、TLS 握手超时或首包读取超时，插件会自动尝试备用解析结果继续建连
- 如果断链发生在当前轮扫描中途，插件会在当前轮内自动重连并继续读取剩余邮件
- `EOF` 或单封异常邮件不会导致 `email.state.json` 被提前推进到未成功读取的位置
- 单次扫描失败会记录结构化错误日志，例如 `stage=scan err=context deadline exceeded`，但不会阻塞下一轮自动重试
- 修改 `email_password` 或 `email_pop3` 后，下一个轮询周期会按新配置重新建连

## 说明

- 本次迭代手册对应当前目录下的 [REQUIREMENT.md](../20260520-1/REQUIREMENT.md)
- 本次仅补充手册说明，不修改需求文档本身
