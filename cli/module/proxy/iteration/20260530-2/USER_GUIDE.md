# Proxy 迭代手册（20260530-2）

本轮迭代继续扩展 `GET /install_app` 的自动探测范围，在保留现有取值和去重逻辑的前提下，新增 `python3` 缺失检测。

## 本次更新

- `/install_app` 继续保留原有 `git` 自动探测
- 当当前机器未安装 `python3` 时，返回结果会新增 `"python3"`
- `--install_app` 追加值仍会与自动探测结果统一去重合并

## 示例

查询：

```bash
curl http://127.0.0.1:9876/install_app
```

如果 `git` 和 `python3` 都未安装，可能返回：

```json
["git", "python3"]
```

如果启动时额外指定：

```bash
./proxy serve --agent-dir ./agents --install_app node,python,git,python3
```

则接口会在自动探测结果基础上继续合并这些自定义项，并保持去重后的返回。
