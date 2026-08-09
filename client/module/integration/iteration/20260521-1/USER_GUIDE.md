# 20260521-1 使用手册

## 目标

本次迭代将 `integration` 中待处理三方消息桥接为一次性任务明细的后台扫描频率调整为每30秒一次。

- 从原来的每分钟扫描一次改为每30秒扫描一次
- 命中可桥接的 `add-request` 后立即生成 `task_detail`
- 继续遵守 `integration` 的二进制收口原则，由 `integration` 主程序统一对外承接该能力

## 背景

`integration` 会收口 `connect`、`proxy`、`cron` 的相关能力。三方插件写入的待处理消息，会先进入 `add-request`，再桥接为一次性任务明细，后续进入统一的任务执行链路。

本次变更前，这条桥接流程与任务执行轮询共用同一个每分钟循环，因此待处理消息进入执行链路存在额外等待。

本次变更后，`integration` 将这条桥接流程拆分为独立后台同步任务：

- 服务启动后立即执行一次待处理消息扫描
- 之后每30秒再执行一次扫描
- 普通 cron 周期展开与普通 cron 执行循环保持原频率

## 行为说明

### 桥接扫描

- 后台每30秒扫描一次 `add-request` 待处理消息
- 命中后立即桥接生成 `task_detail`

### 收口边界

- 对外仍然只暴露 `integration` 主二进制
- `integration connect add-request ...` 与对应 connect 接口写入的消息，都会进入这条统一桥接链路
- 不需要用户额外启动或感知独立的 `proxy` / `connect`

### 不变项

- 消息聚合规则不变
- `meta_ref` / `META_ID` 语义不变
- `response_schema` 透传逻辑不变
- 普通 cron 元数据展开逻辑不变
- 普通 cron 到期执行逻辑不变

## 典型链路

1. 通过 `integration connect add-request ...` 或 connect HTTP 接口写入待处理消息
2. `integration` 后台在最多30秒内扫描到该消息
3. 生成对应的一次性 `task_meta` 与 `task_detail`
4. 后续任务继续沿用统一的 `/v1/chat/completions` 执行链路
5. 执行状态、日志、回推行为继续复用现有实现

## 影响

- `add-request` 到 `task_detail` 的等待时间明显缩短
- 三方消息更快进入统一任务执行链路
- 本次调整只优化桥接时延，不扩大普通 cron 行为变更范围

## 验证方式

可按以下方式验证：

```bash
cd /path/to/deepright/cli/module/integration
./integration serve --agent-dir ./agent --site ./site
```

然后写入待处理消息，例如：

```bash
./integration connect add-request --key feishu --content "hello"
```

观察：

- 最多约30秒内生成对应 `task_detail`
- 已生成任务的类型、内容、`meta_ref`、`response_schema` 与既有逻辑保持一致

## 说明

- 本文档只描述 `integration/iteration/20260521-1` 本轮新增的桥接频率调整
- 其他 CLI、HTTP、cron、metadata 与插件协议能力，继续以上级手册和既有迭代文档为准
