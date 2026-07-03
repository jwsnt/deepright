### 职责
+ 从成员回复中提取两部分内容:
    + `passed`: Passed / Failed
    + `how_to_improve`: How to improve, the more detailed, the better, such as: 'I need to...' 'It is required that..'
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
