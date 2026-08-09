# 20260507-3 使用手册

## 目标

本次迭代把 `proxy` 的插件配置注册语义同步到最新的 Connect 标准命令：

- 顶层优先使用 `meta-create` / `meta-update`
- `--key` 作为插件内部识别、回调映射和通知发送的唯一主语义
- `--callback` 只保留兼容入参，实际保存值固定为应用启动目录下的 `plugins/<plugin-key>`

## 推荐入口

推荐在 `/path/to/deepright/cli/module/proxy` 目录直接执行：

```bash
./proxy connect meta-create --key feishu --meta '{"appId":"cli-app","appSecret":"cli-secret"}' --callback ignored --agent A --chatId chat-001 --model OpenAI
./proxy connect meta-update --key feishu --meta '{"appId":"new","appSecret":"secret"}' --callback ignored-again --agent A --chatId chat-002 --model OpenAI
./proxy connect meta-get --key feishu
```

说明：

- `proxy connect meta-create` / `proxy connect meta-update` 会直接走内嵌 `connect` 服务的统一校验逻辑
- `proxy connect meta-create` / `proxy connect meta-update` 仍兼容，但只建议用于内部联调或兼容排查
- `meta-get --key feishu` 返回插件运行时配置视图，适合后续插件自身读取

## Callback 规则

假设 `proxy` 启动目录为：

```text
/home/proxy
```

插件 key 为：

```text
feishu
```

则最终落库 callback 一定会被规范化为：

```text
/home/proxy/plugins/feishu
```

也就是说，哪怕执行时传入：

```bash
./proxy connect meta-create --key feishu --meta '{}' --callback ./anything --agent A --model OpenAI
```

实际保存的 callback 仍然不会使用 `./anything`，而是固定为 `plugins/feishu` 对应的绝对路径。

## 验证方式

可以按下面步骤验证：

```bash
./proxy connect meta-create --key feishu --meta '{"appId":"cli-app","appSecret":"cli-secret"}' --callback ignored --agent A --chatId chat-001 --model OpenAI
./proxy connect meta-get --key feishu
```

返回结果应重点检查：

- `key` 为稳定插件主键 `feishu`
- `name` 为插件展示名，例如 `飞书`
- `callback` 已被规范化到 `plugins/feishu` 的绝对路径

## 兼容说明

- 本次迭代不改变现有数据库连接复用设计
- `proxy` 内嵌的 `/api/connect/meta` 也同步支持 `key` 优先语义
- `plugins config` 仍可继续使用，但其内部落库规则与 `meta-create` / `meta-update` 已保持一致
