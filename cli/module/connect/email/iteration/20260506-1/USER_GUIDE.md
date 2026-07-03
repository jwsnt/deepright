# 20260506-1 User Guide

## 目标

本次迭代为 `email` 插件补齐独立运行和 Integration 拉起场景下的使用说明，覆盖：

- 固定元信息输出：`param`、`name`
- 插件启动与停止：`start`、`stop`
- 通过 Integration 注册、拉起、停止和注销邮件插件
- 邮件消息的准入、去重、时间线与附件下载规则

## 编译

在 `/path/to/deepright/cli/module/connect` 目录执行：

```bash
go build -o ../plugins/email ./email
```

编译后的二进制会输出到 `connect` 模块同级 `plugins` 目录：

```bash
../plugins/email
```

## 固定输出命令

```bash
../plugins/email param
../plugins/email name
```

返回值固定为：

```json
["email","email_pop3","email_smtp","email_password","email_whitelist"]
```

```json
{"key":"email","name":"邮件"}
```

## 配置方式

`email` 插件运行时从 Integration 代理的 Connect 能力中读取 `name=email` 的配置，典型配置如下：

```json
{
  "email": "demo@example.com",
  "email_pop3": "pop.example.com:995",
  "email_smtp": "smtp.example.com:465",
  "email_password": "secret",
  "email_whitelist": "alice@example.com,bob@example.com",
  "mode": "email"
}
```

字段说明：

- `email`：邮箱账号
- `email_pop3`：POP3 地址
- `email_smtp`：SMTP 地址
- `email_password`：邮箱密码或授权码
- `email_whitelist`：用 `,` 分隔的白名单邮箱；为空表示不过滤

## 通过 Integration 注册

```bash
./integration connect meta-create \
  --name email \
  --meta '{"email":"'$EMAIL'","email_address":"'$EMAIL_ADDRESS'","email_password":"'$EMAIL_PASSWORD'","mode":"email"}' \
  --stream true \
  --callback ./email \
  --agent a \
  --model deepseek
```

如果需要完整传入 POP3、SMTP 和白名单，推荐按实际运行参数补齐对应字段。

## 启动与停止

通过 Integration 启动插件：

```bash
./integration plugins start --name email
```

如果需要单独验证插件 CLI，可直接执行：

```bash
./email start --connect-bin ../integration/integration
./email stop --pid-file ./email.pid
```

说明：

- `start` 如果发现已有运行中的进程，会先执行一次 `stop`，整体行为等同于 `restart`
- 邮件插件启动后会每 `60` 秒扫描一次未读邮件
- 运行日志默认写入 `email.log`

## 消息处理规则

收到邮件后，插件会将邮件转换后推送到 `add-request`，处理规则如下：

- 准入：发件人必须命中 `email_whitelist`，或者等于 `email` 自身地址
- 时间线：记录每次处理到的最后邮件时间，下次仅处理该时间之后的邮件
- 去重：记录已准入邮件的 `Message-ID`，避免重复推送
- `create_time`：取邮件头中的 `Date`
- 原始报文：按 `{"headers":[{}],"content":""}` 结构组装 JSON
- 编码：统一转为 UTF-8，避免正文和日志乱码

## 附件与资源下载

邮件中的图片、文件资源会下载到启动目录下的 `email_artifacts` 目录，不存在时自动创建。

归一化后会追加到消息内容中：

- 图片：`[image]绝对路径`
- 文件：`[file]绝对路径`

命名规则：

- 图片使用 `image_key` 命名
- 文件使用 `file_key` 命名

支持的资源来源包括：

- 邮件附件
- 邮件正文中的内嵌图片或文件
- 附件邮件中的资源

下载失败时只记录日志，不会因为空响应导致 `email` 进程退出。

## 验证流程

```bash
./integration --agent-dir ../agent/test-case --site ../site
```

```bash
./integration connect meta-create \
  --name email \
  --meta '{"email":使用环境变量EMAIL,"email_address":使用环境变量EMAIL_ADDRESS,"email_password":使用环境变量EMAIL_PASSWORD,"mode":"email"}' \
  --stream true \
  --callback ./email \
  --agent a \
  --model deepseek
```

```bash
./integration plugins start --name email
```

```bash
./email start --connect-bin ../integration/integration
```

```bash
./email stop --pid-file ./email.pid
```

```bash
./integration meta-delete --name email
```
