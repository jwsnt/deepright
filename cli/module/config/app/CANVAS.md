# Canvas Format API

`.canvas` 是 DeepRight 无限画布的可编辑、版本化 JSON 格式。它用节点承载信息，用连线表达信息之间的关系；同一份文件既可由画布编辑器打开，也可由 AI 直接生成后写入 Agent 工作区。

## 生成原则

AI 生成画布时，先将用户意图拆成以下四层，并与用户对齐：

| 层次 | 在画布中的表达 | 用于澄清的问题 |
|---|---|---|
| 主题 | `title` | 这张图要回答什么问题？ |
| 对象 | `nodes` | 有哪些人、系统、阶段、决策或资料？ |
| 关系 | `edges` | 谁影响谁、先后如何、是否双向、关系说明是什么？ |
| 阅读顺序 | 节点坐标与连线方向 | 读者应从哪里开始，按什么路径理解？ |

建议默认采用“从左到右的因果/流程，或从上到下的层级/拆解”布局。坐标描述空间位置，不会改变连线语义；**关系语义由 `from`、`to`、`arrow` 与 `note` 共同决定**。

## 格式总览

| 属性 | 类型 | 必填 | 作用 |
|---|---:|:---:|---|
| `version` | integer | 是 | 格式版本。当前写入值为 `2`。 |
| `title` | string | 是 | 画布标题，显示在编辑器顶部；不会改动文件名。 |
| `viewport` | object | 是 | 上次保存时的视口位置与缩放，决定再次打开时的初始视野。 |
| `nodes` | array | 是 | 信息元素集合：便签、引导卡、文本或图片。 |
| `edges` | array | 是 | 节点之间的有向、反向或双向关系。 |

最小有效文件：

```json
{
  "version": 2,
  "title": "未命名画布",
  "viewport": { "x": 0, "y": 0, "scale": 1 },
  "nodes": [],
  "edges": []
}
```

推荐始终输出完整五个顶层字段。编辑器会尽力兼容缺失字段或旧数据，但 AI 不应依赖该回退行为。

## 顶层对象

```ts
interface CanvasDocument {
  version: 2;
  title: string;
  viewport: CanvasViewport;
  nodes: CanvasNode[];
  edges: CanvasEdge[];
}
```

### `version`

用途：标识序列化格式版本。当前版本为 `2`，生成新文件时固定写为 `2`。

兼容性：读取旧文件时，编辑器会兼容缺少 `version` 或缺少连线 `arrow` 的内容；旧连线默认按 `forward` 处理。生成端仍应写入当前版本和全部字段，避免语义依赖兼容逻辑。

### `title`

用途：描述画布要表达的主题，适合作为用户确认关系思路时的总问题或结论，例如“订单履约异常处理流程”“小程序增长漏斗”。

| 规则 | 说明 |
|---|---|
| 类型 | string |
| 建议长度 | 1–60 个字符，便于在编辑器标题栏阅读。 |
| 实际上限 | 240 个字符。 |
| 默认值 | 新建画布使用文件名去掉 `.canvas` 的结果。 |
| 与文件名关系 | 可独立修改；保存时不改文件路径或文件名。 |

### `viewport`

用途：记录画布世界坐标到屏幕的变换，使再次打开文件时保留阅读视角。它不是节点位置，也不参与关系计算。

```ts
interface CanvasViewport {
  x: number;     // 视口水平平移，单位为屏幕像素
  y: number;     // 视口垂直平移，单位为屏幕像素
  scale: number; // 缩放比例
}
```

| 参数 | 可配置范围 | 说明 |
|---|---|---|
| `x` | 有限数字 | 画布在屏幕上的水平偏移；无固定边界。 |
| `y` | 有限数字 | 画布在屏幕上的垂直偏移；无固定边界。 |
| `scale` | `0.2`–`3` | 缩放比例，超出范围会被限制。`1` 表示 100%。 |

保存时，`x`、`y` 最多保留两位小数，`scale` 最多保留四位小数。若 AI 不需要指定打开视野，使用 `{ "x": 0, "y": 0, "scale": 1 }` 即可；对于关系图，建议让主要内容围绕原点分布，便于用户继续编辑。

## 节点 `nodes`

节点是画布的基本信息单元。每个节点都可被选择、拖动、多选和删除；删除节点时，与它相连的边也会被删除。

```ts
type CanvasNode = CardNode | GuideNode | TextNode | ImageNode;

interface NodeBase {
  id: string;
  type: "card" | "guide" | "text" | "image";
  x: number;
  y: number;
  width: number;
  height: number;
}
```

### 通用参数

