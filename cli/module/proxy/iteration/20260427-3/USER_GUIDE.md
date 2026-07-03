# 文件下载 API

## 接口

`GET /api/download?path=xxx`

## 参数

| 参数 | 类型 | 说明 |
|------|------|------|
| path | string | 文件或目录的绝对路径，支持大小写不敏感匹配 |

## 行为

- **文件**：直接流式下载，Content-Type 由文件扩展名自动推断
- **目录**：打包为 zip 后流式下载，文件名为 `目录名.zip`

## 响应头

```
Content-Disposition: attachment; filename="文件名"
```

目录下载时：
```
Content-Type: application/zip
Content-Disposition: attachment; filename="目录名.zip"
```

## 示例

```bash
# 下载文件
curl -O "http://127.0.0.1:8080/api/download?path=/Users/xxx/Documents/file.txt"

# 下载目录（自动打包为 zip）
curl -o photos.zip "http://127.0.0.1:8080/api/download?path=/Users/xxx/Documents/photos"
```

## 错误

- 400: path 参数缺失
- 404: 文件或目录不存在
- 405: 非 GET 请求
