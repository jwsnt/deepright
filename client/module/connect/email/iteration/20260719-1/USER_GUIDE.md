# 邮件迭代手册（20260719-1）

## 需求目录

- 本迭代需求：[REQUIREMENT.md](REQUIREMENT.md)
- 邮件插件主需求：[../../REQUIREMENT.md](../../REQUIREMENT.md)
- 邮件插件主手册：[../../USER_GUIDE.md](../../USER_GUIDE.md)

## 概览

本次增加 `sender` 和 `search` 两个只读查询命令。它们查询 Integration / Connect 的本地通用消息快照，不连接 POP3/SMTP、不读取 `email.log`。Integration / Connect 只提供与插件无关的消息快照能力；邮件插件负责写入 `source=email` 快照、读取邮件配置并将发送者 ID 输出为邮箱地址。

查询窗口由 Integration 运行目录的 `config/config.json` 决定：

```json
{
  "email": {
    "lastMessage": 72
  }
}
```

`lastMessage` 必须是大于 0 的整数，单位为小时。每次执行命令时，窗口从当前时刻向前回溯该时长。

## 查询发件人

```bash
../plugins/email sender --connect-bin ./integration
```

`From` 头会被解析为第一个有效邮箱地址、去掉显示名并转为小写。输出按最后发送时间从新到旧排序：

```json
[
  {
    "sender": "alice@example.com",
    "lastMessageAt": "2026-07-19T10:30:00Z"
  }
]
```

同一邮箱只出现一次；邮件 `Date` 不可用时，会使用插件接收时间。

## 搜索邮件

单个关键词：

```bash
../plugins/email search --query "退款" --connect-bin ./integration
```

不传关键词时，分页列出窗口内全部文本邮件：

```bash
../plugins/email search --limit 20 --offset 0 --connect-bin ./integration
```

按指定发件人精确过滤：

```bash
../plugins/email search --sender alice@example.com --limit 20 --offset 0 --connect-bin ./integration
../plugins/email search --query "退款" --sender alice@example.com --connect-bin ./integration
```

多个关键词为 AND：

```bash
../plugins/email search --query "退款 已处理" --limit 20 --offset 0 --connect-bin ./integration
```

双引号中的内容为完整短语：

```bash
../plugins/email search --query '"退款申请" 已处理' --limit 50 --offset 0 --connect-bin ./integration
```

搜索不区分大小写，只匹配归一化后的主题和正文，不匹配附件内容或附件路径。`--query` 省略或为空时不施加关键词条件；`--sender` 为转小写后的精确发件人过滤，可与 `--query` 取 AND。`--limit` 默认 50、最大 200；`--offset` 默认 0。

```json
{
  "total": 1,
  "limit": 20,
  "offset": 0,
  "items": [
    {
      "messageId": "<mail@example.com>",
      "sender": "alice@example.com",
      "content": "退款申请\n已处理",
      "sentAt": "2026-07-19T10:30:00Z"
    }
  ]
}
```

## 常见错误

- `config/config.json` 不存在或 JSON 无法解析：命令失败，不使用默认窗口。
- `email.lastMessage` 缺失、不是整数或小于等于 0：命令失败。
- `--query` 中的双引号未闭合，或 `--limit` / `--offset` 非法：命令失败并提示参数错误。
