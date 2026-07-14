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
- 浮层第一排会先展示总览卡片：`本地总占用`、`会话落盘占用`、`实时估算占用`、`最大会话`、`运行时 CLI`。
- 第二部分统一改为 `会话空间汇总` 列表，不再展开 key 明细或当前会话字段级诊断。
- 每个会话只展示汇总后的空间结构与最后更新时间，重点用于快速排查“哪个会话占空间、主要占在哪一类”：
  - `正文`
  - `流记录`
  - `快照`
  - `元数据`
  - `运行CLI`
- 这里仍会明确把右侧 URL 预览归到轻量元数据，把运行时 `CLI` 归到“只展示、不持久化”。

## 兼容性说明

- 旧版 `localStorage` 中若已经落盘 `cliCommandHistory`，页面加载时会自动忽略并在后续保存时清掉。
- 服务端 `/api/restore`、live cli poll、聊天主区正文恢复、任务 badge 与 footer inline hint 恢复逻辑保持不变。
- 主手册 [../../USER_GUIDE.md](/Users/shenjiawei/Documents/code/deepright/cli/module/site/USER_GUIDE.md) 已同步补充这次本地持久化收口的总说明。

## 交付文件

- `REQUIREMENT.md`
- `USER_GUIDE.md`
