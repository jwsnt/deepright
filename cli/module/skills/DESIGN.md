# Skills 模块详细技术设计

## 1. 模块定位

`skills` 模块当前是一个 Go CLI 工具模块，职责主要有两类：

- 扫描目录中的技能文档并输出结构化技能列表。
- 扫描解析失败的 `SKILL.md` 并把告警同步到 SQLite。

它不是一个常驻服务，也没有 HTTP 接口。模块对外暴露的是命令行入口和一个可复用的共享内核包 `skillscore`，供其他模块直接调用。

当前模块名来自 [`go.mod`](./go.mod)：

```go
module skill-scanner
```

因此主程序与其他模块复用时，都是围绕“技能扫描器”这个定位展开，而不是完整的技能运行时。

## 2. 代码边界

### 2.1 主程序入口

- [`main.go`](./main.go)
  - 命令行分发入口。
  - 支持三种运行模式：
    - 普通扫描
    - `warning-scan`
    - `warning-list`
  - 对外保留了 `GetSkillsOutputJSON(root, ttl)` 包装函数，实际转调 `skillscore.GetOutputJSON`。

### 2.2 共享扫描内核

- [`skillscore/skillscore.go`](./skillscore/skillscore.go)
  - 定义技能数据结构与告警数据结构。
  - 实现 front matter 解析、字段校验、目录扫描、JSON 输出。
  - 提供多个扫描变体：
    - `Scan`
    - `ScanAgentSkills`
    - `ScanWarnings`
    - `ScanWithWarnings`
    - `GetOutputJSON`

### 2.3 告警持久化

- [`skillscore/warning_store.go`](./skillscore/warning_store.go)
  - 提供 SQLite 存储 `skills_warning` 表。
  - 实现 warning 全量同步、列表查询、JSON 输出。
  - 内部维护按绝对路径缓存的 `WarningStore` 实例。

### 2.4 测试与样例

- [`main_test.go`](./main_test.go)
  - 覆盖 CLI 对外暴露能力与 SQLite 同步行为。
- [`skillscore/skillscore_test.go`](./skillscore/skillscore_test.go)
  - 覆盖 front matter、字段校验、扫描边界、正则规则。
- [`test-case/`](./test-case/)
  - 提供扫描测试样本。

## 3. 对外命令设计

### 3.1 普通扫描

命令格式：

```bash
skill-scanner [--skill-cache ms] <directory>
```

实现入口是 `runScan(args)`，行为如下：

- 解析 `--skill-cache`，默认 `10000` 毫秒。
- 读取目标目录。
- 调用 `GetSkillsOutputJSON(root, ttl)`。
- 将结果直接打印为格式化 JSON。

需要注意的是，`ttl` 参数目前只是接口兼容参数，实际实现中没有缓存逻辑，`GetOutputJSON` 内部会直接扫描目录，然后执行：

```go
_ = ttl
```

所以当前的真实行为是“每次都实时扫描”，不是“带 TTL 的目录缓存”。

### 3.2 `warning-scan`

命令格式：

```bash
skill-scanner warning-scan [--db data] [--interval 1m] <directory>
```

实现入口是 `runWarningScan(args)`，行为如下：

- `--db`
  - 默认值是 `data`，表示 SQLite 文件路径。
- `--interval`
  - 默认 `1m`。
  - `0` 或负数表示只跑一次。

执行流程：

1. 调用 `skillscore.ScanAndSyncWarnings(root, dbPath)` 扫描并全量同步告警。
2. 再次打开 warning store。
3. 读取 `ListJSON()` 结果并打印。
4. 如果初次执行失败则退出；如果进入周期循环后失败，只打印错误并继续下一轮。

输出规则有一个实现细节：

- 如果本轮扫描告警数为 `0`，直接打印 `[]`。
- 如果有告警，打印数据库中的完整 JSON 列表。

### 3.3 `warning-list`

命令格式：

```bash
skill-scanner warning-list [--db data]
```

实现入口是 `runWarningList(args)`，行为非常直接：

- 打开指定 SQLite 文件。
- 查询 `skills_warning` 表。
- 按 JSON 格式输出当前告警列表。

### 3.4 帮助信息

当命令参数为以下值时：

- `help`
- `--help`
- `-h`

主程序会打印固定 usage 文本，而不会进入扫描逻辑。

## 4. 数据模型设计

### 4.1 Skill

核心结构体是：

```go
type Skill struct {
    Name          string             `json:"name" yaml:"name"`
    Location      string             `json:"location" yaml:"-"`
    Description   string             `json:"description" yaml:"description"`
    License       string             `json:"license,omitempty" yaml:"license"`
    Compatibility compatibilityValue `json:"compatibility,omitempty" yaml:"compatibility"`
    Metadata      map[string]any     `json:"metadata,omitempty" yaml:"metadata"`
    AllowedTools  string             `json:"allowed-tools,omitempty" yaml:"allowed-tools"`
}
```

字段语义如下：

- `Name`
  - 技能唯一标识。
- `Location`
  - 解析完成后补写的绝对文件路径，不从 YAML 读取。
