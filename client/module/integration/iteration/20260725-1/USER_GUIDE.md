# Agent `tmp` 自动归档与清理

Integration 在启动时从主应用 `config/config.json` 读取 `temp`：

```json
{
  "temp": {
    "clear": 168,
    "pack": 168,
    "scan": 2
  }
}
```

- 三项均为正整数，单位为小时；修改后重启 Integration 生效。
- 启动后立即在后台扫描一次，再每 `scan` 小时扫描一次；不会阻塞服务启动、请求或定时任务。
- `tmp/` 按一级文件或目录归档。目录只有在所有递归文件均超过 `pack` 小时未修改时才移动到 `tmp/bak/`；任一文件较新则整个目录保留。空目录按自身修改时间判断。
- `tmp/bak/` 的一级文件或目录在全部内容超过 `clear` 小时未修改时删除；`bak` 本身不会再次被归档。
- 归档碰到已有同名文件时保留 `bak` 中的文件，源文件不覆盖、不改名，并在后续扫描时重试。
- Integration 日志包含 `temp pack` 归档与冲突跳过、`temp clear` 删除，以及 Agent 处理错误。