| 参数 | 必填 | 作用与约束 |
|---|:---:|---|
| `id` | 是 | 节点唯一标识，供 `edges[].from` / `to` 引用。推荐 UUID 或稳定的语义 ID；同一文件内必须唯一，且不得为空。 |
| `type` | 是 | 节点类型：`card`、`guide`、`text`、`image`。未知类型读取时会按 `card` 兼容。 |
| `x` | 是 | 节点左上角的世界坐标 X，有限数字。向右增大。 |
| `y` | 是 | 节点左上角的世界坐标 Y，有限数字。向下增大。 |
| `width` | 是 | 宽度，单位为世界坐标像素；读取时限制为 `100`–`900`。 |
| `height` | 是 | 高度，单位为世界坐标像素；读取时限制为 `42`–`900`。 |

坐标可以为负数。生成关系图时，建议预留节点之间的空白：横向流程可从 `x = -600` 起，节点间隔约 `100`–`180`；纵向层级可使用 `y` 间隔约 `100`–`160`。连线会自动从节点边界连接，不需要提供锚点或路径。`width` / `height` 是可由 AI 写入的存储参数；当前编辑器没有拖拽缩放节点的界面控件。

### `card`：便签/信息卡

作用：承载一个概念、事项、阶段、角色、结论或简短说明。它是 AI 生成结构化关系图时的默认节点类型。

```ts
interface CardNode extends NodeBase {
  type: "card";
  content: string;
  color?: "#RRGGBB";
}
```

| 参数 | 作用 | 配置规则 |
|---|---|---|
| `content` | 节点正文，可换行 | 字符串；读取时最多 50,000 个字符。建议保持在 1–6 行，方便阅读。 |
| `color` | 自定义背景色 | 可选，格式必须为六位十六进制 `#RRGGBB`。未设置时跟随主题默认便签色。 |
| 默认尺寸 | 阅读型便签 | 新建默认 `260 × 142`。 |

能力：可直接编辑文字、拖动、框选、设置/恢复背景颜色、作为连线的起点或终点。

### `text`：独立文本

作用：表达不应被卡片边框强调的信息，例如分区标题、泳道标签、图例、注释或关键结论。

```ts
interface TextNode extends NodeBase {
  type: "text";
  content: string;
  color?: "#RRGGBB";
}
```

| 参数 | 作用 | 配置规则 |
|---|---|---|
| `content` | 文本内容，可换行 | 字符串；读取时最多 50,000 个字符。 |
| `color` | 可选背景色 | 未设置时为透明背景；设置后显示圆角色块，并自动选择高对比文字颜色。 |
| 默认尺寸 | 轻量文字 | 新建默认 `220 × 54`。 |

能力与 `card` 相同，但视觉上更轻。AI 可用它标出“输入”“处理”“输出”或“战略层 / 执行层”等分区；不要用大量 `text` 替代核心信息卡，否则关系层次会变弱。

### `guide`：引导卡

作用：新建空画布中的可删除使用说明。数据结构与 `card` 相同，但用于教学而非业务内容。

```ts
interface GuideNode extends NodeBase {
  type: "guide";
  content: string;
  color?: "#RRGGBB";
}
```

新建画布默认包含一个 `260 × 142` 的 `guide` 节点，内容提示拖动画布、滚轮缩放和从工具栏添加内容。AI 生成面向用户的正式关系图时，通常应**不输出 `guide`**，或用业务 `card` 替换它；它不具备额外业务语义。

### `image`：图片

作用：把截图、示意图、照片、图标板或视觉证据嵌入关系图。图片节点可拖动、选择、连线，并会随 `.canvas` 一起保存，重新打开后仍可编辑。

```ts
interface ImageNode extends NodeBase {
  type: "image";
  src: string; // data:image/...;base64,...
}
```

| 参数 | 作用 | 配置规则 |
|---|---|---|
| `src` | 图片内容 | 必须是以 `data:image/` 开头的 Data URL，例如 `data:image/png;base64,...`。不能使用本机路径、HTTP URL 或 `file://` URL。 |
| 默认尺寸 | 图片框 | 通过编辑器新增时为 `260 × 180`；AI 应始终显式提供宽高。 |
| `color` / `content` | 不适用 | 图片节点不保存背景色或文本内容。 |

不合法、缺失或非 `data:image/` 的 `src` 会使该图片节点在读取时被跳过。由于 Base64 会显著增大文件，生成时应仅在图片确实帮助理解关系时使用，优先采用压缩后的 PNG/JPEG Data URL。

### 颜色规则

