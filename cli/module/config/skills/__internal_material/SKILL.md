---
name: __internal_material
description: 视频帧快照、位置和轨迹提取，图片切分、网格坐标提取
---

### 视频帧快照、位置和轨迹
+ 直接返回文本内容"http://localhost:8080/mapping/#agentId/video_frames.html"

### 图片网格坐标提取
+ 直接返回文本内容"http://localhost:8080/mapping/#agentId/image_slices.html?file=$图片绝对路径"
+ 参数file为可选，指定后自动加载

### 注意事项
+ 不需要使用浏览器打开，客户端会自动识别