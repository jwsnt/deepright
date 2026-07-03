# Proxy 迭代手册（20260516-3）

## 本次更新

- 新增 `proxy token` CLI 命令
- 支持直接读取设置中已经保存的全部模型与密钥
- 支持按模型名读取单个密钥

## CLI 用法

```bash
cd /path/to/deepright/cli/module/proxy
./proxy token
./proxy token --provider deepseek
```

输出示例：

```json
[{"deepseek":"aaa"},{"kimi":"bbb"}]
```

```json
{"deepseek":"aaa"}
```

## 当前行为

1. `proxy token` 会读取当前模块目录下共享 SQLite `data` 中保存的模型与密钥
2. 不带 `--name` 时，输出按模型名排序后的 JSON 数组
3. 带 `--provider MODEL` 时，只输出指定 provider 的单个 JSON 对象
4. 如果指定模型不存在，则输出空对象 `{}`
5. 该命令是只读 CLI，不会改写 `token_store` 或 `proxy_agent_provider_log`

## 说明

- 这个命令适合在不通过 HTTP `/api/token` 的情况下，直接查看当前 `proxy` 启动目录下已经保存的模型密钥
- 更完整说明请继续参考上级手册 `../../USER_GUIDE.md`
