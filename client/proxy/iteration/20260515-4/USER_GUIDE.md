# Proxy 迭代手册（20260515-4）

## 本次更新

- 新增 `GET /file/lastUpdate?file=xxx[&agentId=yyy]`
- 新增 `proxy file-last-update` CLI 子命令
- 支持绝对路径和相对路径两种文件定位方式

## 当前行为

1. `file` 为绝对路径时，会直接按文件系统路径解析，并支持大小写不敏感匹配
2. `file` 为相对路径时，会以当前 `agentId` 对应的 Agent workspace 为根目录解析
3. 相对路径场景下，`agentId` 为必填，也兼容 `agent`
4. 返回值为目标文件最后更新时间距离当前时间的毫秒数
5. 文件和目录都支持；`~` 路径不支持，`..` 越界路径会直接拒绝

## 示例

HTTP：

```bash
curl 'http://127.0.0.1:9876/file/lastUpdate?agentId=b&file=USER.md'
curl 'http://127.0.0.1:9876/file/lastUpdate?file=/abs/path/to/USER.md'
```

CLI：

```bash
./proxy file-last-update --agent b --file USER.md
./proxy file-last-update --file /abs/path/to/USER.md
```
