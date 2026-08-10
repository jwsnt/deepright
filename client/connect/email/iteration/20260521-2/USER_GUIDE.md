# Email Iteration 20260521-2 User Guide

## 需求目录

- 迭代需求文档：[REQUIREMENT.md](REQUIREMENT.md)
- 邮件插件主需求：[../REQUIREMENT.md](../../REQUIREMENT.md)
- 邮件插件主手册：[../USER_GUIDE.md](../../USER_GUIDE.md)

## 本次迭代内容

本次迭代只强调并固化邮件插件的收件触发语义：

- POP3 扫描到准入邮件后，立即调用 `add-request`
- 不再额外等待下一轮批处理窗口
- 推送命令仍统一走 Integration 代理的 Connect CLI 契约

## 当前行为

邮件插件在一次扫描周期内拿到符合条件的邮件后，会立刻执行以下链路：

1. 校验白名单和去重状态
2. 解析主题、正文、附件并完成归一化
3. 立刻调用 `connect add-request`
4. 成功后记录 `Message-ID` 和时间线状态

也就是说，只要本次扫描已经拿到一封准入邮件，就会在当前扫描周期内直接落库，不会为了“等待更多邮件”而延后。

## 命令示意

收到邮件后实际通过 Integration 代理执行的请求形态为：

```bash
./integration connect add-request \
  --key email \
  --externalId <Message-ID 或回退唯一键> \
  --content <归一化后的主题+正文+附件标记> \
  --artifacts <逗号分隔的本地附件绝对路径> \
  --original <原始邮件JSON> \
  --created <邮件时间戳> \
  --schema '<email schema 返回值>'
```

## 说明

- 这里的“立即”是指：邮件已经被当前轮 POP3 扫描取回后，不再做插件内部额外延迟
- POP3 轮询周期仍由 `email_pop3_interval` 控制；它决定“多久扫描一次”，不影响“扫描到后是否立刻推送”
- 本次迭代不改变 `send`、`init`、白名单、去重、持久化或日志字段语义
