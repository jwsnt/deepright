# 20260503-1 使用手册

## 简介

本次迭代将 Site 设置页中的模型与密钥迁移到服务端 `/api/token`，页面不再把密钥保存到浏览器本地存储。

## 使用方式

1. 打开设置
2. 选择 Agent
3. 填写一组或多组模型与密钥
4. 点击保存

保存时，页面会向服务端发送：

```json
{
  "models": {
    "openai": "Bearer sk-openai",
    "kimi": "Bearer sk-kimi"
  }
}
```

再次打开页面或设置弹窗时，页面会通过 `GET /api/token` 重新拉取模型与密钥。

## 行为说明

- 页面本地只保存 `agentId`
- 模型与密钥不再写入浏览器 localStorage
- 聊天发送和右上角备忘录创建任务，都会使用 `/api/token` 拉回的同一份模型密钥
- 如果服务端没有任何模型密钥，页面会提示先进入设置完成配置

## 依赖

- 需要 proxy 或 integration 提供 `/api/token`
- 需要 `/api/agentId` 继续提供 Agent 列表
