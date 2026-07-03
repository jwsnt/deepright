# 20260503-1 使用手册

## 简介

本次迭代为 Proxy 模块新增模型密钥持久化接口 `/api/token`，用于给 Site 设置页和其他模块统一保存、读取模型与密钥。

## 接口

### `GET /api/token`

返回当前已保存的全部模型与密钥：

```json
{
  "status": 0,
  "models": {
    "openai": "Bearer sk-openai",
    "kimi": "Bearer sk-kimi"
  },
  "updatedAt": {
    "openai": "2026-05-03T14:30:00+08:00",
    "kimi": "2026-05-03T14:30:00+08:00"
  }
}
```

### `POST /api/token`

保存或更新模型密钥：

```json
{
  "models": {
    "openai": "Bearer sk-openai",
    "kimi": "Bearer sk-kimi"
  }
}
```

也支持单条写法：

```json
{
  "model": "openai",
  "token": "Bearer sk-openai"
}
```

## 行为说明

- 模型名称是唯一键
- 不存在时新增，存在时更新
- 每次写入都会更新 `updated_at`
- 每次写入都会新增一条 `token_store_log`
- 数据保存在模块目录下的 SQLite `data` 文件中的 `token_store` 和 `token_store_log` 表

## 编译

```bash
cd /path/to/deepright/cli/module/proxy
/opt/homebrew/bin/go build -o proxy ./
```
