# Remote 迭代 20260521-1 使用手册

## 变更说明

本次迭代为 `remote` 新增 `scp` 命令，支持按系统 `scp` 语义执行本地上传和远程下载。

这项能力会通过 `--session` 复用 `remote create` 已建立的 SSH 会话，并继续使用系统 `scp` 二进制完成传输。

## 命令用法

本地上传到远程：

```bash
./remote scp /local/path/file.txt ubuntu@43.155.234.33:/remote/path/ --session agent-a@chat-001
```

从远程下载到本地：

```bash
./remote scp ubuntu@43.155.234.33:/remote/path/file.txt . --session agent-a@chat-001
```

## 行为说明

- `remote scp` 通过 `--session` 复用已缓存 SSH 会话
- 参数顺序、路径方向判断、退出码语义都与系统命令保持一致
- 该命令依赖 `remote create` 先建立 SSH 会话
- 该命令不会修改 `remote.json`
- 如果系统环境中找不到 `scp`，命令会直接报错返回

## 与主手册关系

- 主手册：[../../USER_GUIDE.md](../../USER_GUIDE.md)
- 主需求：[../../REQUIREMENT.md](../../REQUIREMENT.md)
- 当前迭代需求：[REQUIREMENT.md](REQUIREMENT.md)
