# 20260718-1 使用手册

## 本次变更

飞书、邮件、Browser 等 Connect 插件精简了 `--help` 输出。帮助内容不再介绍 `start`、`stop`、`init` 三个命令；Browser 的 `daemon --help` 和 `instance --help` 也遵循相同规则。

此变更只影响帮助文案，不改变命令能力、参数、执行结果或 Integration 的插件生命周期管理流程。插件 `command` 子命令返回的 JSON 能力列表也保持不变。

## 查看帮助

```bash
./plugins/feishu --help
./plugins/email --help
./plugins/browser --help
./plugins/browser daemon --help
./plugins/browser instance --help
```

上述输出不会展示被隐藏命令的用法、说明或案例；其他插件使用方式仍以其自身帮助和主手册为准。

## 兼容性

- 已有脚本和 Integration 内部流程仍可继续调用被隐藏的命令
- `command` 返回的能力列表不变，便于既有能力发现和兼容逻辑继续工作
- `integration plugins --help` 是 Integration 管理命令，不属于插件自身帮助，本次不修改
