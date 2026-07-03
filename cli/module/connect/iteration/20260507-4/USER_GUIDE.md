# 20260507-4 使用手册

## 目标

本次迭代把插件配置命令统一到 `integration` 顶层入口，并明确以下语义：

- `meta-create` / `meta-update` 以 `--key` 作为主语义
- `name` 只保留给展示和兼容旧调用，不再作为插件内部识别主键
- `--callback` 仍保留作兼容参数，但实际落库值固定为 `应用启动目录/plugins/<plugin-key>`

## 推荐入口

最终用户优先在 `/path/to/deepright/cli/module/integration` 目录执行：

```bash
./integration connect meta-create --key feishu --meta '{"appId":"cli-app","appSecret":"cli-secret"}' --callback ignored --agent A --chatId chat-001 --model OpenAI
./integration connect meta-update --key feishu --meta '{"appId":"new","appSecret":"secret"}' --callback ignored-again --agent A --chatId chat-002 --model OpenAI
./integration connect meta-get --key feishu
```

说明：

- `--key` 是插件运行时主键，负责插件内部识别、回调映射和通知发送
- `--name` 仅保留作兼容输入，适合展示名或旧脚本迁移
- `--callback` 可以传任意值，但实际保存时不会使用该值
- `meta-get --key <plugin-key>` 返回的是插件运行时配置视图，适合插件自身读取

## Callback 规则

假设应用启动目录为：

```text
/home/integration
```

插件 key 为：

```text
a
```

那么 `meta-create` / `meta-update` 最终落库的 callback 一定是：

```text
/home/integration/plugins/a
```

也就是说，即使执行时传入：

```bash
./integration connect meta-create --key a --meta '{}' --callback ./anything --agent A --model OpenAI
```

实际保存的 callback 仍然会被规范化为 `/home/integration/plugins/a`。

## Connect 兼容入口

底层 `connect` 命令仍兼容这套语义，但只建议用于内部实现联调或兼容排查：

```bash
./connect meta-create --key feishu --meta '{"appId":"cli-app","appSecret":"cli-secret"}' --callback ignored --agent A --model OpenAI
./connect meta-update --key feishu --meta '{"appId":"new","appSecret":"secret"}' --callback ignored-again --agent A --model OpenAI
```

`connect help` 仅作为内部实现或兼容说明入口，不作为最终用户主流程手册入口。

## 验证方式

可以按下面步骤验证当前行为：

```bash
./integration connect meta-create --key feishu --meta '{"appId":"cli-app","appSecret":"cli-secret"}' --callback ignored --agent A --chatId chat-001 --model OpenAI
./integration connect meta-get --key feishu
```

返回结果中应看到：

```json
{
  "key": "feishu",
  "name": "飞书",
  "callback": "/abs/path/plugins/feishu"
}
```

重点检查：

- `key` 为稳定插件主键 `feishu`
- `callback` 已被规范化到 `plugins/feishu`
- 插件后续应始终通过 `meta-get --key feishu` 读取配置

## 兼容说明

- 本次迭代不改变共享 SQLite 连接复用设计
- `connect` / `integration` 仍复用启动时初始化的全局数据库连接
- 现有旧脚本若仍传 `--name` 或自定义 `--callback`，系统会尽量兼容输入，但内部语义统一按 `key` 和固定 callback 规则执行
