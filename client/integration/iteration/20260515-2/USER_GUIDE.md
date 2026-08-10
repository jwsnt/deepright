# Integration 迭代手册（20260515-2）

本轮迭代把 `proxy/iteration/20260515-3` 的知识库更新时间门控正式收口进 `integration`。

## 本次更新

- `integration` 自己的 `/v1/chat/completions` 转发链路现在会在转发前检查 `metadata.knowledge.lastUpdate`
- 当知识库最近刚更新过时，不再重复把 `lastUpdate` 提交给上游
- 当某个请求刚触发过一次知识库更新申请后，锁窗口内的后续请求也不会再次提交 `lastUpdate`

## 新增参数

- `--knowledge_update_interval=7200000`
- `--knowledge_update_lock=1800000`

含义分别为：

- 知识库更新时间阈值：2 小时
- 知识库更新申请锁窗口：30 分钟

## 行为规则

- 如果 `knowledge.lastUpdate` 距离当前请求时间未超过 `knowledge_update_interval`，则转发前删除 `lastUpdate`
- 如果已超过该时间，则继续检查共享 sqlite 中最近一次“申请知识库更新”的时间
- 如果最近申请时间距离当前请求未超过 `knowledge_update_lock`，同样删除 `lastUpdate`
- 只有知识库已过期，且锁窗口也已过期时，才保留 `lastUpdate` 原样转发，并把当前请求时间写入共享 sqlite 的 `knowledge_update_lock`

## 作用

这样 `integration` 作为最终交付二进制时，就不会因为内部仍保留一份旧代理实现，导致并发或连续请求重复触发知识库刷新。
