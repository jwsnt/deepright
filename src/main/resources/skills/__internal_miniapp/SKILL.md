---
name: __internal_miniapp
description: 制作站内HTML迷你应用（mini app）
---

### 制作步骤
+ 使用`#workspace/app/DESIGN.md`设计风格的HTML页面展示用户想要的迷你应用（mini app）
+ 使用`#workspace/app/API.md`提供的API来驱动系统与用户的互动

### 动态服务
+ 优先通过API调用，而不是另外启动服务

### 动画框架
+ 动画框架选择：GSAP > Anime.js
+ 3D首选：Three.js

### 页面要求
+ 如果无需参考页面设计，保持UI紧凑布局，如果需要参考页面设计，需要100%复刻
+ 自适应高度和宽度，支持在iFrame打开
