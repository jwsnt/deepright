# 20260509-4 使用手册

## 目标

本次迭代补齐 Browser 代理 Playwright 时的 `eval --code` 兼容能力，并明确后台 daemon 必须脱离前台命令生命周期。

覆盖范围：

- `browser`
- `browser_playwright`
- 对应 `help`
- 对应 `USER_GUIDE.md`

## eval --code 兼容

本次迭代后，以下两种写法都需要等价：

```bash
./browser --session demo eval 'document.title'
./browser --session demo eval --code 'document.title'
```

```bash
./browser_playwright --session demo eval 'document.title'
./browser_playwright --session demo eval --code 'document.title'
```

说明：

- `--code` 只是兼容入口
- 进入底层执行前会被归一化为标准 `eval <js>`
- 最终不会把 `code` 当成额外业务 flag 继续透传

## daemon 脱离前台命令

本次迭代继续强调后台 daemon 的启动要求：

- `start` 拉起的 daemon 必须独立于当前前台命令生命周期
- 不能继承父进程的临时 `stdout/stderr` 管道
- 前台命令退出后，daemon 仍应继续存活
- 不能只用“端口还能连上”做验收，还要验证 daemon 归属和独立会话语义

在当前实现中：

- 后台标准输出与标准错误会重定向到空设备
- daemon 会以独立会话方式启动
- 运行日志固定写入同目录 `browser.log`

## 文档同步

本次迭代同步更新了：

- `browser/USER_GUIDE.md`
- `browser/playwright/USER_GUIDE.md`
- `browser help`
- `browser_playwright help`

## 验收建议

```bash
./browser_playwright start
./browser_playwright --session demo eval --code 'document.title'
./browser --session demo eval --code 'document.title'
```

重点检查：

- `eval --code` 可正常执行
- 最终请求按 `eval <js>` 处理
- `browser.log` 位于插件或可执行文件同目录
- `start` 结束后 daemon 不会因为前台命令退出而立刻消失
