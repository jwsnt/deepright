# 飞书迭代手册（20260728-1）

## 需求目录

- 本迭代需求：[REQUIREMENT.md](REQUIREMENT.md)
- 飞书插件主需求：[../../REQUIREMENT.md](../../REQUIREMENT.md)
- 飞书插件主手册：[../../USER_GUIDE.md](../../USER_GUIDE.md)

## 严格防重回推

飞书任务完成后的结果以 `task_detail.id` 为唯一回推边界。系统会为该 detail 持久化一个固定的 RFC 4122 UUID，并将其传给 `feishu send` 和飞书 API；同一 detail 的文本、图片、文件分别使用固定的类别 UUID。重跑或周期任务会产生新的 detail，因此可以发送新的结果。

发送前只有一个进程能将任务从 `pending` 原子领取为 `sending`。成功时任务会标记为 `sent` 并写入 `replied_at`。如果飞书调用超时、失败，或飞书调用后本地状态无法确认，任务会进入 `unknown` 并停止自动发送；该状态需要人工确认，系统不会以新 UUID 重发。

升级时，历史上已经完成但没有 `replied_at` 的第三方任务会直接迁移为 `unknown`，因此不会因升级重新向飞书推送旧结果。

这保证系统不会自动重复推送同一 detail 的结果。飞书 UUID 的远端去重能力可用于处理同 UUID 的安全重试；若其去重范围不可确认或已过期，系统仍以 `unknown` 不自动重试为准，优先保证不重复发送。
