## 变更说明

- `browser help` 已补充 Playwright 代理命令可用的超时参数说明：
  - `--timeout`
  - `--navigation-timeout`
  - `--browser-timeout`
- `browser_playwright help` 也同步补充了上述超时参数说明
- `help` 示例里新增了一条 `eval` 长等待场景，演示如何同时设置动作超时和总命令超时

## 使用示例

```bash
./browser --session agent-a@ctrip-home --timeout 15000 --browser-timeout 30s eval 'new Promise(resolve => setTimeout(resolve, 10000))'
```

说明：

- `--timeout` 控制单次 Playwright 动作超时，单位毫秒
- `--browser-timeout` 控制整条 CLI 命令总超时
