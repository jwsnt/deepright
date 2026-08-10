# Browser 迭代 20260509-5 使用手册

## 变更目标

本次迭代聚焦 Browser 代理 Playwright 时的 `eval --code` 兼容收口，要求：

- `browser` 支持 `eval <js>` 与 `eval --code <js>` 两种写法
- 底层 `browser_playwright` 同样支持两种写法
- `help` 必须覆盖完整用法
- 日志继续固定写入同目录 `browser.log`
- 后台 daemon 必须真正脱离前台命令生命周期

## 当前结果

当前实现已经满足本轮需求，行为如下：

- `browser` 支持：

```bash
./browser --session demo eval 'document.title'
./browser --session demo eval --code 'document.title'
```

- `browser_playwright` 支持：

```bash
./browser_playwright --session demo eval 'document.title'
./browser_playwright --session demo eval --code 'document.title'
```

- 两种写法在进入执行层前都会被统一归一化成 `eval <js>`
- `--code` 不会继续作为业务 flag 透传到 daemon 执行层
- `browser help` 与 `browser_playwright help` 都已补充 `eval --code` 案例
- `browser.log` 固定写在可执行文件同目录

## daemon 行为

本轮需求同时要求继续保证后台 daemon 的独立生命周期：

- `start` 后台进程不会继承前台命令的临时 `stdout/stderr` 管道
- macOS / Linux 下会以独立会话方式启动
- 前台命令退出后，daemon 仍可继续存活
- 不能只靠端口探活验收，还要校验 PID 文件和 daemon 元数据归属

## 验收命令

```bash
./browser help
./browser --session demo eval --code 'document.title'
./browser_playwright help
./browser_playwright --session demo eval --code 'document.title'
./browser_playwright start
```

## 验收重点

- `help` 中能看到 `eval --code`
- `eval --code` 可以与 `eval <js>` 等价执行
- `browser.log` 位于可执行文件同目录
- `start` 拉起的 daemon 不会在前台命令结束后几秒内自灭
- daemon 复用时会校验自身 PID 和状态目录元数据，避免误连旧进程
