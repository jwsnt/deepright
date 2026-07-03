# 20260516-2 User Guide

## 目标

本次迭代为 `proxy` 新增了知识库真实路径查询接口 `/knowledge_path`。

- 接口返回当前应用知识库目录的真实文件系统绝对路径
- 路径规则与 `knowledge` 模块保持一致
- 可由 `site` 或其他同源模块直接调用，用于展示知识库目录绝对路径

## 使用方式

启动 `proxy` 后，请求：

```bash
curl http://127.0.0.1:9876/knowledge_path
```

返回示例：

```text
/abs/path/to/knowledge
```

## 说明

- 仅支持 `GET`
- 根目录固定解析为当前应用启动目录下的 `knowledge`
- 如果知识库目录不存在，返回 `404`
- 如果知识库路径存在但不是目录，返回 `500`
- 更完整说明请继续参考上级手册 `../../USER_GUIDE.md`
