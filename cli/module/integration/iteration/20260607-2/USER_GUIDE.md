# 迭代说明

本次迭代为 `integration` 增加了启动配置文件读取能力。程序启动时会从应用目录下的 `config/config.json` 读取启动参数，并按统一优先级合并到最终配置。

## 新增能力

- 启动时固定读取应用目录下的 `config/config.json`
- 配置优先级固定为：
  - 命令行 `--参数`
  - `config/config.json`
  - 程序内置默认值
- `config/config.json` 支持常见键名写法，例如：
  - `host`
  - `port`
  - `agent-dir` / `agent_dir` / `agentDir`
  - `default-dir` / `default_dir` / `defaultDir`
  - `pid-file` / `pid_file` / `pidFile`
  - `log-file` / `log_file` / `logFile`

## 使用示例

配置文件：

```json
{
  "host": "http://www.dr.cn"
}
```

启动结果：

- 执行 `./integration --host http://www.deepright.cn` 时，最终 `host` 为 `http://www.deepright.cn`
- 执行 `./integration` 时，最终 `host` 为 `http://www.dr.cn`

## 兼容性

- 只读取 `config/config.json`
- 不再兼容应用同目录下的旧路径 `config.json`
- 未在命令行和 `config/config.json` 中提供的参数，仍按原有默认值处理

## 测试

- 新增了读取 `config/config.json` 的测试
- 新增了命令行参数覆盖 `config/config.json` 的测试
- 新增了生命周期默认端口、`pid-file`、`log-file` 从 `config/config.json` 回退读取的测试
- 新增了忽略旧路径同级 `config.json` 的测试
