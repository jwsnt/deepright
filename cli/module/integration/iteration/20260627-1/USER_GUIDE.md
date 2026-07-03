# Integration 迭代 20260627-1 使用手册

## 变更说明

- 新增静态资源映射路径 `GET /app/$agentId/*`
- 映射根目录固定为 `--agent-dir/$agentId/app/`
- 适合直接托管静态生成的 HTML 页面及其依赖资源

目录关系示例：

```text
agent/
├── agent-a/
│   └── app/
└── agent-b/
    └── app/
```

例如：

- `--agent-dir=/Users/demo/runtime/agent`
- 则 `GET /app/agent-a/index.html` 对应本地文件 `/Users/demo/runtime/agent/agent-a/app/index.html`

## 使用方式

启动 `integration` 后，直接通过浏览器或 HTTP 请求访问：

```bash
curl http://127.0.0.1:8080/app/agent-a/
curl http://127.0.0.1:8080/app/agent-a/index.html
curl http://127.0.0.1:8080/app/agent-a/assets/main.js
```

## 约束

- 仅支持 `GET` / `HEAD`
- 请求路径必须包含 `agentId`
- 请求路径禁止跳出 `app` 根目录，包含 `..` 时返回 `400`
- `app` 目录不存在时返回 `404`
- `app` 目录后续新增或更新文件后，无需重启 `integration`，可直接通过 `/app/$agentId/*` 访问
