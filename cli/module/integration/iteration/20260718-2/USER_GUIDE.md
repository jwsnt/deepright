# 迭代 20260718-2：从 Git 安装 Skill

在左侧 `Skill` 图标打开的「提炼 Skill」浮层中，点击「从Git安装」，输入技能仓库的 Git 地址并确认。页面会立即把配置好的安装文案发送到当前 Agent 会话。

## 配置

主应用 `config/config.json` 使用 `skills_git_install` 保存发送给 Agent 的文案。文案中必须包含 `$git_path`，页面会用输入框的原始文本替换全部同名变量。页面通过 Integration 的 `/api/runtime_config` 读取该字段，不会依赖浏览器或服务进程的当前目录：macOS 使用 `integration.app/Contents/Resources/config/config.json`，WSL 使用可执行文件同级的 `config/config.json`（默认 `~/deepright/config/config.json`）。

```json
{
  "skills_git_install": "请从 Git 仓库 $git_path 安装所有技能到当前 Agent 工作目录的`skills/`目录。"
}
```

页面不会改写、规范化或转义 Git 地址。若地址是 HTTP/HTTPS URL，它会在发送后的用户消息中显示为可点击 URL 气泡；SSH 等其他非空 Git 地址同样会原样发送。

## 异常提示

- Git 地址在去除首尾空白后为空时，不会发送。
- `skills_git_install` 缺失、为空、未包含 `$git_path`，或 `config/config.json` 无法读取时，不会发送，并会在当前浮层显示错误。

页面只处理模板读取、变量替换和自动发送。实际的 Git 克隆、Skill/资源安装、覆盖确认和结果输出由 Agent 根据文案完成。