- `Description`
  - 技能描述。
- `License`
  - 可选授权描述。
- `Compatibility`
  - 兼容性要求，内部类型是自定义 `compatibilityValue`。
- `Metadata`
  - 任意键值的附加元数据。
- `AllowedTools`
  - 原样保存为字符串，没有拆成数组。

### 4.2 SkillWarning

告警结构体是：

```go
type SkillWarning struct {
    Path   string `json:"path"`
    Reason string `json:"reason"`
    Time   int64  `json:"time"`
}
```

用于表达“某个 `SKILL.md` 被发现但无法成功解析”的结果。

## 5. 技能文档解析设计

### 5.1 front matter 提取

解析入口是 `parseSkill(path)`，它首先读取整个文件，再通过正则 `frontMatterRe` 提取开头的 YAML front matter。

正则规则：

```go
(?s)^-{3,}\n(.*?)\n-{3,}
```

这意味着当前 front matter 识别有以下约束：

- 必须出现在文件开头。
- 起止分隔线至少 3 个 `-`。
- 起止分隔线都要求独占一行。
- 空 front matter 不算有效。

如果不满足，会报：

- `缺少 YAML front matter`

### 5.2 YAML 反序列化

成功提取后，模块使用 `gopkg.in/yaml.v3` 反序列化到 `Skill`。

其中 `Compatibility` 字段支持两种 YAML 形态：

- 标量字符串
- 字符串数组

如果是数组，会在 `compatibilityValue.UnmarshalYAML` 中：

- 去除空白项
- 用 `"; "` 连接成单个字符串

例如测试中：

```yaml
compatibility:
  - macOS (Darwin)
  - zsh shell
```

会被标准化为：

```text
macOS (Darwin); zsh shell
```

如果 `compatibility` 既不是标量也不是数组，就返回：

- `compatibility 字段无效`

### 5.3 字段校验

`validateSkill()` 当前只校验三个方面：

- `name`
  - 非空
  - 长度不超过 64
  - 满足正则：
    - `^[a-zA-Z0-9_]([a-zA-Z0-9_-]*[a-zA-Z0-9_])?$`
- `description`
  - 去空白后非空
  - 长度不超过 1024
- `compatibility`
  - 结果字符串长度不超过 500

当前没有校验：

- `license`
- `metadata` 内容结构
- `allowed-tools` 格式
- Markdown 正文内容

### 5.4 路径补写

校验通过后，`parseSkill()` 会把源文件路径转成绝对路径写入 `Location`。因此 JSON 输出中的 `location` 一定是绝对路径。

## 6. 扫描设计

### 6.1 基础扫描

底层扫描函数是：

```go
scan(root string, allowPlainSkill bool)
```

扫描方式为 `filepath.WalkDir` 全量遍历目录树，处理规则如下：

- 默认只识别文件名为 `SKILL.md` 的文件。
- 当 `allowPlainSkill == true` 时，也接受文件名为 `SKILL` 的文件。

对外接口对应关系：

- `Scan(root)`
  - 只扫描 `SKILL.md`
- `ScanAgentSkills(root)`
  - 允许 `SKILL.md` 和 `SKILL`
- `ScanWarnings(root)`
  - 只返回告警
- `ScanWithWarnings(root)`
  - 同时返回技能和告警

### 6.2 同名技能覆盖规则

扫描结果不是简单 append，而是通过 `map[string]Skill` 以 `Skill.Name` 作为 key 聚合。

规则是：

- 首次看到某个 `name` 时，记录它在输出顺序中的位置。
- 之后如果再次扫到同名技能，会覆盖该名字对应的技能内容。
- 但输出顺序仍保留第一次出现时的位置。

因此当前语义是：

- “技能名是主键”
- “后发现的同名技能覆盖前者”
- “列表顺序按首次出现顺序保持稳定”

### 6.3 告警收集规则

扫描过程中如果某个候选文件解析失败：

- 只有文件名为 `SKILL.md` 时才会生成 warning。
- `SKILL` 文件解析失败不会写入 warning 列表。

warning 内容包括：

- 绝对路径
- 解析错误原因
- 当前 Unix 时间戳

也就是说，warning 的设计目标是面向标准技能文档巡检，而不是面向 agent 私有 `SKILL` 文件的所有异常。

### 6.4 失败边界

如果 `filepath.WalkDir` 自身失败，例如：

- 根目录不存在
- 某一层目录不可读

扫描函数会直接返回错误，而不是吞掉异常继续扫描。

## 7. JSON 输出设计

普通扫描输出由 `GetOutputJSON(root, ttl)` 生成，内部逻辑是：

1. 调用 `Scan(root)`。
2. 对 `[]Skill` 执行 `json.MarshalIndent(..., "", "  ")`。

因此输出是稳定格式化 JSON，而不是流式输出或 NDJSON。

warning 列表输出由 `WarningStore.ListJSON()` 生成，同样使用 `json.MarshalIndent`。

## 8. Warning Store 设计

### 8.1 SQLite 表结构

