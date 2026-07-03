# Integration 迭代手册（20260530-2）

本轮迭代把 `proxy/iteration/20260530-2` 中 `/install_app` 的 `python3` 缺失探测继续收口到 `integration` 主二进制。

## 本次更新

- `integration` 的 `/install_app` 保留现有 `git` 自动探测
- 当当前机器未安装 `python3` 时，返回结果会新增 `"python3"`
- `--install_app` 追加项仍会和自动探测结果统一去重合并

## 示例

查询：

```bash
curl http://127.0.0.1:8080/install_app
```

如果 `git` 和 `python3` 都未安装，可能返回：

```json
["git", "python3"]
```

如果启动时使用：

```bash
./integration --install_app node,python,git,python3
```

则接口会在自动探测结果基础上合并这些自定义项，并保持去重后的输出。
