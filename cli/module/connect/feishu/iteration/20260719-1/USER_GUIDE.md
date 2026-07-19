# 飞书迭代手册（20260719-1）

## 需求目录

- 本迭代需求：[REQUIREMENT.md](REQUIREMENT.md)
- 飞书插件主需求：[../../REQUIREMENT.md](../../REQUIREMENT.md)
- 飞书插件主手册：[../../USER_GUIDE.md](../../USER_GUIDE.md)

## 概览

本次新增两个只读查询命令。它们查询的是 Integration / Connect 已落库的飞书消息快照，不访问飞书 Open API，也不会读取飞书日志。Integration / Connect 只提供通用的“消息快照”存储和查询能力；飞书插件负责写入 `source=feishu` 的快照、读取飞书配置并将发送者 ID 呈现为 `openid`，因此 Integration 不依赖飞书插件。

查询窗口由 Integration 运行目录的 `config/config.json` 决定：

```json
{
  "feishu": {
    "lastMessage": 72
  }
}
```

`lastMessage` 必须是大于 0 的整数，单位为小时。命令每次执行时，都会从当前时刻向前回溯该窗口。

## 查询发送者 openid

```bash
../plugins/feishu openid --connect-bin ./integration
```

输出只保留窗口内每个发送者的最后一条消息：

```json
[
  {
    "openid": "ou_7a8b",
    "lastMessageAt": "2026-07-19T10:30:00Z"
  },
  {
    "openid": "ou_9c0d",
    "lastMessageAt": "2026-07-19T09:12:00Z"
  }
]
```

结果按 `lastMessageAt` 从新到旧排序。图片和文件消息也会用于判断某个 `openid` 的最后发送时间。

## 搜索文本消息

单个关键词：

```bash
../plugins/feishu search --query "退款" --connect-bin ./integration
```

不传关键词时，分页列出窗口内全部文本消息：

```bash
../plugins/feishu search --limit 20 --offset 0 --connect-bin ./integration
```

按指定 Open ID 精确过滤：

```bash
../plugins/feishu search --openid ou_7a8b --limit 20 --offset 0 --connect-bin ./integration
../plugins/feishu search --query "退款" --openid ou_7a8b --connect-bin ./integration
```

多个关键词为 AND：

```bash
../plugins/feishu search --query "退款 已处理" --limit 20 --offset 0 --connect-bin ./integration
```

双引号中的内容为完整短语：

```bash
../plugins/feishu search --query '"退款申请" 已处理' --limit 50 --offset 0 --connect-bin ./integration
```

返回示例：

```json
{
  "total": 1,
  "limit": 50,
  "offset": 0,
  "items": [
    {
      "messageId": "om_xxx",
      "openid": "ou_7a8b",
      "content": "退款申请已处理",
      "sentAt": "2026-07-19T10:30:00Z"
    }
  ]
}
```

搜索不区分大小写，只匹配插件归一化后的文本消息；纯图片和纯文件消息不会出现在 `items` 中。`--query` 省略或为空时不施加关键词条件；`--openid` 为精确发送者过滤，可与 `--query` 取 AND。`--limit` 默认 50、最大 200，`--offset` 默认 0。

## 常见错误

- `config/config.json` 不存在或 JSON 无法解析：命令失败，不使用默认窗口。
- `feishu.lastMessage` 缺失、不是整数或小于等于 0：命令失败。
- `--query` 中的双引号未闭合，或 `--limit` / `--offset` 非法：命令失败并提示参数错误。
