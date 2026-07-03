# 迭代说明

本次迭代为 `integration` 增加了 `backup-clean` 命令，用于定期整理每个 Agent 工作目录下的 `User/Soul` 备份文件。

## 新增命令

```bash
./integration backup-clean
./integration backup-clean --agent-dir ./agent
./integration backup-clean --archive-after 24h --delete-after 72h
```

## 行为说明

- `integration backup-clean` 会扫描每个 Agent 的 workspace 根目录
- 命中文件名中带 `bak` 或时间戳、且语义上属于 `USER/SOUL` 备份的文件后：
  - 如果文件仍在 workspace 根目录，且最后更新时间已超过 `24h`
  - 会自动移动到该 workspace 下的 `bak/`
- 如果 workspace 下还没有 `bak/`，命令会自动创建
- 如果 `bak/` 里已存在同名文件，命令会自动为新归档文件追加递增后缀，避免覆盖旧备份
- 命令随后会继续检查 `bak/` 中的文件
  - 最后更新时间已超过 `72h` 的文件会被删除

## 识别范围

- 当前生效中的 `USER.md`、`SOUL.md` 不会被当成备份处理
- `USER.md.bak`、`Soul-20260701-120000.md`、`SOUL_20260628_120000.md` 这类文件会被识别为备份
- `USER_GUIDE.md` 这类普通文档不会误判

## 输出示例

```json
{
  "status": 0,
  "agentDir": "/Users/demo/Library/Application Support/deepright/agent",
  "agentCount": 3,
  "archived": [],
  "archivedCnt": 0,
  "deleted": [],
  "deletedCnt": 0
}
```

## 兼容性说明

- `backup-clean` 复用 integration 现有的 Agent 根目录解析规则
- 未传 `--agent-dir` 时，仍会按主应用 `config/config.json`、环境变量 `AGENT_DIR`、默认目录的既有优先级解析
- 该命令属于轻量本地 CLI，不依赖插件运行时初始化，可直接单独执行
