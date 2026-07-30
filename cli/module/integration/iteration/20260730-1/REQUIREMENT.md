### 第一性原则

+ 仅可以新增/更新/删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../build.sh`、`../../../build/install.ps1`、`../../../build/USER_GUIDE.md`、`../../../build/USER_GUIDE.txt` 与 `../../../config/app/API.md`、`../../../config/app/CANVAS.md`、`../../../config/app/DESIGN.md`。

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
+ 不新增外部依赖，不破坏既有 API 的字段兼容性。

### 需求介绍

为 Site 的“构建迷你应用”入口受控透传主应用运行配置中的 `miniapp` 对象。

- 主应用资源目录 `config/config.json` 支持可选 `miniapp` 对象：`miniapp.build` 是 Site 发送构建请求的文案模板，模板使用 `$name` 表示用户输入的 CLI 名称或 Git 地址，使用 `$function` 表示功能描述；`miniapp.function` 是用户没有输入任何功能时的默认功能描述。默认配置为：

  ```json
  "miniapp": {
    "build": "请使用 @__internal_cli 为 $name 的 $function 构建迷你应用",
    "function": "全部功能"
  }
  ```

- `GET /api/runtime_config` 成功响应的受控 `config` 对象必须透传完整 `miniapp` 对象，以便 Site 在用户确认时读取最新模板和默认功能。接口只允许 `GET`，只读取主应用实际资源目录的 `config/config.json`，不得读取 Agent 工作目录的同名文件，不得补写、修改或持久化配置。
- `miniapp` 缺失、不是对象、字段缺失或字段类型无效时，接口仍保持既有成功响应与字段透传语义；Site 负责在发起构建请求前给出可见的配置校验错误。不得因为该字段扩大 `/api/runtime_config` 的白名单范围，`provider`、模型密钥和其它未授权字段仍不得暴露给浏览器。
- 此配置仅生成当前会话中的用户请求文案；Integration 不执行 CLI、拉取 Git 仓库、构建迷你应用或替换模板变量。

### 编写代码
+ 最小范围更新，不新增外部依赖。
+ 将 `miniapp` 加入既有运行配置白名单，并补充服务端测试，覆盖成功透传、主应用配置位置读取和未白名单字段继续不可见。

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`，说明 `miniapp` 的配置格式、运行配置作用域和 `/api/runtime_config` 的受控透传语义。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求，不记录实现过程。
