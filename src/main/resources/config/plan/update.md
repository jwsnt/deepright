### 规划内容
#plan
### 更新原则
+ 工具`[plan-update]`只能更新现有条目的状态#verify，所有未在原规划中的动作一律视为越界
+ 每完成或跳过一项规划，必须调用工具`[plan-update]`更新状态
    + `[✅]` 表示已完成
    + `[❎]` 表示已跳过或失败
+ 工具`[plan-update]`接收 JSON Array，每项包含：
    + `pattern`：待精确匹配的原文
    + `replacement`：替换后的内容
+ 可一次提交多项更新，批量标记
###### 请求示例
+ 必须严格遵守如下JSON SCHEMA，不需要任何解释性描述，不能包含任何非JSON内容，例如Markdown标记（如```）
``` JSON SCHEMA
[
    {
        "pattern": "[ ]三个文件命名规范：`index.html`、`style.css`、`script.js`",
        "replacement": "[✅]三个文件命名规范：`index.html`、`style.css`、`script.js`"
    },
    {
        "pattern": "[ ]包含简单 README说明如何运行",
        "replacement": "[❎]包含简单 README说明如何运行"
    }
]
```
###### 闭环判定
+ 优先以所有相关Checkpoint通过为准，仅当原规划不存在可用Checkpoint时，才可用"已产出最小可交付结果且剩余风险已说明"作为闭环判定
+ 若存在已完成、已跳过或已失败但尚未同步状态的相关条目，不得直接结束
+ 输出最终结果前，相关条目状态必须与实际执行一致