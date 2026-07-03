# Proxy 迭代 20260512-1 使用手册

## 本次新增

- 新增 HTTP 接口 `GET /skills_warning`
- 新增 CLI 命令 `proxy skills-warning`
- 服务启动后每分钟自动同步一次 `SKILL.md` 解析告警

## HTTP 用法

```bash
curl http://127.0.0.1:9876/skills_warning
curl http://127.0.0.1:9876/skills_warning?refresh=1
```

返回示例：

```json
{
  "status": 0,
  "data": [
    {
      "path": "/abs/path/SKILL.md",
      "reason": "description 字段无效",
      "time": 1747020000
    }
  ]
}
```

说明：

- 默认读取当前共享 sqlite `data` 中的 `skills_warning`
- `refresh=1` 会先立即扫描，再返回最新结果

## CLI 用法

```bash
cd /path/to/deepright/cli/module/proxy
./proxy skills-warning
./proxy skills-warning --agent-dir ./agent --refresh
./proxy skills-warning --refresh --root ./agent/custom-skills
```

## 同步规则

- 服务模式下，启动时会先同步一次
- 之后每分钟自动同步一次
- 默认扫描根目录优先使用当前 `--agent-dir/skills`
- 文件修复后，对应告警会在下一次同步时自动删除

更完整说明请参考上级手册 `../../USER_GUIDE.md`。
