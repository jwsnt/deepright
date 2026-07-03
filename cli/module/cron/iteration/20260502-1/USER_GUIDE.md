# 20260502-1 使用手册

## 简介

本迭代为 Cron 的任务明细 `TaskDetail` 新增了 `chatId` 字段，类型为字符串，可为空。

## 变更说明

- `TaskDetail` 结构新增 `chatId`
- `task_detail` 表新增 `chat_id TEXT NOT NULL DEFAULT ''`
- 旧库启动时会自动执行补列迁移
- 新创建的任务明细默认写入空字符串
- 读取任务明细列表时会一并返回 `chatId`

## 数据行为

- `chatId` 允许为空
- 如果调用方没有写入会话标识，Cron 会保持 `chatId=""`
- 该字段用于供上层模块在后续执行链路中复用指定会话

## 代码位置

- 结构体定义：`module/cron/main.go`
- 建表与迁移：`module/cron/main.go`
- 明细创建默认值：`module/cron/main.go`
- 明细查询返回：`module/cron/main.go`

## 样例

任务明细返回示例：

```json
{
  "id": 1,
  "metaId": 10,
  "execTime": 1777702200,
  "agentId": "alpha",
  "chatId": "",
  "model": "openai",
  "thinking": true,
  "content": "执行任务",
  "started": 0
}
```

## 检查结论

- 需求已完成
- 当前环境未安装 `go`，因此本次未能现场执行 `go test ./...`
- 已通过代码实现与现有测试用例内容交叉检查确认本轮变更已经落地
