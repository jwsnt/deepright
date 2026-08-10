# 20260507-2 使用手册

## 目标

本次迭代调整了 `proxy` 在“任务完成后回推三方消息”流程中的插件能力检查方式：

- 完成回推继续调用插件的 `send`
- 但在真正执行前，优先调用 `<callback> command`
- 只有当插件声明支持 `send` 时，才执行 `<callback> send ...`
- 如果旧插件暂未实现 `command`，仍兼容回退到 `<callback> --help`

## 执行流程

当 `proxy` 扫描最近 24 小时内已完成且类型不为 `cron` 的任务明细时，会：

1. 通过任务明细中的 `META_ID/meta_ref` 精确定位对应的 connect 原始 request
2. 仅在该原始 request 状态仍为“已启动”时继续处理
3. 通过 `connect meta-list` 读取该插件配置里的 `callback` 绝对路径
4. 先执行 `<callback> command`，确认返回能力列表中包含 `send`
5. 如果 `command` 不可用，则兼容回退到 `<callback> --help` 检查
6. 检查通过后，执行：

```bash
plugin send --message '{原始消息JSON}' --content '任务完成结果'
```

## 行为说明

- `send` 仍然必须带上 `--message`，内容为精确匹配到的原始 request JSON
- 插件能力判断从旧的“直接看 `--help`”切换为优先看 `command`
- 如果插件未声明 `send`，proxy 会记录对应插件日志并跳过本次回推
- 为兼容尚未升级的旧插件，只有在 `command` 检查失败时，才回退到 `--help`
- 跳过回推时不会误匹配其他 request；仍然只认 `META_ID/meta_ref` 指向的那条原始消息

## 验证

本次迭代已完成以下对齐：

- `proxy` 完成回推前优先通过 `command` 检查 `send`
- 旧插件缺少 `command` 时，仍可通过 `--help` 兼容执行 `send`
- `proxy` 主手册已同步说明 `init/send` 都优先走 `command`
- 已补充 `send` 的兼容回退测试
