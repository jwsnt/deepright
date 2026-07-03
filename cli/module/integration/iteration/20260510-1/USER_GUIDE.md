# Integration 迭代 20260510-1 使用手册

## 变更说明

本次迭代按 `DESIGN.md` 的单一实现原则，对以下需求进行了 integration 收口：

- `knowledge` 模块：知识库目录与 `lastUpdate` 元数据
- `knowledge` 模块 20260510-1：新增 `update-time` CLI 命令
- `agent` 模块 20260510-1：Agent 元数据新增可选 `knowledge` 字段
- `proxy` 模块 20260510-1：转发请求 metadata 增加 `knowledge`

本次收口不是在 `integration` 里重新实现一份 knowledge 逻辑，而是让 `integration` 入口层直接复用共享的 Agent 元数据内核输出。

## 收口范围

当前以下链路都会统一带上同一份 Agent 元数据：

- `/v1/chat/completions`
- `cli/get`
- `cli/pub`
- integration 内部发起的 cron 执行请求

也就是说，knowledge 字段一旦满足输出条件，会在这些链路里保持一致。

除此之外，Knowledge 的更新时间写入命令也被统一收口到 `integration` 主 CLI：

```bash
integration knowledge update-time 1715337600000
integration knowledge update-time --timestamp 1715337600000
```

该命令底层仍复用共享 `knowledge` 模块的 sqlite 与状态逻辑，不在 `integration` 内复制实现；其中 `lastUpdate` 统一按 Unix 毫秒时间戳写入和输出。

## `knowledge` 字段结构

当应用启动目录下存在知识库时，metadata 中会额外包含：

```json
{
  "knowledge": {
    "path": "/app/knowledge",
    "lastUpdate": 0
  }
}
```

字段含义：

- `path`
  - 知识库绝对路径
  - 固定解析为 `<app-dir>/knowledge`
- `lastUpdate`
  - 最后更新时间
  - 来自 `<app-dir>/data` 中 `knowledge_runtime.last_update`
  - 若目录已存在但尚未写入更新时间，则为 `0`

## 输出规则

- 如果 `<app-dir>/knowledge` 不存在：
  - metadata 中不会出现 `knowledge`
- 如果 `<app-dir>/knowledge` 已存在，但 `<app-dir>/data` 不存在：
  - 仍会输出 `knowledge`
  - `lastUpdate = 0`
- 如果目录和数据都存在：
  - 输出真实的 `lastUpdate`

## `app-dir` 来源

integration 不会自己猜另一套 knowledge 路径规则，而是复用已有运行时约定。

优先级如下：

1. `runtime.json` 中的 `app-dir`
2. `runtime.json` 中的 `app`
3. 当前工作目录

因此：

- HTTP 服务模式启动后，`runtime.json` 会成为 knowledge 路径解析的主要依据
- 测试或独立运行场景下，如果没有 `runtime.json`，则回退到当前工作目录

## 设计说明

本次实现遵守了以下设计要求：

- 单一实现原则
  - knowledge 字段的判定与输出逻辑仍在共享内核中维护
- 入口薄包装原则
  - integration 仅负责把入口接到共享内核，不在入口层复制业务实现
- 二进制收口原则
  - 最终用户仍然只感知 `integration`
  - `cli/get`、`cli/pub`、HTTP 转发链路都通过 `integration` 统一暴露

## 验证方式

可以重点验证这三类行为：

1. 启动目录没有 `knowledge` 时，metadata 不包含 `knowledge`
2. 启动目录有 `knowledge` 时，`/v1/chat/completions` 的 metadata 包含 `knowledge`
3. `cli/get` 与 `cli/pub` 提交的 metadata 中，同样包含相同的 `knowledge`
4. `integration knowledge update-time 时间戳` 会更新 `<app-dir>/data` 中的 `knowledge_runtime.last_update`

更完整的启动方式、runtime 规则和命令说明，请查看上级手册：

- `../../USER_GUIDE.md`
