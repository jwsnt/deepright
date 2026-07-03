# Integration 迭代手册（20260512-2）

本轮迭代把 `proxy/iteration/20260512-2` 的 `/install_app` 能力继续收口到 `integration`。

## 本次更新

- `integration` 的 `/install_app` 会继续自动探测当前机器是否缺少 `git`
- 新增启动参数 `--install_app`
- `--install_app` 支持以逗号分隔传入额外待安装应用
- `/install_app` 返回时会把自动探测结果与 `--install_app` 指定值做去重合并

## 示例

启动：

```bash
./integration --install_app node,python,git
```

查询：

```bash
curl http://127.0.0.1:8080/install_app
```

返回可能为：

```json
["git", "node", "python"]
```

如果 `git` 已安装，则自动探测部分不会重复追加，返回仍会保持去重后的结果。
