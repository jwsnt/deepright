# 20260507-2 User Guide

## 目标

本次迭代补充 `feishu` 与 `connect add-request` 的对接说明，明确飞书插件保存三方请求时统一走 `add-request` 命令，不直接操作数据库。

## 接收链路

飞书插件收到消息后，会先在本地完成消息解析、附件下载和内容归一化，然后通过 Integration 代理的 Connect 能力写入请求：

```bash
./integration connect add-request \
  --key feishu \
  --externalId <md5(create_time+content)> \
  --content <归一化后的文本> \
  --artifacts <附件绝对路径，逗号分隔> \
  --original <原始飞书事件 JSON> \
  --created <消息时间戳>
```

说明：

- `key` 固定使用 `feishu`
- `externalId` 由 `create_time + content` 的 MD5 生成
- `content` 为归一化后的文本内容
- `artifacts` 为下载后的本地绝对路径
- `original` 保存原始事件 JSON，用于后续 `send` / `init` 回复

## 模块边界

- `feishu` 插件只通过 `connect` / `integration connect` CLI 与 Connect 交互
- 插件不会直接访问 SQLite
- 插件不会直接依赖 Agent 目录

## 主动回复的输入来源

后续执行：

```bash
../plugins/feishu send --message 原消息报文JSON ...
../plugins/feishu init --message 原消息报文JSON ...
```

其中 `--message` 推荐直接使用 `connect add-request` 返回的请求 JSON；插件优先读取其中的 `original` 字段，同时兼容旧字段 `rawRequest`。

## 验证方式

```bash
./integration connect meta-create --name feishu --meta '{"appId":"x","appSecret":"y"}' --callback ./feishu --agent a --model deepseek
./integration plugins start --name feishu
```

收到飞书消息后，验证：

- Connect 中新增了一条 `add-request` 请求
- 请求 `key` 为 `feishu`
- 请求里带有 `original` 原始事件 JSON
- 插件自身没有直接操作数据库
