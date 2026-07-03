# remote 迭代 20260521-5 使用手册

## 本次完成内容

- `start` 命令会把命令生命周期写入 `remote.log`
- `stop` 命令会把命令生命周期写入 `remote.log`
- 新增 `scope` 命令，固定返回空数组 `[]`

## scope 命令

执行：

```bash
./remote scope
```

返回：

```json
[]
```

说明：

- Remote 当前固定返回空数组
- 这表示当前不暴露容器通用配置范围
- 该行为稳定，不依赖运行态、不触发 daemon 启动

## start/stop 日志

执行：

```bash
./remote start
./remote stop
```

会在 `remote.log` 中追加生命周期日志，例如：

```text
remote-cli 2026/05/21 12:00:00 start requested
remote-cli 2026/05/21 12:00:01 start completed pid=12345
remote-cli 2026/05/21 12:10:00 stop requested
remote-cli 2026/05/21 12:10:00 stop completed
```

说明：

- `start requested` 表示收到启动请求
- `start completed` 表示 manager 已成功拉起
- `stop requested` 表示收到停止请求
- `stop completed` 表示 stop 清理已完成
- 如果启动或停止失败，也会把失败原因写进 `remote.log`

## 验收重点

- `remote scope` 稳定返回 `[]`
- `remote start` 后，`remote.log` 中能看到 `start requested` 和 `start completed`
- `remote stop` 后，`remote.log` 中能看到 `stop requested` 和 `stop completed`

## 对应需求

- [REQUIREMENT.md](REQUIREMENT.md)
