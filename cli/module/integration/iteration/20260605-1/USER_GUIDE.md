# Integration 迭代手册（20260605-1）

## 本次更新

- integration 服务启动成功后，会在后台延迟约 200ms 异步打开浏览器，不阻塞主服务监听与对外提供 HTTP 能力
- 自动打开地址统一为 `http://localhost:<port>/site/#app`，其中 `<port>` 取当前 `--port` 参数，默认 `8080`
- 自动打开浏览器时，会优先按操作系统规则查找指定浏览器，并附带最大化参数启动
- 如果没有命中优先浏览器，则会回退到系统默认浏览器
- 浏览器打开失败只记日志，不会让 integration 启动失败

## 浏览器查找顺序

### macOS

- `Google Chrome`
- `Google Chrome for Testing`
- `Microsoft Edge`
- `Brave Browser`
- `Chromium`
- 若都不存在，则回退到 `open`

### Linux

- `google-chrome`
- `google-chrome-stable`
- `chromium-browser`
- `chromium`
- `microsoft-edge`
- `microsoft-edge-stable`
- `brave-browser`
- 若都不存在，则回退到 `xdg-open`

### Windows（含 WSL）

- 优先从常见安装目录查找 `Chrome`、`Chrome for Testing`、`Edge`、`Chromium`
- 若未命中，再从 PATH 中查找 `chrome`、`msedge`、`chromium`、`brave`
- 若仍未命中，则回退到 `cmd /c start /max`

## 日志

- 打开成功时会记录实际打开的 URL
- 打开失败时会输出失败原因，但服务保持正常运行

## 同步结果

- `integration/main.go` 已改为使用 integration 模块内的浏览器打开逻辑
- `integration/USER_GUIDE.md` 已同步更新启动行为说明
- 本迭代手册对应当前目录下的 `REQUIREMENT.md`
