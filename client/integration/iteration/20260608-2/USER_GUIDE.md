# 迭代说明

本次迭代为 `integration` 增加了延迟自关闭接口 `/api/shutdown`，让 HTTP 服务可以在收到关闭请求后约 5 秒自行走完整生命周期退出链路。

## 新增能力

- 新增 `GET /api/shutdown` 与 `POST /api/shutdown`
- 接口返回成功后，会在约 5 秒后触发 integration 自关闭
- 自关闭流程与 `integration stop` 保持等效
- 关闭前会先通过插件自身 `stop` 命令停止已启动插件
- 主进程退出时会继续回收 pid、runtime.json 等运行态文件

## 返回说明

- 返回体固定为 `{"status":0,"data":{"scheduled":...,"delayMs":5000}}`
- 首次调用时 `scheduled=true`
- 同一进程内重复调用时不会重复排队，返回 `scheduled=false`

## 测试

- 补充了 `/api/shutdown` 延迟调度测试
- 覆盖了“先停插件，再取消主进程”的顺序测试
- 覆盖了重复调用只调度一次的幂等测试
