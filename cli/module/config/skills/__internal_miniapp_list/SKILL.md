---
name: __internal_miniapp_list
description: 视频调色、帧位置或轨迹提取，图片网格切分坐标提取
---

### 视频帧位置和轨迹提取
+ 直接返回文本内容"#origin/mapping/#agentId/video_frames.html"

### 视频调色
+ 直接返回文本内容"#origin/mapping/#agentId/video_color.html"

### 图片网格切分坐标提取
+ 直接返回文本内容"#origin/mapping/#agentId/image_slices.html?file=$图片绝对路径"
+ 参数file为可选，指定后自动加载

### 注意事项
+ 不需要使用浏览器打开，客户端会自动识别
+ 打开、启动均值直接返回文本内容