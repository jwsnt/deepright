# Static 模块详细技术设计

## 1. 模块定位

`static` 模块当前是一个极简的 Go 静态文件服务模块，职责非常集中：

- 把本地目录映射为 HTTP 下的 `/site/` 路径。
- 为上层模块提供一个可复用的静态资源注册函数。
- 在需要时，也可以独立启动成一个单独的静态服务器进程。

它不是完整的前端网关，也不是通用 Web 框架封装。当前实现只有两层：

- `main.go`：独立运行入口
- `server/`：路由注册逻辑

模块名来自 [`go.mod`](./go.mod)：

```go
module static-server
```

## 2. 代码边界

### 2.1 主程序入口

- [`main.go`](./main.go)
  - 解析命令行参数。
  - 解析站点目录路径。
  - 初始化 `http.ServeMux`。
  - 调用 `server.Register(mux, site)`。
  - 启动 `http.ListenAndServe`。

### 2.2 静态服务内核

- [`server/server.go`](./server/server.go)
  - 对外只暴露一个核心函数：
    - `Register(mux *http.ServeMux, siteDir string) error`
  - 负责校验目录并注册 `/site/` 前缀对应的文件服务。

### 2.3 测试与样例

- [`main_test.go`](./main_test.go)
  - 验证真实 HTTP 服务能从 `/site/` 读取测试文件。
- [`server/server_test.go`](./server/server_test.go)
  - 验证注册成功、非法路径、文件路径误传、以及 `/site/` 前缀映射行为。
- [`test-case/`](./test-case/)
  - 提供最小静态文件样例：
    - [`hello.html`](./test-case/hello.html)
    - [`js/hello.js`](./test-case/js/hello.js)

## 3. 启动与配置设计

### 3.1 命令行参数

`main.go` 当前支持两个参数：

- `--port`
  - 监听端口
  - 默认值：`8080`

- `--site`
  - 静态资源目录
  - 默认值为空

### 3.2 默认站点目录解析

如果用户没有传 `--site`，程序不会使用当前工作目录下的 `./site`，而是使用“可执行文件所在目录下的 `site` 子目录”：

1. 调用 `os.Executable()` 获取当前可执行文件路径。
2. 取其目录 `filepath.Dir(exe)`。
3. 拼接 `site` 目录：
   - `filepath.Join(filepath.Dir(exe), "site")`

这是当前实现里一个很重要的运行时约束。它意味着：

- 默认资源目录跟随二进制文件位置。
- 不跟随当前 shell 的 `cwd`。

### 3.3 监听启动

目录注册成功后，主程序会：

1. 构造地址 `":<port>"`。
2. 打印启动日志。
3. 调用：

```go
http.ListenAndServe(addr, mux)
```

当前没有：

- 优雅关闭
- 超时配置
- TLS
- 多监听器
- 健康检查端点

## 4. 路由注册设计

### 4.1 核心注册函数

`server.Register(mux, siteDir)` 是整个模块最核心的接口。它的职责只有两步：

1. 校验 `siteDir`
2. 在 `mux` 上注册 `/site/` 对应的文件服务

注册逻辑是：

```go
mux.Handle("/site/", http.StripPrefix("/site", http.FileServer(http.Dir(absDir))))
```

这里有几个实现细节非常关键：

- 对外暴露路径前缀是 `/site/`
- `StripPrefix` 使用的是 `"/site"`，不是 `"/site/"`
- 文件服务根目录是传入目录的绝对路径 `absDir`

因此请求：

- `/site/hello.html`

会映射到：

- `<absDir>/hello.html`

请求：

- `/site/js/hello.js`

会映射到：

- `<absDir>/js/hello.js`

### 4.2 路由范围

当前模块只注册一条静态文件路由：

- `/site/`

它不会处理：

- `/`
- `/api/*`
- `/v1/*`
- 其他业务前缀

因此在集成场景下，`static` 的角色是“给上层 mux 挂一段 `/site/` 静态资源目录”，而不是接管整个 HTTP 服务。

## 5. 路径校验设计

在真正注册文件服务前，`Register()` 会做三步校验。

### 5.1 绝对路径解析

先执行：

```go
absDir, err := filepath.Abs(siteDir)
```

如果绝对路径解析失败，返回：

- `cannot resolve site path: ...`

### 5.2 存在性检查

接着执行：

```go
info, err := os.Stat(absDir)
```

