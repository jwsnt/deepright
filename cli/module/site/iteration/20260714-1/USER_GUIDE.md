# Site 迭代 20260714-1 使用手册

## 本次更新

- 右侧 `CLI` 子任务历史不再写入浏览器 `localStorage`。
- 当前页内右侧 `CLI` 子任务仍按原有方式即时展示；页面刷新后，会继续通过 `/api/restore` 与 live poll 重建。
- 本地存储预算治理的压缩顺序同步更新：优先从持久化 payload 中直接丢弃 `cliCommandHistory`，再继续清理 `rawSse` 与其他冗余副本。
- 左侧 `最近` 右侧新增 `本地存储统计` 小图标，可直接打开浮层查看当前浏览器本地存储明细与多维分析。

## 右侧 CLI 子任务

- 当前页里实时 SSE、live cli poll 和 `/api/restore` 的展示逻辑不变，右侧 `CLI` 子任务仍会即时刷新。
- 页面刷新前，右侧面板展示依赖运行时内存中的 `cliCommandHistory`。
- 页面刷新后，右侧面板允许在 `/api/restore` 返回前短暂为空；恢复结果返回后，会继续把 `cli/get`、`cli/pub` 重建为右侧历史。
- 冷历史模式仍保持原规则：只恢复中心区最终态正文，不回放右侧 `CLI` 子任务历史。

## 本地存储治理

- `deepright_chats` 继续保存聊天正文和必要会话元数据，但不再长期保存右侧 `CLI` 子任务的完整输出镜像。
- 这次治理的重点是避免 `cliCommandHistory.output` 这类超大命令输出继续把单个 `chat` 顶到数 MB。
- 当本地存储空间紧张时，页面会先从持久化 payload 中移除 `cliCommandHistory`，再继续清理已完成 assistant 消息里的 `rawSse` 与思考快照；只有仍不够时，才继续裁剪较旧的正文窗口。
- 右侧 URL 预览 iframe 仍只保存轻量元数据，不会把 iframe 页面正文写入本地存储。

## 本地存储统计浮层

- `最近` 右侧的小统计图标会打开一个居中浮层，专门做浏览器本地存储诊断。
- 浮层会同时展示：
  - 当前 origin 的 `localStorage` 总体积
  - `deepright_chats` 已落盘体积
  - 按当前内存态重算后的 `deepright_chats` 体积
  - 最大单会话体积
  - 运行时 `CLI` 历史体积，并明确标注“未持久化”
- `localStorage Key 排名` 用来判断是否真的是 `deepright_chats` 占满空间，还是其它 key 也在增长。
- `会话占用排名` 用来快速定位最重的 chat，方便判断是正文、`rawSse`、快照还是其它元数据在膨胀。
- `当前会话拆解` 会把 `content`、`cliSubContent`、`rawSse`、`explodedSnapshots`、`explodedLogicalIds`、URL 预览元数据、chat 级元数据、scoped wrapper 与运行时 `CLI` 分别列出来。
- 这里会明确把右侧 URL 预览归类为“轻量元数据”，把运行时 `CLI` 归类为“只展示、不持久化”，避免再次把两者和本地持久化 payload 混淆。

## 兼容性说明

- 旧版 `localStorage` 中若已经落盘 `cliCommandHistory`，页面加载时会自动忽略并在后续保存时清掉。
- 服务端 `/api/restore`、live cli poll、聊天主区正文恢复、任务 badge 与 footer inline hint 恢复逻辑保持不变。
- 主手册 [../../USER_GUIDE.md](/Users/shenjiawei/Documents/code/deepright/cli/module/site/USER_GUIDE.md) 已同步补充这次本地持久化收口的总说明。

## 交付文件

- `REQUIREMENT.md`
- `USER_GUIDE.md`
