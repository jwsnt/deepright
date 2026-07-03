# Feishu Iteration 20260521-1 User Guide

## 需求目录

- 迭代需求文档：[REQUIREMENT.md](REQUIREMENT.md)
- 飞书插件主需求：[../REQUIREMENT.md](../../REQUIREMENT.md)
- 飞书插件主手册：[../USER_GUIDE.md](../../USER_GUIDE.md)

## 本次迭代内容

本次迭代只补飞书接收侧的待处理聚合规则：

- 每 30 秒扫描一次 10 分钟窗口内尚未推送 `add-request` 的待处理消息
- 同一会话里只要已经出现文本消息，就把该批次里的图片和文件一起归一化后推送
- 如果 10 分钟内始终只有图片或文件消息，则标记过期，不再推送

## 当前行为

飞书插件接收到消息后，会先把解析结果写入本地 pending 队列，而不是立刻逐条执行 `add-request`。

扫描周期内的处理规则如下：

1. 如果当前批次只有文本消息，则本轮直接推送
2. 如果当前批次是“文本 + 文本”，则合并为同一批次直接推送
3. 如果当前批次里同时有文本和图片/文件，则先完成资源下载，再把附件归一化到文本后面一起推送
4. 如果当前批次只有图片或文件，则继续等待下一个扫描周期
5. 如果超过 10 分钟仍未等到文本消息，则整批消息过期丢弃

## 归一化格式

推送到 Connect 时，附件内容会统一转成文本标记：

- 图片：`[image]绝对路径`
- 文件：`[file]绝对路径`

组合后的最终内容示例：

```text
文本消息 [image]/abs/path/demo.png [file]/abs/path/report.pdf
```

如果图片或文件消息先到、文本后到，则会按消息时间顺序聚合，但最终输出仍保持“文本正文在前，附件标记在后”的发送结果。

## add-request 形态

满足推送条件后，插件仍通过 Integration 代理执行：

```bash
./integration connect add-request \
  --key feishu \
  --externalId <聚合后的唯一键> \
  --content <归一化后的最终文本> \
  --artifacts <逗号分隔的本地附件绝对路径> \
  --original <包含 pending 明细的原始 JSON> \
  --created <文本消息时间戳> \
  --schema '<feishu schema 返回值>'
```

## 说明

- 本次迭代只改变接收侧“何时推送 add-request”的规则，不影响 `send` / `init`
- 主回复链路仍然通过原始报文里的锚点消息 `message_id` 进行回复
- 插件仍然只通过 CLI 与 Connect / Integration 交互，不直接访问数据库
