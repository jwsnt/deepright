# 迭代说明

本次迭代调整 `cli-get` 及页面自定义 CMD（`/api/cmd`）的命令超时回传：命令被超时取消前已经读取到的标准输出和标准错误不会再被丢弃。执行器不会创建标准输入、PTY 或任何运行时用户交互通道；交互式命令仍应使用非交互参数、配置文件或环境变量执行。

## 超时输出规则

命令达到任务 `timeout`（毫秒）后，结果状态均为 `1`，并按已收集的输出内容选择回传文本：

- 未收集到非空白输出时：

  ```text
  [Warning: Command execution timed out.]
  ```

- 已收集到 stdout 或 stderr 时：保留已采集的内容，并按读取合并顺序直接在末尾追加：

  ```text
  <已收集的输出内容>[Warning: Command execution timed out, the returned content may be incomplete.]
  ```

不会额外插入换行、截断已有文本或把 stderr 与 stdout 分开回传。由于超时会终止尚未完成的子进程，追加 Warning 后的内容只能作为命令终止前的部分输出使用。

## 执行链路

### 本地 Shell

`taskexec.Execute()` 使用同一个 `context.WithTimeout` 管理命令生命周期，并分别建立 stdout、stderr 管道后并发读取到同一个受互斥锁保护的缓冲区。

超时取消可能同时触发管道关闭错误。实现会先判断 `context.DeadlineExceeded` 和 `context.Canceled`，再处理管道读取错误，避免已经读取的输出被“file already closed”等派生错误覆盖。超时时使用缓冲区内容生成上述 Warning 文本；手动取消仍保持原有的“命令被终止”语义。

### 外部 CLI_SANDBOX

`ExecuteTaskViaSandboxApp()` 调用外部 `CLI_SANDBOX --cmd <cmd> --timeout <ms>`，并将其 stdout/stderr 合并到同一缓冲区。外层超时时使用同一套格式化规则，之后再将最终文本执行 `GZIP + Base64` 编码写入 `ResultPayload.cmd`。

独立沙盒服务和 WSL helper 也采用相同规则，因此沙盒内部先到达超时时，已获取的输出仍会与不完整提示一起返回给 `cli-get`。

### 页面自定义 CMD

页面确认执行后会将秒数转换为毫秒并通过 `/api/cmd` 发送。integration 与 proxy 的本地执行器使用 `CombinedOutput()` 合并采集 stdout/stderr；命令上下文到期后，二者会以相同的空白判定和 Warning 格式回传缓冲内容。

## 非交互约束

- 不设置 `cmd.Stdin`，不读取或转发用户输入。
- 不分配终端或 PTY，也不新增 WebSocket、SSE 输入通道或弹窗交互。
- stdout 与 stderr 只用于结果采集，不构成命令的交互会话。
- 命令因等待输入、授权、外部服务或锁而未结束时，达到既有任务超时后会被终止并按本手册的超时规则回传。

## 验证覆盖

- 本地 `taskexec`：验证无输出超时文本，以及输出后超时仍保留部分内容。
- 独立沙盒服务：验证无输出和部分输出两种超时结果。
- WSL helper 与外部沙盒包装链路：验证两种格式化分支。
- 原有取消、权限拒绝、成功执行和 `cli/pub` 报文结构保持不变。
