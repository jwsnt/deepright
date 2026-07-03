# Skills 迭代 20260512-1 使用手册

## 本次新增

- 增加 `SKILL.md` 解析失败巡检能力
- 增加 `skills_warning` sqlite 告警表
- 增加 `warning-scan` / `warning-list` CLI

## 使用方式

```bash
cd /path/to/deepright/cli/module/skills
go build -o skill-scanner ./
```

单次巡检并同步告警：

```bash
./skill-scanner warning-scan --interval 0 ./test-case
```

持续每分钟巡检：

```bash
./skill-scanner warning-scan ./test-case
```

读取当前告警：

```bash
./skill-scanner warning-list
```

## 告警说明

- 仅检查文件名为 `SKILL.md` 的文件
- 解析失败时会记录：
  - 错误文件绝对路径
  - 失败原因
  - 扫描时间
- 同一个文件修复后，下一次巡检会自动删除对应告警

## 存储说明

- sqlite 文件名固定默认使用 `data`
- `WarningStore` 会复用已打开连接，避免每次扫描重复创建 sqlite 连接
- 更完整说明请参考上级手册 `../../USER_GUIDE.md`