- `color` 仅适用于 `card`、`guide`、`text`。
- 允许格式为 `#RRGGBB`（大小写均可，保存时规范为小写）；不支持短写 `#RGB`、透明色、渐变、RGBA 或 CSS 色名。
- 未设置 `color` 时，`card` / `guide` 使用当前主题的默认颜色；`text` 保持透明。
- 自定义背景色后，编辑器会根据亮度自动使用深色或浅色文字，确保可读性。

## 连线 `edges`

连线表达节点间关系，不承载独立内容。编辑器会以贝塞尔曲线自动绘制，不支持手动控制锚点、折点、线宽或颜色。

```ts
interface CanvasEdge {
  id: string;
  from: string;
  to: string;
  arrow: "forward" | "reverse" | "both";
  note: string;
}
```

| 参数 | 必填 | 作用与约束 |
|---|:---:|---|
| `id` | 是 | 连线唯一标识。推荐 UUID 或稳定的语义 ID。 |
| `from` | 是 | 起点节点 ID，必须引用 `nodes` 中的节点。 |
| `to` | 是 | 终点节点 ID，必须引用 `nodes` 中的另一节点。不可与 `from` 相同。 |
| `arrow` | 是 | 箭头方向：`forward`、`reverse`、`both`。缺失或非法值读取时为 `forward`。 |
| `note` | 是 | 关系说明，最多 240 个字符；空字符串表示无备注。 |

箭头语义以 `from = A`、`to = B` 为例：

| `arrow` | 可视化 | 表达的关系 |
|---|---|---|
| `forward` | `A → B` | A 流向、影响、依赖或导致 B；默认值。 |
| `reverse` | `A ← B` | B 流向、影响、依赖或导致 A。 |
| `both` | `A ↔ B` | A 与 B 双向作用、协同、反馈或互相依赖。 |

`note` 用于补足箭头无法表达的关系类型，例如“审批”“调用 API”“异常回流”“每周同步”“优先级更高”。已有备注会常驻显示在连线中部；空备注在悬停或选中连线时可添加。对 AI 而言，建议：

- 流程图：每条关键边写动词，如“提交”“校验”“派单”“通知”。
- 架构图：写接口、协议或依赖类型，如“HTTPS”“读取”“发布事件”。
- 因果图：写关系强度或条件，如“当库存不足时”“提高转化率”。
- 简单顺序关系可留空，避免画面被重复文字淹没。

读取时，端点不存在、自环（`from === to`）的连线会被忽略。编辑器创建连线时也会避免重复的同向端点组合；AI 应主动保证 `id` 唯一、端点有效，且不要输出语义完全重复的边。

## 完整示例：电商退款处理关系图

下面示例展示一份可直接保存为 `退款处理.canvas` 的文档。它把对象、动作与异常反馈分别作为节点和边表达：

```json
{
  "version": 2,
  "title": "电商退款处理关系图",
  "viewport": { "x": 80, "y": 90, "scale": 0.9 },
  "nodes": [
    {
      "id": "customer-request",
      "type": "card",
      "x": -520,
      "y": 40,
      "width": 220,
      "height": 110,
      "content": "用户\n提交退款申请",
      "color": "#ffe0ad"
    },
    {
      "id": "order-check",
      "type": "card",
      "x": -180,
      "y": 40,
      "width": 240,
      "height": 110,
      "content": "订单与售后条件校验",
      "color": "#ffcf9c"
    },
    {
      "id": "refund",
      "type": "card",
      "x": 190,
      "y": 40,
      "width": 220,
      "height": 110,
      "content": "原路退款\n同步退款结果",
      "color": "#ffe0ad"
    },
    {
      "id": "manual-review",
      "type": "card",
      "x": -160,
      "y": 250,
      "width": 260,
      "height": 110,
      "content": "人工复核\n高风险或资料不全订单",
      "color": "#ffc18e"
    },
    {
      "id": "section-title",
      "type": "text",
      "x": -520,
      "y": -55,
      "width": 540,
      "height": 54,
      "content": "标准退款流程与异常分支"
    }
  ],
  "edges": [
    {
      "id": "request-to-check",
      "from": "customer-request",
      "to": "order-check",
      "arrow": "forward",
      "note": "提交申请"
    },
    {
      "id": "check-to-refund",
      "from": "order-check",
      "to": "refund",
      "arrow": "forward",
      "note": "符合条件"
    },
    {
      "id": "check-to-review",
      "from": "order-check",
      "to": "manual-review",
      "arrow": "forward",
      "note": "高风险或信息不全"
    },
    {
      "id": "review-to-check",
      "from": "manual-review",
      "to": "order-check",
      "arrow": "forward",
      "note": "补充结论后重新校验"
    }
  ]
}
```

