# Integration 迭代 20260511-3 使用手册

## 本次收口

- 新增 `GET /skills_warning`
- 新增 `integration skills-warning [--refresh]`
- 将 `skills` 模块的解析告警能力收口进统一 `integration` 主二进制

## HTTP 用法

```bash
curl http://127.0.0.1:8080/skills_warning
curl http://127.0.0.1:8080/skills_warning?refresh=1
```

## CLI 用法

```bash
cd /path/to/deepright/cli/module/integration
./integration skills-warning
./integration skills-warning --refresh
```

## 行为说明

- 服务启动后会先同步一次告警
- 之后每分钟自动同步一次
- 默认扫描当前运行配置中的 `agent-dir/skills`
- 告警统一写入应用启动目录下的 `data` sqlite
- 修复后的 `SKILL.md` 会在下一次同步时自动移除对应告警

更完整说明请参考上级手册 `../../USER_GUIDE.md`。
