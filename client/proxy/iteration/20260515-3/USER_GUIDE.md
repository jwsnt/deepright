# Proxy 迭代手册（20260515-3）

本轮迭代为 `/v1/chat/completions` 转发补上了知识库更新时间门控，避免并发请求同时触发知识库刷新。

## 新行为

- Proxy 在转发前会检查 `metadata.knowledge.lastUpdate`
- 如果 `lastUpdate` 距离当前请求时间未超过 `--knowledge_update_interval`，则转发前删除 `lastUpdate`
- 如果已超过该时间，则继续检查共享 sqlite 中最近一次“申请知识库更新”的时间
- 如果该申请时间距离当前请求未超过 `--knowledge_update_lock`，同样删除 `lastUpdate`
- 只有知识库已过期且锁窗口也已过期时，才保留 `lastUpdate` 原样转发，并把当前请求时间写入 sqlite

## 默认参数

- `--knowledge_update_interval=7200000`
- `--knowledge_update_lock=1800000`

分别对应：

- 知识库更新时间阈值：2 小时
- 知识库更新申请锁：30 分钟

## 作用说明

服务端仍通过是否收到 `metadata.knowledge.lastUpdate` 判断要不要刷新知识库。

因此本轮修改后的效果是：

- 近期刚更新过知识库时，不再重复触发刷新
- 某个请求刚申请过刷新后的 30 分钟内，其它并发或后续请求不会再次触发刷新
- 超过锁窗口后，下一次真正过期的请求才会重新带上 `lastUpdate`
