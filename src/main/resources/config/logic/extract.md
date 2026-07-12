### 职责
+ 从成员回复中提取两部分内容:
    + `passed`: Passed / Failed
    + `how_to_improve`: How to improve, the more detailed, the better, such as: 'I need to...' 'It is required that..'
+ 所有字符串字段都必须是合法JSON字符串，不能包含未转义的真实换行，需要换行时一律使用`\n`
+ 这里只输出JSON对象本体，不要输出SSE包装内容：例如`data:`、`event:`、`id:`或额外的空行
+ 字符串中的双引号必须转义为`\"`，反斜杠必须转义为`\\`，确保整体内容可被标准`JSON.parse`直接解析
+ 若JSON SCHEMA无对应字段则省略，且不要新增字段、不要修改字段名、不要输出与JSON SCHEMA无关的内容
+ 输出格式必须严格遵守如下JSON SCHEMA，不需要任何解释性描述，不能包含任何非JSON内容，例如Markdown标记（如```）
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
