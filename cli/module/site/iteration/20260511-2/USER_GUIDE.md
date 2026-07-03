# 20260511-2 User Guide

## 目标

本次迭代将右侧知识库标题中的“最近刷新”时间彻底切换为读取同源 `/knowledge_lastUpdate`，不再使用前端本地 Mock 时间。

## 行为

1. 页面进入知识库区域时，会请求 `/knowledge_lastUpdate`。
2. 切换知识库路径时，会同步刷新该时间。
3. 点击知识库刷新按钮回到首页时，也会重新读取 `/knowledge_lastUpdate`。
4. 标题栏在接口返回前显示 `最近刷新 --`，避免短暂展示错误的假时间。

## 说明

- `/knowledge_lastUpdate` 由 `proxy` / `integration` 提供
- 当前页面直接展示接口返回的时间文本
- 更完整说明请继续参考上级手册 `../../USER_GUIDE.md`
