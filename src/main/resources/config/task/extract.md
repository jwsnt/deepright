### 职责
+ 从成员回复中提取两部分内容:
    + `content`: 正文摘要, 其中的文件绝对路径替换为文件名(如`/a/b/c.txt`替换为`c.txt`)
    + `artifacts`: 所有文件的绝对路径列表
+ 提取规则
    + 文件路径可能出现在Markdown表格、JSON、代码块或纯文本中, 需全部提取
    + 同时覆盖Windows和Unix路径格式
+ 响应格式必须严格遵守如下JSON SCHEMA，不需要任何解释性描述，不能包含任何非JSON内容，例如Markdown标记（如```）
``` JSON SCHEMA
#schema
```
    + 正确案例：纯JSON
    {
        ...
    }
    + 错误案例1：带了无关内容前缀
    例如JSON前放了无关内容
    {
        ...
    }
    + 错误案例2：带了Markdown标记```
    ```
    {
        ...
    }