告警持久化表定义为：

```sql
CREATE TABLE IF NOT EXISTS skills_warning (
    path TEXT PRIMARY KEY,
    reason TEXT NOT NULL,
    time INTEGER NOT NULL
);
```

这意味着：

- 同一路径的 warning 在数据库里只有一条。
- `path` 是唯一主键。

### 8.2 Store 打开与缓存

`OpenWarningStore(dbPath)` 会先把 `dbPath` 转为绝对路径，再使用该绝对路径作为内存 cache key。

当前实现细节：

- 相同绝对路径的数据库只会打开一次，并复用同一个 `WarningStore`。
- `sql.DB` 配置为：
  - `SetMaxOpenConns(1)`
  - `SetMaxIdleConns(1)`
  - `SetConnMaxLifetime(0)`

这个缓存只存在于当前进程内，进程退出后不会保留。

### 8.3 同步策略

`WarningStore.Sync(warnings)` 采用“全量覆盖”策略，而不是增量更新：

1. 开启事务。
2. `DELETE FROM skills_warning`
3. 把本轮 warning 全部重新插入。
4. 提交事务。

因此 warning store 反映的是“最近一次扫描的完整结果”，不是历史归档。

如果某个错误已经修复，那么下一次 `Sync()` 会把旧记录直接删掉。`main_test.go` 里也专门验证了这个行为。

### 8.4 列表查询

`WarningStore.List()` 查询 SQL 为：

```sql
SELECT path, reason, time FROM skills_warning ORDER BY path
```

因此输出顺序按路径字典序稳定排序。

## 9. 与其他模块的协作边界

`skills` 模块本身不执行技能，只负责发现和描述技能。它当前可被上游模块复用的主要接口有：

- `Scan`
- `ScanAgentSkills`
- `ScanWarnings`
- `ScanWithWarnings`
- `GetOutputJSON`
- `OpenWarningStore`
- `ScanAndSyncWarnings`

其中比较关键的边界是：

- 普通技能扫描只认 `SKILL.md`
- Agent 场景扫描额外兼容 `SKILL`
- warning 同步只针对 `SKILL.md`

这三个边界区分了“标准技能目录”和“Agent 特殊技能文件”的处理差异。

## 10. 测试现状

### 10.1 `main_test.go`

当前覆盖的关键行为包括：

- 扫描测试样例目录后应得到 3 个技能。
- `location` 必须是绝对路径。
- `compatibility` 序列应被标准化为字符串。
- `metadata` 与 `license` 能正确透传。
- `ScanAndSyncWarnings()` 会在修复坏文件后清空旧告警。
- `ListJSON()` 能输出数据库中的 warning。
- SQLite 文件路径 `data` 会被真实创建并可查询到 `skills_warning` 表。

### 10.2 `skillscore_test.go`

当前覆盖的关键行为包括：

- front matter 提取成功与失败。
- `name` 正则校验。
- `description` / `compatibility` 长度限制。
- 缺失 front matter、YAML 非法时的错误。
- 空目录扫描行为。
- `FlushCache()` 可调用但当前为空实现。

### 10.3 测试缺口

当前没有覆盖：

- CLI 参数解析分支的端到端测试。
- 周期性 `warning-scan --interval` 行为测试。
- 同名技能覆盖规则的显式测试。
- 大目录性能测试。
- 并发复用 `WarningStore` 的稳定性测试。

## 11. 当前实现约束

### 11.1 `--skill-cache` 目前无效

虽然主命令暴露了缓存 TTL 参数，但扫描逻辑没有缓存层，`FlushCache()` 也是空实现。当前文档和调用方如果把它当成“真实缓存能力”，会产生误解。

### 11.2 front matter 解析较严格

当前正则只接受非常固定的 front matter 位置和格式，兼容性不算宽松。某些 Markdown 解析器能接受的写法，这里未必能通过。

### 11.3 校验范围有限

目前只校验：

- `name`
- `description`
- `compatibility`

其他字段几乎都是“能反序列化就接受”，因此上游如果需要更严格的 schema，还要自己补规则。

### 11.4 warning 不是历史日志

数据库采用全量覆盖同步，无法追溯历史错误演化，只能看到最近一次扫描的当前状态。

### 11.5 没有文件系统监听

扫描完全依赖命令触发或定时轮询，没有 watch 机制。因此“实时”在当前实现里指的是“每次执行都重扫磁盘”，而不是“文件变化自动推送”。

## 12. 演进建议

如果后续继续扩展这个模块，比较自然的方向是：

1. 决定 `--skill-cache` 是删除兼容参数，还是补上真正的 TTL 缓存。
2. 为 `allowed-tools`、`metadata` 和正文结构补充更明确的校验规则。
3. 给同名技能覆盖策略加显式测试，并在文档中对外稳定化。
4. 如果需要历史审计，再把 `skills_warning` 从“快照表”升级成“快照 + 历史表”。

当前版本的 `skills` 模块更适合被理解为“技能清单扫描器 + 告警快照同步器”，而不是完整的技能生命周期管理系统。
