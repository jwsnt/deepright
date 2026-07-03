# Static Server 使用手册

## 简介

Static Server 为静态资源文件提供 HTTP 映射服务，将 `/site/` 路径映射到指定目录。作为 Proxy 模块的子模块，共用同一个 HTTP 服务；也可独立运行。

## 安装

```bash
# 独立运行
cd static && go build -o static-server .

# 作为 Proxy 子模块（无需额外安装，Proxy 已集成）
cd proxy && go build -o proxy .
```

## 独立运行

```bash
./static-server [--port <端口>] [--site <目录>]
```

### 参数说明

| 参数 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `--port` | 否 | `9876` | 监听端口 |
| `--site` | 否 | 启动目录下的 `site` | 静态资源目录绝对路径 |

### 示例

```bash
./static-server --site /var/www/html --port 8080
```

## 作为 Proxy 子模块

Proxy 启动时自动加载 static 子模块，通过 `--site` 参数指定目录：

```bash
./proxy --agent-dir ./agents --site ./site
```

### 代码集成

Proxy 内部通过 `proxy/static` 子包注册：

```go
import "proxy/static"

mux := http.NewServeMux()
static.Register(mux, "/path/to/site")
```

`Register` 函数签名：

```go
func Register(mux *http.ServeMux, siteDir string) error
```

## 访问方式

```
http://localhost:9876/site/hello.html
http://localhost:9876/site/js/app.js
http://localhost:9876/site/css/style.css
```

支持多层路径，HTML/JS/CSS/Image 等静态资源均可访问。

## 注意事项

- 目录不存在时报错（Proxy 模式下跳过，不影响代理功能）
- Content-Type 由 Go 标准库根据文件扩展名自动推断
- 不存在的文件返回 404
