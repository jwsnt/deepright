# Proxy 迭代 20260511-1 使用手册

本次迭代为 `proxy` 新增了 `GET /knowledge` 静态映射接口，用于直接访问当前应用启动目录下的 `knowledge` 目录内容。

## 接口行为

- `GET /knowledge`
  - 返回 `knowledge` 根目录的树形结构
- `GET /knowledge/<相对路径>`
  - 如果路径指向目录，返回该目录的树形结构
  - 如果路径指向文件，直接返回文件原始内容

该接口的根目录固定解析为：

```text
<app-dir>/knowledge
```

其中 `app-dir` 与当前 `proxy` 的其余运行时语义保持一致：

1. 优先读取当前工作目录下 `runtime.json` 的 `app-dir`
2. 其次读取 `runtime.json` 的 `app`
3. 否则回退到当前工作目录

## 返回示例

目录结构响应示例：

```text
knowledge/
|-- README.md
`-- docs/
    `-- guide.txt
```

文件内容响应示例：

```text
hello knowledge
```

## 约束

- 仅支持 `GET`
- 如果 `knowledge` 目录不存在，返回 `404`
- 如果访问路径尝试跳出 `knowledge` 根目录，例如 `..`，返回 `400`
- 文件访问不会再额外包一层 JSON，而是按静态资源方式直接输出原始内容

## 典型用法

```bash
curl http://127.0.0.1:9876/knowledge
curl http://127.0.0.1:9876/knowledge/README.md
curl http://127.0.0.1:9876/knowledge/docs/guide.txt
```
