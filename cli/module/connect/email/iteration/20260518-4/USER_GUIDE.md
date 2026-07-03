# Email 迭代手册（20260518-4）

## 本次变更

本次迭代补齐了邮件插件在轮询参数和日志落盘上的兼容能力：

- `param` 命令新增 `email_pop3_interval`
- POP3 轮询间隔优先读取 `email_pop3_interval`
- 为兼容旧配置，仍回退读取 `email_pop3_seconds`
- POP3 / SMTP 的明细日志与失败日志统一写入 `email.log`
- `email.log` / `email.runtime.log` 新增按大小自动分卷，单文件最大 `10MB`，最多保留 `4` 个历史分卷

## 当前行为

1. `../plugins/email param` 固定返回：

```json
["email","email_pop3","email_smtp","email_password","email_whitelist","email_pop3_interval"]
```

2. 邮件扫描进程启动后，会从 `connect meta-get --key email` 读取配置
3. POP3 轮询秒数读取顺序为：
   - `email_pop3_interval`
   - `email_pop3_seconds`
   - 默认值 `300`
4. POP3 拉取、SMTP 发送、失败重试和异常信息都会进入 `email.log`
5. 运行诊断日志会同时复制到 `email.runtime.log`
6. `email.log` 和 `email.runtime.log` 都会在达到 `10MB` 后自动轮转，最多保留 `4` 个历史分卷

## 配置示例

推荐使用当前字段：

```json
{
  "email": "demo@example.com",
  "email_pop3": "pop.example.com:995",
  "email_smtp": "smtp.example.com:465",
  "email_password": "secret",
  "email_whitelist": "alice@example.com,bob@example.com",
  "email_pop3_interval": "300"
}
```

兼容旧字段时，以下配置也可以继续工作：

```json
{
  "email": "demo@example.com",
  "email_pop3": "pop.example.com:995",
  "email_smtp": "smtp.example.com:465",
  "email_password": "secret",
  "email_whitelist": "",
  "email_pop3_seconds": "300"
}
```

## 验证方式

先确认参数集合：

```bash
../plugins/email param
```

再创建或更新 Meta：

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

- `param` 返回值中包含 `email_pop3_interval`
- 未配置轮询秒数时，默认每 `300` 秒扫描一次
- 仅保留旧字段 `email_pop3_seconds` 时，仍能按旧值扫描
- POP3 / SMTP 明细和错误信息会写入 `email.log`
- 当日志文件超过 `10MB` 时，会生成 `email.log.1` 到 `email.log.4`

## 说明

- 本次迭代手册对应当前目录下的 `REQUIREMENT.md`
- 本次仅更新手册，不改动需求文档本身
