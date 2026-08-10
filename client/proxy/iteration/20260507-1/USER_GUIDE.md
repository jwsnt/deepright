# 20260507-1 使用手册

## 目标

本次迭代调整了 `proxy` 在“待处理消息自动转任务”流程中的开始通知方式：

- 开始通知不再调用插件的 `send`
- 改为通过 `connect meta-list` 读取插件配置中的 `callback` 绝对路径
- 执行对应插件的 `init` 命令发送开始通知
- 同时继续通过 `--message` 透传目标原始消息报文 JSON，保证插件仍然按原消息上下文回复

## 执行流程

在某个插件本轮至少生成一条 `started=0` 的待执行明细时，`proxy` 会：

1. 调用 `connect meta-list`，读取该插件配置中的 `callback`
2. 先调用 `<callback> command`，确认返回的能力列表中包含 `init`
3. 如果插件尚未实现 `command`，则兼容回退到 `<callback> --help` 检查
4. 执行开始通知：

```bash
plugin init --message '{原始消息JSON}' --content '<开始执行>可通过新消息更新任务'
```

5. 如果插件自身记录消息流水日志，则开始通知通常会写成 `init ...`

## 行为说明

- 开始通知内容仍来自 `--reply`，默认值为 `<开始执行>可通过新消息更新任务`
- `callback` 路径不会写死在 `proxy` 内，而是始终从 `connect meta-list` 动态读取
- `--message` 仍然传递目标原始消息 JSON，不会因为从 `send` 改为 `init` 而丢失上下文
- 如果插件 `command` 未声明 `init`，或 callback 程序不存在，则会按原有 skip 逻辑记录插件日志并跳过开始通知
- 如果插件暂未实现 `command`，proxy 会继续兼容旧式 `--help` 检查，避免直接打断存量插件
- 任务完成后的结果回推逻辑不受影响，仍然使用插件的 `send --message {} --content {}`

## 验证

本次迭代已完成以下对齐：

- `proxy` 开始通知代码从 `send` 切换为 `init`
- `proxy` 开始通知前会先按插件规范检查 `command`
- `proxy` 主手册已更新开始通知说明
- 插件侧 `feishu` / `email` 已支持 `init` 日志与 `send` 日志区分