## AI 生成约定

### 推荐工作流

1. 先复述主题、读者、主要对象和关系方向；对象或关系仍不明确时，优先提问，不要把猜测伪装成事实。
2. 为每个需要被讨论、移动、关联或复用的概念创建一个节点；不要把整段叙述塞进单一大卡片。
3. 为每条有业务含义的关系创建一条边，并使用 `note` 写出动词、条件或机制。
4. 选择阅读方向并布局：流程/时间线从左到右，层级/归属从上到下，反馈循环用 `both` 或两条方向明确的边。
5. 校验全部 ID、端点、尺寸、颜色和图片编码，再保存为 UTF-8 JSON。

### 生成检查表

- `version` 是否为 `2`，且 `title` 是否说明这张图要回答的问题？
- `nodes[].id`、`edges[].id` 是否均唯一？每个 `from` / `to` 是否都能找到节点？
- 一条连线是否只表达一种关系？若需要解释，是否通过 `note` 清楚说明？
- `forward` 的实际含义是否确实是“`from` 指向 `to`”？反向/双向是否有意选择？
- 关键节点是否避免重叠，并按阅读方向有足够间距？
- 节点宽高是否在 `100–900` / `42–900` 范围内？
- `card`、`text` 是否只使用 `content`，`image` 是否只使用合法 `data:image/...` 的 `src`？
- 是否避免了本机绝对路径、`file://` 链接和未编码的图片路径？

### 不支持或不应假设的能力

当前格式不支持以下属性；AI 不应输出或承诺它们会生效：

- 节点的字体、字号、边框、圆角、阴影、旋转、层级（z-index）、锁定状态或自定义图标；
- 连线的自定义颜色、线型、粗细、控制点、折线路由、端口/锚点或箭头大小；
- 富文本、Markdown 渲染、HTML、外部图片 URL、音视频嵌入或附件引用；
- 画布级背景图、分组容器、泳道容器、自动布局规则或脚本逻辑。

可以用 `text` 节点模拟分区标题，用 `card` + `note` 表达角色与关系；需要真实图片时使用 `image` 的 Data URL。

## 存储、读取与导出

### 保存 `.canvas`

画布文件是 UTF-8 JSON 文本，通过已有文件接口写入 Agent 工作区。文件名只能省略后缀或使用 `.canvas`；省略时编辑器会自动补为 `.canvas`。

```bash
curl -X POST 'http://localhost:#port/api/edit?agentId=demo-agent&path=boards/refund.canvas' \
  -H 'Content-Type: application/json' \
  --data "$(jq -Rs '{content: .}' refund.canvas.json)"
```

对应 CLI：

```bash
integration api edit \
  --agentId demo-agent \
  --path boards/refund.canvas \
  --content "$(<refund.canvas.json)"
```

### 读取 `.canvas`

`/api/raw` 返回 Base64 原文；解码为 UTF-8 后按本文格式解析。编辑器遇到无法识别的 JSON 时，会以可编辑的空白画布回退，而不是让文件预览崩溃。

```bash
curl 'http://localhost:#port/api/raw?agentId=demo-agent&path=boards/refund.canvas'
```

### 导出视图

编辑器可以将“整个画布”或“当前视口”导出为 PNG 或 PDF。导出物不是 `.canvas` 数据的一部分，会保存到当前 Agent 工作区的 `canvas/` 子目录；同名导出会自动创建带时间戳的新文件，不覆盖历史导出。当前不支持 SVG 导出。

## 解析与兼容规则

为方便 AI 生产和排错，读取时的关键规则如下：

| 数据情况 | 编辑器行为 |
|---|---|
| 根内容不是 JSON 对象 | 以空白可编辑画布回退，并提示内容无法识别。 |
| `nodes` / `edges` 缺失或不是数组 | 按空数组处理。 |
| 未知节点类型 | 按 `card` 处理。 |
| 节点坐标、尺寸、视口值不是有限数字 | 使用默认值；尺寸和缩放会被限制到允许范围。 |
| 文本节点没有 `content` | 使用该类型的默认文案。 |
| 图片 `src` 不是 `data:image/` | 丢弃该图片节点。 |
| 非法 `color` | 回退为当前主题的默认节点颜色。 |
| 边缺少/使用非法 `arrow` | 按 `forward` 处理。 |
| 边引用不存在节点、自环 | 丢弃该连线。 |

这些兼容行为用于保护用户已有文件，不是生成规范的替代品。AI 输出应始终满足本文的完整格式与约束。
