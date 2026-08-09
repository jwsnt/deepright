# 迭代手册

## 本次变更

- Agent metadata 中的 `git` 字段改为每次查询时实时探测
- `git` 不再跟随 `--agent-cache` 返回缓存值

## 行为说明

- `skills` 仍然保持每次实时扫描
- `git` 现在与 `skills` 一样，属于每次读取 metadata 时都会刷新的动态字段
- 其余共享字段仍按原有 `root + appDir + deviceID` 缓存键复用

## 验证方式

```bash
cd /path/to/deepright/cli/module/agent
go test -run 'TestGetAgentOutputIncludesGitField|TestGitPathIsRefreshedEvenWhenCacheIsWarm'
```

## 结果预期

- 即使缓存仍然有效，只要当前机器的 git 安装路径发生变化
- 下一次 `GetAgentOutput` / `GetAgentOutputForApp` 输出里的 `git` 都会立即变为最新值
