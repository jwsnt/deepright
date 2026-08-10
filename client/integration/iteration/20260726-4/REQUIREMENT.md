### 第一性原则
+ 仅可以新增/更新/删除integration（../..）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../../../DESIGN.md
+ 本模块设计文档：../../DESIGN.md

### 迭代要求
+ Integration介绍：../../REQUIREMENT.md
+ Integration手册：../../USER_GUIDE.md
+ 不能破坏现有设计和功能

### 同步代码
+ ../../../integration/REQUIREMENT.md
+ 所以设计/编译都需要遵守integration的二进制和CLI收口原则

### 需求介绍
+ 为 Site 的音频裁剪提供受限的“另存为新文件”能力，不新增音频转码或裁剪服务接口。页面生成的 WAV 二进制通过 `POST /api/agent/audio?agentId=<agentId>&path=audios/<filename>.wav` 持久化。
+ 保存目标必须固定限制在请求 `agentId` 工作目录的 `audios/` 目录；目录缺失时安全创建。只接受 `audios/<文件名>.wav` 相对路径；绝对路径、`~`、`..` 逃逸、其它目录、目录写入、跨 Agent 路径和符号链接逃逸均必须拒绝。
+ 裁剪文件名由页面显示输入框并默认生成裁剪标识和时间戳，服务端只接受 `audios/` 下仅含文件名的 `.wav` 路径；服务端返回 `status`、`agentId`、请求 `path` 和新文件绝对路径 `savedAs`，供页面复制到系统剪贴板；同名冲突不得覆盖、删除、重命名或修改裁剪源音频。
+ 音频裁剪保存的 WAV 按二进制内容处理；服务端不得依赖 FFmpeg、不得对客户端提交的音频再次转码，也不得新增外部依赖。保存失败应返回 JSON 错误，供页面保留裁剪界面并提示用户。

### 编写代码
+ 复用受限 Agent 工作目录解析、`audios` 路径校验与创建、二进制 Base64 解码和 WAV 格式校验逻辑；不得放宽 `/api/edit` 的现有文件访问范围。
+ 为 WAV 的二进制另存为补充回归测试，覆盖缺失 `audios` 自动创建、成功生成新文件、源文件保持不变、时间戳新文件名、返回 `savedAs`，以及非法路径被拒绝。
+ 最小范围更新，不新增外部依赖。

### 撰写手册
+ 更新 `../../USER_GUIDE.md` 及本迭代目录 `USER_GUIDE.md`，说明复用的 `/api/edit?saveAsNew=true` 二进制另存为语义、工作目录边界和源音频不变规则。

### 其他要求
+ `REQUIREMENT.md` 仅描述需求，不记录实现过程。
