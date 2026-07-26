# Integration 迭代手册（20260726-9）

## 图片另存为 PNG

图片编辑不新增专用接口，继续使用受限的工作区写入接口：

```text
POST /api/edit?agentId=<agentId>&path=images/<filename>.png&saveAsNew=true
```

请求体中的 `content` 是 PNG 的 Base64：

```json
{
  "content": "iVBORw0KGgo..."
}
```

- `path` 必须是当前 Agent 工作区内的相对 `images/<filename>.png` 路径。服务端会自动创建缺失的 `images/` 目录。
- `saveAsNew=true` 由服务端在扩展名前追加高精度时间戳，避免覆盖已有图片；响应 `savedAs` 返回最终文件的系统绝对路径，供页面复制到系统剪贴板。
- PNG Base64 会按二进制解码后写入，不经过文本编码或图片转码。原图与历史图片不会被修改。
- 绝对路径、`~`、`..`、目录目标、跨 Agent 路径和工作区外符号链接都会被拒绝，失败时不会在工作区外创建文件。
