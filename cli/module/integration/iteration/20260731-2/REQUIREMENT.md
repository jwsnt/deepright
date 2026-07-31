### 第一性原则
+ 仅可以新增/更新/删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../site/index.html`、`../../../build.sh`、`../../../build/install.ps1`、`../../../build/USER_GUIDE.md`、`../../../build/USER_GUIDE.txt` 与 `../../../config/app/API.md`、`../../../config/app/CANVAS.md`、`../../../config/app/DESIGN.md`。

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
+ 不新增外部依赖，不改变既有 `/api/files`、`/api/runtime_config` 或普通消息发送协议。

### 需求介绍
+ Site 的“构建迷你应用”浮层在功能列表后新增唯一的“参考文档（可选，仅 .md）”字段；该字段不是可重复列表，至多保存一个 `.md` 文件的文件系统绝对路径。字段默认收起，使用小型虚拟文件系统图标打开目录选择器，或使用展开按钮后直接粘贴/输入绝对路径。目录浏览器从当前会话 Agent 的工作区根目录逐层进入目录，只显示目录与 `.md` 文件；目录只用于浏览，不能作为参考文档；选取文件后必须回填其绝对路径。非 `.md` 的手动路径必须拒绝发送。
+ 参考文档选择器只读取既有 `/api/workspace` 与 `/api/files`，不上传、复制、修改或删除文件；读取失败时在浮层中显示错误。关闭构建浮层、点击遮罩或按 `Esc` 时保留未提交的参考文档草稿，成功发送后与其它构建草稿一起清空。
+ 主应用静态 `config/config.json` 的 `miniapp` 对象新增 `reference` 字符串，并将 `build` 模板扩展为可含 `$reference`。发布默认值为：

```json
"miniapp": {
  "build": "请使用 [SKILL:__internal_cli] 为 $name 的 $function 构建迷你应用 $reference",
  "reference": "（READ_ME.md: $reference）",
  "function": "全部功能",
  "recover": 30
}
```

+ Site 在确认时重新读取最新的 `miniapp` 配置，继续替换全部 `$name` 与 `$function`。若用户填写了非空参考文档路径，`miniapp.reference` 必须为非空字符串：先将其中全部 `$reference` 替换为用户填写的绝对路径，再将所得结果替换 `miniapp.build` 中全部 `$reference`。若未填写参考文档，则将 `miniapp.build` 中全部 `$reference` 替换为空字符串，不要求或读取 `miniapp.reference`。配置缺失、类型错误或模板变量不满足上述规则时，必须在浮层显示可见错误，且不得发送构建请求。
+ 本功能仅生成当前会话中的用户请求文案；浏览器不会直接读取参考文件内容、执行 CLI、拉取 Git 仓库或构建迷你应用。

### 编写代码
+ 最小范围更新，不新增外部依赖。
+ 更新构建浮层的表单、草稿状态和键盘/遮罩关闭行为；选择器的目录层级、排序与文件图标应与左侧虚拟文件系统保持一致，并且不得影响左侧虚拟文件系统当前目录、选中项或刷新状态。
+ 扩展构建请求模板校验与替换逻辑，覆盖“有参考文档”“无参考文档”“reference 模板缺失或无效”以及全部占位符替换；补充运行配置透传测试，确保 `miniapp.reference` 会按既有完整对象透传而不暴露非白名单配置。

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`

### 其他要求
+ `REQUIREMENT.md` 仅描述需求，不记录实现过程。
