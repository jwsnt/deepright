# Skills 迭代 20260511-1 使用手册

## 目标

本次迭代为 `skills` 模块补齐两点：

- 提供可复用的共享技能扫描内核，供 `agent`、`proxy`、`integration` 统一复用
- 保持 `skills` CLI 每次按目录实时遍历 `SKILL.md`，只在 `--skill-cache` 指定的 TTL 内缓存当前 CLI 输出

## 扫描规则

- `skills` 模块自身只识别文件名为 `SKILL.md` 的技能文件
- 递归遍历指定目录及其全部子孙目录
- 从文件头部 `---` 包裹的 YAML front matter 中提取技能元数据
- 如果同名技能重复出现，按扫描顺序后出现的结果覆盖前面的结果

目标输出结构仍为：

```json
[
  {
    "name": "__internal_A",
    "location": "/abs/path/SKILL.md",
    "description": "技能A"
  }
]
```

## 使用方式

按原方式运行即可：

```bash
cd /path/to/deepright/cli/module/skills
go build -o skill-scanner ./
./skill-scanner --skill-cache 10000 ./test-case
```

## 说明

- `location` 固定为当前技能文件 `SKILL.md` 的绝对路径
- `name`、`description`、`compatibility` 等字段约束保持不变
- 本次新增的共享扫描内核不会改变 `skills` CLI 的对外参数和输出格式
- 更完整说明请继续参考上级手册 `../../USER_GUIDE.md`
