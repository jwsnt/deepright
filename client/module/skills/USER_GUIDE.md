# Skill Scanner 使用手册

## 简介

Skill Scanner 是一个命令行工具，用于遍历指定目录及其子孙目录，查找所有 `SKILL.md` 文件，提取其中的 YAML 元数据，并输出为 JSON 数组。

当前实现同时提供：

- `skill-scanner` CLI
- 共享扫描内核 `skillscore`
- `skills_warning` 告警扫描与 sqlite 持久化能力

其中共享内核可供 `agent` 及其上游模块复用，但 `skills` 模块自身的 CLI 仍只对 `SKILL.md` 生效。

本次更新后，技能扫描改为每次实时遍历目录：

- 每次执行都会重新扫描指定目录及其子孙目录
- 不再缓存技能元数据
- 每分钟巡检场景可将解析失败的 `SKILL.md` 同步到同目录下的 `data` sqlite

## 安装

```bash
go build -o skill-scanner .
```

## 使用方法

```bash
./skill-scanner [--skill-cache <毫秒数>] <目录路径>
./skill-scanner warning-scan [--db data] [--interval 1m] <目录路径>
./skill-scanner warning-list [--db data]
```

### 参数说明

| 参数 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `<目录路径>` | 是 | — | 要扫描的根目录 |
| `--skill-cache` | 否 | 保留兼容 | 当前版本不再缓存技能元数据；如仍传入该参数，仅用于兼容旧调用 |
| `warning-scan --db` | 否 | `data` | 告警 sqlite 文件名 |
| `warning-scan --interval` | 否 | `1m` | 巡检周期；传 `0` 表示只执行一次 |
| `warning-list --db` | 否 | `data` | 读取告警 sqlite 文件名 |

### 示例

```bash
# 扫描 test-case 目录，实时输出最新技能元数据
./skill-scanner test-case

# 兼容旧调用；即使传入 --skill-cache，仍会实时扫描
./skill-scanner --skill-cache 60000 ./skills

# 实时扫描 skills 目录
./skill-scanner ./skills

# 立刻扫描一次并把解析错误写入 data sqlite
./skill-scanner warning-scan --interval 0 ./skills

# 每分钟巡检一次并持续同步 skills_warning 表
./skill-scanner warning-scan ./skills

# 读取当前 sqlite 中保存的解析错误
./skill-scanner warning-list
```

## SKILL.md 文件格式

在文件头部使用 `---`（至少三个短横线）包裹 YAML 元数据：

```markdown
---
name: my-skill
description: 技能描述
license: MIT
compatibility: macOS 12+
metadata:
  author: example
  version: "1.0"
allowed-tools: tool-a tool-b
---

# 正文内容（不会被提取）
```

`compatibility` 也支持列表写法，例如：

```markdown
---
name: my-skill
description: 技能描述
compatibility:
  - macOS (Darwin)
  - zsh shell
---
```

最终输出时会被规范化为：

```json
{
  "compatibility": "macOS (Darwin); zsh shell"
}
```

### 字段说明

| 字段 | 必填 | 约束 |
|------|------|------|
| `name` | 是 | 最多 64 字符，字母、数字、下划线和连字符 |
| `description` | 是 | 最多 1024 字符，不能为空 |
| `license` | 否 | 许可证名称或引用 |
| `compatibility` | 否 | 最多 500 字符，环境要求说明；支持字符串或字符串列表，列表会被规范化为以 `; ` 连接的单个字符串 |
| `metadata` | 否 | 自定义键值映射 |
| `allowed-tools` | 否 | 以空格分隔的工具列表 |

## 输出格式

JSON 数组，每个元素为一个技能对象，未声明的可选字段会被省略：

```json
[
  {
    "name": "my-skill",
    "location": "/absolute/path/to/skill/SKILL.md",
    "description": "技能描述",
    "license": "MIT"
  }
]
```

## 作为子模块调用

```go
import "skill-scanner/skillscore"

skills, err := skillscore.Scan("/path/to/skills")
if err != nil {
    log.Fatal(err)
}
fmt.Println(skills)
```

或直接获取 JSON：

```go
data, err := skillscore.GetOutputJSON("/path/to/skills", 10*time.Second)
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(data))
```

说明：

- 出于兼容原因，相关 API 仍保留原有函数签名
- 即使继续传入缓存时长参数，当前版本也会实时扫描目录并返回最新结果
- `skillscore.ScanWithWarnings(root)` 可同时返回成功 skills 与失败告警
- `skillscore.ScanAndSyncWarnings(root, "data")` 可直接把告警同步到 sqlite

## 告警巡检

当 `SKILL.md` 存在以下问题时，会记录一条解析告警：

- 缺少 YAML front matter
- YAML 解析失败
- `name` / `description` / `compatibility` 字段不符合约束
- 文件读取失败

告警表名固定为 `skills_warning`，字段如下：

| 字段 | 说明 |
|------|------|
| `path` | 错误 `SKILL.md` 的绝对路径 |
| `reason` | 解析失败原因 |
| `time` | 最近一次扫描到该错误的 Unix 时间戳（秒） |

说明：

- 每次巡检都会先重新扫描目录，再全量同步当前错误集合
- 某个 `SKILL.md` 修复后，对应告警会在下一次巡检时自动从 sqlite 删除
- sqlite 默认使用当前目录下文件名为 `data` 的数据库文件
- `WarningStore` 内部复用连接池，不会为每次扫描重新创建连接

## 注意事项

- 仅识别文件名为 `SKILL.md` 的文件，其他文件名（如 `SKILL`）会被忽略
- 缺少必填字段（`name` 或 `description`）的文件会被跳过
- 如果存在同名技能，后扫描到的会覆盖先前的
- 每次执行都会实时扫描目录，不会复用历史缓存结果
- `location` 固定为技能文件本身的绝对路径，而不是目录路径
- 解析失败的 `SKILL.md` 不会出现在正常 skills JSON 输出中，而是进入 `skills_warning` 告警表
