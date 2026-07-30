# 迷你应用参考文档自动恢复使用手册

在主应用静态 `config/config.json` 中为 `miniapp` 配置恢复周期，单位是分钟：

```json
"miniapp": {
  "build": "请使用 [SKILL:__internal_cli] 为 $name 的 $function 构建迷你应用",
  "function": "全部功能",
  "recover": 30
}
```

`recover` 必须为正整数。Integration 启动时立即执行一次检查，随后以该间隔检查 `agent-dir` 下的每个合法 Agent。缺少 `miniapp`、`recover`，或者值不是正整数时，文档恢复会禁用并在 Integration 日志中写明配置错误；不会回退到其它周期。

受保护范围只有每个 Agent 的以下文件：

- `app/API.md`
- `app/CANVAS.md`
- `app/DESIGN.md`

恢复源是当前运行 `default-dir/app/` 下的同名文件，和新建 Agent 时的复制源完全一致。发布包构建阶段已完成的 `#port` 替换会随源文件一同复制；运行时 `--port` 不会改写这三份已发布文档。

当某一目标文件缺失、不是普通文件、内容不同或权限不同，服务只恢复该一个文件，并保留源文件内容和权限。其余两份文档、未变化的文件以及所有其它迷你应用资源不会被覆盖。服务拒绝经 Agent 或 `app` 符号链接逃出工作目录的写入；一个文件失败不会妨碍其它文件或 Agent 继续检查。恢复和失败会记录 Agent、文档名、源路径和目标路径，但不记录文档内容。

`GET /api/runtime_config` 仍会受控只读返回完整 `miniapp` 对象，其中包含 `recover`；该接口不触发扫描、不写配置，也不返回单个 Agent 的恢复结果。
