# SKILL 修复入口使用手册

## 功能说明

- 左侧虚拟文件系统中，只要某个文件或目录因为 `SKILL.md` 解析异常而进入红色高频闪动链路，条目右侧就会出现一个同色的修复小图标。
- 点击修复小图标后，会弹出确认浮层：
  - 标题：`是否需要修复SKILL.md`
  - 内容：本次准备修复的异常 `SKILL.md` 绝对路径
- 点击 `是` 后，Site 会自动向当前活跃会话发送以下请求：

```text
参考[https://agentskills.io/specification]修复<绝对路径>的错误。
```

## Go 子模块与 CLI

- 复用包：`site/skillrepairprompt`
- 主要方法：`skillrepairprompt.Build(path string) (string, error)`
- 独立 CLI：`go run ./cmd/skillrepairprompt --path /abs/path/to/SKILL.md`

示例输出：

```text
参考[https://agentskills.io/specification]修复/abs/path/to/SKILL.md的错误。
```