如果路径不存在或不可访问，返回：

- `cannot access site directory <absDir>: ...`

### 5.3 目录类型检查

如果路径存在但不是目录，则返回：

- `<absDir> is not a directory`

只有全部通过后，才会真正把文件服务挂到 mux 上。

## 6. 文件服务行为

当前文件服务完全基于 Go 标准库 `http.FileServer`，因此行为也基本继承标准库默认语义。

### 6.1 已实现能力

- 静态文件按路径返回。
- 嵌套目录可访问。
- Content-Type 由标准库按扩展名推断。
- 文件内容直接由标准库流式写回响应。

### 6.2 当前没有的能力

当前模块没有实现以下扩展能力：

- SPA fallback
  - 例如 `/site/chat/123` 自动回退到 `index.html`
- 自定义缓存头
- ETag/版本策略定制
- 禁止目录访问
- 白名单/黑名单过滤
- 文件压缩
- Range/下载策略自定义
- 跨域控制
- 鉴权

所以它的真实定位就是“薄封装的 `http.FileServer` 注册器”。

## 7. 独立运行与集成运行

### 7.1 独立运行

独立运行时，由 [`main.go`](./main.go) 自己创建 `ServeMux` 并监听端口。

适用场景：

- 单独本地预览静态目录
- 本地测试 `/site/` 映射是否正确
- 不依赖其他模块时快速启动一个最小服务

### 7.2 集成运行

集成运行时，上游模块可以直接复用 `server.Register()`，把静态目录挂到自己的 `http.ServeMux` 上。

这种设计的好处是：

- `static` 不强绑定自己的 HTTP 进程模型
- 上游可以自由组合 `/site/` 与其他路由
- 不会强制占用额外端口

不过当前仓内 `static` 自身并没有提供更高层的集成适配层，复用边界就是这个 `Register()` 函数。

## 8. 测试现状

### 8.1 `main_test.go`

[`main_test.go`](./main_test.go) 当前做的是接近集成测试的验证：

- 新建一个 `ServeMux`
- 调用 `server.Register(mux, "test-case")`
- 用 `httptest.NewServer` 启动 HTTP 服务
- 校验两个路径都能返回 200 和正确内容：
  - `/site/hello.html` -> `HELLO WORLD`
  - `/site/js/hello.js` -> `1+1`

这说明模块当前最关键的用户价值就是“路径映射正确并能把文件实际读出来”。

### 8.2 `server/server_test.go`

当前覆盖的行为包括：

- 有效目录可以成功注册。
- 不存在路径会返回错误。
- 传入文件路径而不是目录会返回错误。
- 注册后访问 `/site/hello.txt` 能得到 `200 OK`。

### 8.3 测试缺口

当前没有覆盖：

- 默认 `--site` 路径解析行为
- 默认 `--port=8080` 启动行为
- 启动日志内容
- 目录访问、索引文件、404 页面等标准库细节
- 高并发访问与大文件传输

## 9. 当前实现约束

### 9.1 只有 `/site/`，没有根路径托管

如果上游想把静态首页直接挂到 `/`，当前模块并不支持，需要自行封装或改动代码。

### 9.2 没有 SPA 路由回退

旧设计里容易让人联想到“给前端单页应用托管”，但当前代码并没有 `index.html` fallback。前端如果依赖 history 路由直达刷新，必须由上层额外处理。

### 9.3 默认目录依赖可执行文件位置

这让部署更可预测，但也意味着：

- 本地直接 `go run` 和打包后二进制运行时，默认目录语义可能与开发者直觉不完全一致。

### 9.4 完全依赖标准库默认文件服务

模块的优点是简单，但也意味着它没有对标准库行为做更细的产品化约束。目录列举、缓存语义、错误页表现都沿用标准库默认实现。

## 10. 演进建议

如果后续继续增强 `static`，比较自然的方向是：

1. 明确是否需要支持 `/` 根路径托管。
2. 如果主要服务前端 SPA，再补 `index.html` fallback。
3. 按部署需求增加缓存头、压缩或目录访问控制。
4. 如果要作为上层通用模块复用，可以把路径前缀做成可配置参数，而不是固定 `/site/`。

当前版本的 `static` 模块应该被理解为：

- 一个只做 `/site/` 前缀映射的最小静态文件服务
- 一个可被上游 mux 直接复用的薄封装

而不是完整的前端托管平台。
