# 20260617-4 使用手册

## 目标

本次迭代只调整 `browser instance create` / `browser instance init` 在 Windows WSL / WSL2 下的受管 Chrome Profile 目录。

核心变化：

- WSL 下新的 `--user-data-dir` 不再使用 `C:\temp\chrome_<随机后缀>`
- 统一改为 `C:\ProgramData\deepright\chrome_<随机后缀>`
- 当该目录首次创建时，优先尝试从 `C:\ProgramData\deepright\chrome_def` 复制一份精简副本
- 如果 `chrome_def` 不存在，或复制过程中任意文件失败，则只记录日志并回退为空目录，不阻断实例启动

## help

```bash
./browser help
./browser instance help
```

说明：

- `help` 继续覆盖完整插件使用手册
- 手册里明确说明 WSL 下实例目录改为 `C:\ProgramData\deepright\chrome_<随机后缀>`
- 手册里明确说明新的 WSL 目录会 best-effort 从 `C:\ProgramData\deepright\chrome_def` 预置

## instance create

```bash
./browser instance create --agentId agent-a --chatId chat-001
```

行为：

- 在 Windows WSL / WSL2 下，仍通过插件同目录的 `browser_launcher.sh` 进入 `browser_instance_wsl`
- 成功响应仍保留原有 `profileDir` 字段
- `profileDir` 的真实值改为 `C:\ProgramData\deepright\chrome_<随机后缀>`
- 如果该目录原本不存在，则先尝试从 `C:\ProgramData\deepright\chrome_def` 复制一份精简副本
- 如果 `chrome_def` 不存在，或复制任意文件失败，则跳过复制，只记录日志，并继续使用空目录启动 Chrome
- 其余 `create` 逻辑不变，包括身份归一化、Chrome 路径解析、headless 规则和状态文件写入

## instance init

```bash
./browser instance init --agentId agent-a --chatId chat-001
```

行为：

- 仍然先尝试关闭旧实例，再重新创建新的有头实例
- 在 Windows WSL / WSL2 下，新的实例目录同样改为 `C:\ProgramData\deepright\chrome_<随机后缀>`
- 如果目录首次创建，则与 `create` 共用同一套 `chrome_def` 复制逻辑
- 如果 `chrome_def` 缺失或复制失败，仍然只记日志，不阻断 `init`
- 其余 `init` 逻辑不变，包括 ready 检查、状态写入和返回结构

## 总结

- WSL 下实例 Profile 根目录从 `C:\temp` 切换到 `C:\ProgramData\deepright`
- 新目录会优先复用 `chrome_def` 作为预置模板
- 模板缺失或复制失败时，语义是“记录日志并继续”
