### 第一性原则

+ 仅可以新增/更新/删除 integration（../..）同目录及其子目录，以及本需求直接涉及的 `../../../build.sh`、`../../../build/install.ps1`、`../../../build/USER_GUIDE.md`、`../../../build/USER_GUIDE.txt` 与 `../../../config/app/API.md`、`../../../config/app/CANVAS.md`、`../../../config/app/DESIGN.md`。

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md
+ 不新增外部依赖，不破坏既有 API 的字段兼容性。

### 需求介绍

扩展 Site 已使用的运行配置读取接口，使其可安全提供 `config/config.json` 的快捷回复配置，供居中对话的最后一条 SSE 响应展示快捷回复列表。

- `config/config.json.shortcut` 是可选快捷回复列表。服务端仅将其原始配置值透传给受控运行配置响应；列表的字符串过滤、去空白和去重由 Site 按页面展示规则处理，服务端不得写回、补写或修改该配置。
- `GET /api/runtime_config` 成功响应的 `config` 对象新增 `shortcut` 字段，字段不存在时保持省略；除该受控字段外，既有响应字段、状态码、错误格式和缓存语义保持不变。
- `shortcut` 必须与 `agent-dir`、`version` 等现有客户端运行字段一样，来自启动资源目录的 `config/config.json`。接口不得将 `provider`、模型密钥或其它未列入白名单的配置一并暴露。
- 运行配置读取失败时继续返回既有失败响应；`shortcut` 缺失、为空、非数组或包含非字符串值均不得导致接口异常。

### 编写代码
+ 最小范围更新，不新增外部依赖。
+ 仅扩展 `/api/runtime_config` 客户端配置白名单，不新增端点、持久化状态或配置写入路径；覆盖 `shortcut` 的透传、缺失兼容性，以及敏感的未白名单字段仍不暴露的自动化测试。

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`，说明 `shortcut` 的运行配置作用域和 `/api/runtime_config` 的受控透传语义。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求，不记录实现过程。
