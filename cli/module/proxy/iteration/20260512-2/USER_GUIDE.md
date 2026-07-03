# 迭代手册

## 本次变更

- `/v1/chat/completions` 注入的 metadata 中，`git` 字段改为每次请求实时获取
- 新增 `GET /install_app`
- 新增启动参数 `--install_app`

## 接口说明

### `GET /install_app`

返回当前机器待安装应用的字符串数组。

当前已支持的检测项：

- `git`

另外也支持通过启动参数追加自定义项目：

```bash
./proxy serve --agent-dir ./agents --install_app node,python,git
```

接口会把自动检测结果与该参数做去重合并。

示例：

```bash
curl http://127.0.0.1:9876/install_app
```

若 git 未安装，返回：

```json
["git"]
```

若当前支持的应用都已安装，返回：

```json
[]
```

若额外指定了 `--install_app node,python,git`，则可能返回：

```json
["git", "node", "python"]
```

## Metadata 行为

- `metadata.agents[].skills` 仍然实时扫描
- `metadata.git` 现在也会在每次 `/v1/chat/completions` 转发前实时探测
- 即使 `--agent-cache` 较长，git 路径变化也会在下一次请求中立即生效

## 验证方式

```bash
cd /path/to/deepright/cli/module/proxy
go test -run 'TestProxyInjectsMetadataAndPreservesFields|TestHandleInstallAppReturnsMissingGitWhenNeeded'
```
